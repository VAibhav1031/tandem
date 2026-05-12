package downloader

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	utls "github.com/refraction-networking/utls"
	"net"
	"net/http"
	"time"
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
		"AppleWebKit/537.36 (KHTML, like Gecko) Chrome/124.0.0.0 Safari/537.36")
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,"+
		"image/avif,image/webp,*/*;q=0.8")
	req.Header.Set("Accept-Language", "en-US,en;q=0.9")
	req.Header.Set("Accept-Encoding", "gzip, deflate, br ,zstd")

	return t.Next.RoundTrip(req)
}

type LocalCookieTransport struct {
	Next http.RoundTripper
}

func (t *LocalCookieTransport) RoundTrip(req *http.Request) (*http.Response, error) {

	if GlobalCookieCache != "" {

		req.Header.Set("Cookie", "cf_clearance="+GlobalCookieCache)
	}

	return t.Next.RoundTrip(req)
}

func dialUTLS(ctx context.Context, network, addr string, _ *utls.Config) (net.Conn, error) {
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

	uConn := utls.UClient(tcpConn, &utls.Config{
		ServerName: host,
		// NextProtos is set automatically by HelloChrome_Auto to include
		// "h2" and "http/1.1", so ALPN negotiation works correctly.
	}, utls.HelloChrome_Auto)

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

	return uConn, nil
}

func runFlareSolver(targetURL string) (string, string, error) {

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
