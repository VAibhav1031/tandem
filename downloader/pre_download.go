package downloader

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"

	"crypto/tls"
	"net"
	"net/http"
	"net/url"
	"time"

	utls "github.com/refraction-networking/utls"
	"golang.org/x/net/publicsuffix"

	// "golang.org/x/net/http2"
	"github.com/VAibhav1031/tandem/cookiesManager"
)

// Payload we send to FlareSolverr
type FlareSolverrRequest struct {
	Cmd        string `json:"cmd"`        // Always "request.get"
	URL        string `json:"url"`        // The protected site URL
	MaxTimeout int    `json:"maxTimeout"` // Max time to wait for JS challenge (in ms)
}

// Structs to read the response back
type FlareSolverrResponse struct {
	Status   string `json:"status"` // Should be "ok"
	Solution struct {
		UserAgent string `json:"userAgent"` // The exact UA Chrome used to solve it
		Cookies   []struct {
			Name  string `json:"name"`
			Value string `json:"value"`
		} `json:"cookies"`
	} `json:"solution"`
}

var GlobalCookieCache = cookies.CookieSolver()

type uTLSTransport struct {
	Next http.RoundTripper
}

func (t *uTLSTransport) RoundTrip(req *http.Request) (*http.Response, error) {

	// Match headers to the HelloChrome_Auto fingerprint version.
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) "+
		"AppleWebKit/537.36 (KHTML, like Gecko) Chrome/148.0.0.0 Safari/537.36")
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,"+
		"image/avif,image/webp,*/*;q=0.8")
	req.Header.Set("Accept-Language", "en-US,en;q=0.9")
	req.Header.Set("Accept-Encoding", "gzip, deflate, br ,zstd")

	if req.Header["Cookie"][0] != "" {
		log.Println("[Tier 1] Cookie is set ..")

	}
	return t.Next.RoundTrip(req)
}

func getBaseDomain(rawURL string) string {

	u, err := url.Parse(rawURL)
	if err != nil {
		return ""
	}

	base, err := publicsuffix.EffectiveTLDPlusOne(u.Hostname())
	if err != nil {
		u.Hostname()
	}
	return base
}

type LocalCookieTransport struct {
	Next       http.RoundTripper
	HomeDomain string
}

func (t *LocalCookieTransport) RoundTrip(req *http.Request) (*http.Response, error) {

	if t.HomeDomain == "" {
		t.HomeDomain = getBaseDomain(req.URL.String())
	}
	currentBase := getBaseDomain(req.URL.String())
	log.Println("[Tier 2] Base domain ", t.HomeDomain)
	if currentBase == t.HomeDomain {
		if GlobalCookieCache != "" {
			log.Println("[Tier 2] Currently they are same and we got the GlobalCookieCache,", GlobalCookieCache)

			req.Header.Set("Cookie", "cf_clearance="+GlobalCookieCache)
		}
	} else {
		req.Header.Del("Cookie")
		log.Printf("[Tier 2] Cross-domain Jump: %s -> %s. Cookies Stripped", t.HomeDomain, currentBase)
	}
	return t.Next.RoundTrip(req)
}

func dialUTLS(ctx context.Context, network, addr string, _ *tls.Config) (net.Conn, error) {
	dialer := &net.Dialer{Timeout: 10 * time.Second}

	tcpConn, err := dialer.DialContext(ctx, network, addr)
	if err != nil {
		return nil, fmt.Errorf("tcp dial: %w", err)
	}

	host, _, err := net.SplitHostPort(addr)
	if err != nil {
		tcpConn.Close()
		return nil, fmt.Errorf("split host/port: %w", err)
	}
	log.Printf("[Tier1]-[spook_TLS]: HOST %v", host)

	uConn := utls.UClient(tcpConn, &utls.Config{
		ServerName: host,
		// NextProtos is set automatically by HelloChrome_Auto to include
		// "h2" and "http/1.1", so ALPN negotiation works correctly.
		NextProtos: []string{"h2", "http/1.1"}, // ALPN
	}, utls.HelloRandomized)

	if err := uConn.HandshakeContext(ctx); err != nil {
		uConn.Close()
		return nil, fmt.Errorf("utls handshake: %w", err)
	}

	// Safety check: if the server downgraded us to HTTP/1.1, bail here
	// so the caller knows not to use this conn for H2 framing.
	if uConn.ConnectionState().NegotiatedProtocol != "h2" {
		uConn.Close()
		return nil, fmt.Errorf("server did not negotiate h2 (got %q)",
			uConn.ConnectionState().NegotiatedProtocol)
	}

	log.Println("[Tier 1]-[spook_TLS] dialTLS worked as intended to... ")
	return uConn, nil
}

