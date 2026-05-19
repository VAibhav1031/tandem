package downloader

import (

	// "fmt"
	"golang.org/x/net/http2"
	"log"
	"net/http"
	"time"
)

func (r *Responseheaders) ConcurrentCheck() bool {

	accept_range := r.accept_ranges

	if accept_range == "" {
		return false
	} else {
		return true
	}
}

func (d *DownloadInfo) maxim() {

	h2t := &http2.Transport{
		DialTLSContext: dialUTLS,
		// AllowHTTP: false, // keep false for real use
	}

	// DefaultTransport is also RoundTripper casuse it has the RoundTrip method with it
	// so now in this condition it was like we have to  have to remove that for teh internet request andd add the new one her eit is the h2t
	// which is also a Transport but not the DefaultTransport one , but it satisfy condition ,plus with our custom TLS thing ,  and the uTLSuTLSTransport struct will create the request to the h2t with those uTuTLSTransport RoundTrip request added Headers
	var chain http.RoundTripper = h2t

	chain = &uTLSTransport{Next: h2t}
	chain = &LocalCookieTransport{Next: chain}
	chain = &SolverTransport{Next: chain}

	client := &http.Client{Transport: chain, Timeout: 30 * time.Second}

	req, err := http.NewRequest("GET", d.Rs.Link, nil)
	if err != nil {
		log.Printf("[Downloader] Error Ocurred <http Client GET req> : %v\n", err)
	}
	resp, err := client.Do(req)
	if err != nil {
		log.Println("[Downloader] Network error")
	}

	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		log.Println("[Downloader] Failure :", resp.StatusCode)
		return
	}

	req_head := ServerResponse(resp.Header)
	if req_head.ConcurrentCheck() {
		d.ConcurrentDownloader()
	} else {
		d.DownloadNormal()
	}

}
