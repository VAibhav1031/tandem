package downloader

import (
	"context"
	"log"
	"net/http"
	"time"
	// "github.com/VAibhav1031/tandem/cmd"
)

func (r *Responseheaders) ConcurrentCheck() bool {

	accept_range := r.Accept_ranges

	if accept_range == "bytes" {
		return true
	} else {
		return false
	}

}

// state file json format
// {url:<link>,currentOffset:..,expectedLimit:..,filename:for_which_file}

type ranges struct {
	CurrentOffsets int64 `json:"currentOffset"`
	ExpectedLimit  int64 `json:"expectedLimit"`
}
type State_File_Format struct {
	Url        string   `json:"url"`
	LastRanges []ranges `json:"lastRanges"`
	Filepath   string   `json:"filepath"`
}

// hgere it will come
type concurrentFlow struct {
	ctx     context.Context
	client  http.Client
	headers *Responseheaders
	stf     State_File_Format
	// isReady bool
}

// Current Requirement for this to work nicely and do the task eassily for us
// Managing the incoming request and based on that  pass the request based on the availability of concurrent approach and all shit
func (d *DownloadInfo) Maxim(ctx context.Context, stf State_File_Format) {

	ht := NewDualTransport()
	var chain http.RoundTripper = ht
	chain = &UTLSTransport{Next: ht}

	// chain = &LocalCookieTransport{Next: chain}
	// chain = &SolverTransport{Next: chain}
	client := &http.Client{Transport: chain, Timeout: 10 * time.Minute}

	req, err := http.NewRequest("HEAD", d.Rs.Link, nil)
	if err != nil {
		log.Printf("[Downloader-Maximizer]: Error Ocurred <http Client GET req> : %v\n", err)
	}

	resp, err := client.Do(req)
	if err != nil {
		log.Println("[Downloader-Maximizer]: Network error")
	}

	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		log.Printf("[Concurrent-Error]:  %v , Code-> %d", err, resp.StatusCode)
		return
	}
	req_head := ServerResponse(resp.Header)
	// we need to pass the  variable  somewhere to there so that it happen here easily without  unecessary problem in

	if req_head.ConcurrentCheck() {
		// req_head is good for getting the concurrent_check
		// we need to check the stf is populated or not , if it is tghen  we ould go  and tghen there is one more thing
		// if stf.
		var conFlow concurrentFlow
		if stf.Url == "" {

			conFlow.client = *client
			conFlow.headers = req_head
			conFlow.ctx = ctx
		} else {
			conFlow.client = *client
			conFlow.stf = stf
			conFlow.ctx = ctx

		}
		d.ConcurrentDownloader(conFlow)
	} else {
		d.DownloadNormal(req_head, client)
	}

	// resumption headers  we need something important for the  concurrent to continuye ,  i think for the resumption thing it has to check for that and then use  that global struct values and all shit for the work
}