type SolverTransport struct {
	Next http.RoundTripper
}

func (t *SolverTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	// 1. Send the request down the chain
	log.Println("[Tier 3] Pass the Request Forward")
	resp, err := t.Next.RoundTrip(req)
	if err != nil {
		return nil, err
	}

	// 2. RIPPLE BACK check: Did Cloudflare block us with a challenge?
	// (Your exact debug header was "Cf-Mitigated: [challenge]")
	if resp.StatusCode == 403 && resp.Header.Get("Cf-Mitigated") == "challenge" {
		fmt.Println("======================First-Two-Level-FAILED=======================")
		log.Println("[Tier 3] ALERT: Cloudflare Challenge Detected (Cf-Mitigated: [challenge])!")
		log.Println("[Tier 3] Local cookie failed or was expired.")

		// ALWAYS close the rejected response body to prevent memory leaks
		resp.Body.Close()

		// 3. Trigger Tier 3 solver logic (FlareSolverr or prompt user)
		log.Println("[Tier 3] Spinning up solver sequence...")

		newCookie, solvedUA, err := runFlareSolverr(req.URL.String())
		if err == nil {
			// 4. Update our local SQLite/disk cache so we have it for next time
			log.Println("[Tier 3] New Cookie: ", newCookie)
			GlobalCookieCache = newCookie
			log.Println("[Tier 3] Fresh cookie cached successfully.")

			req.Header.Set("User-Agent", solvedUA)

			// 5. RETRY: Send the request down the chain a second time
			log.Println("[Tier 3] Retrying request with the fresh cookie...\n")
			return t.Next.RoundTrip(req)
		}
		log.Printf("[Tier 3] Err : %v", err)
	}
	// If no challenge, just let the successful response ripple back normally
	return resp, nil
}
func runFlareSolverr(targetURL string) (string, string, error) {

	flareURL := "http://localhost:8191/v1"

	payload := FlareSolverrRequest{

		Cmd:        "request.get",
		URL:        targetURL,
		MaxTimeout: 60000,
	}

	jsonPayload, err := json.Marshal(payload)

	if err != nil {
		return "", "", err
	}

	client := &http.Client{Timeout: 70 * time.Second}

	resp, err := client.Post(flareURL, "application/json", bytes.NewBuffer(jsonPayload))

	if err != nil {
		return "", "", fmt.Errorf("Failed to contact FlareSolverr  :%w", err)
	}

	defer resp.Body.Close()

	var flareResp FlareSolverrResponse
	if err := json.NewDecoder(resp.Body).Decode(&flareResp); err != nil {
		return "", "", err
	}

	if flareResp.Status != "ok" {
		return "", "", fmt.Errorf("flareSolverr failed to solve Challenge")

	}
	var cfClearance string
	for _, cookie := range flareResp.Solution.Cookies {
		if cookie.Name == "cf_clearance" {
			cfClearance = cookie.Value
			break
		}
	}

	if cfClearance == "" {
		return "", "", fmt.Errorf("cf_clearance cookie not found in solution")
	}

	// Return both the golden ticket cookie AND the user agent it used
	return cfClearance, flareResp.Solution.UserAgent, nil
}
