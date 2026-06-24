package downloader

import (

	// "fmt"
	"log"
	"net/http"
	"time"
)

func (r *Responseheaders) ConcurrentCheck() bool {

	accept_range := r.accept_ranges

	if accept_range == "bytes" {
		return true
	} else {
		return false
	}
}

// Current Requirement for this to work nicely and do the task eassily for us
// Managing the incoming request and based on that  pass the request based on the availability of concurrent approach and all shit
func (d *DownloadInfo) Maxim() {

	ht := NewDualTransport()
	var chain http.RoundTripper = ht
	chain = &uTLSTransport{Next: ht}

	// chain = &LocalCookieTransport{Next: chain}
	// chain = &SolverTransport{Next: chain}
	client := &http.Client{Transport: chain, Timeout: 10 * time.Minute}

	req, err := http.NewRequest("HEAD", d.Rs.Link, nil)
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
	// we need to pass the  variable  somewhere to there so that it happen here easily without  unecessary problem in
	if req_head.ConcurrentCheck() {
		d.ConcurrentDownloader(req_head, client)
	} else {
		d.DownloadNormal(req_head, client)
	}

}
