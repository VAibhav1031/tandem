package downloader

import (
	"context"
	"fmt"
	"log/slog"
	"math"
	"net/http"
	"os"
	"strconv"
	"time"
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

type Ranges struct {
	Start          int64 `json:"start"`
	CurrentOffsets int64 `json:"currentOffset"`
	ExpectedLimit  int64 `json:"expectedLimit"`
}
type State_File_Format struct {
	Con        int8     `json:"con"`
	Url        string   `json:"url"`
	LastRanges []Ranges `json:"lastRanges"`
	Filepath   string   `json:"filepath"`
}

// hgere it will come
type concurrentFlow struct {
	upd_chann  chan ResultUpdate
	ctx        context.Context
	client     http.Client
	headers    *Responseheaders
	total_size int64
	resumeStf  *State_File_Format
	stf        *State_File_Format
	isReady    bool
}

type ResultUpdate struct {
	Part_Id    int
	Total_Size int64
	CurrOffset int64
	Start      int64
}

type StateFile struct {
	Resume_stf   *State_File_Format
	Stf          *State_File_Format
	UpdateResult chan ResultUpdate
}

func update(ctx context.Context, results chan ResultUpdate, stf []Ranges) {
	con := len(stf)
	latestOffset := make([]int64, con)
	starts := make([]int64, con)
	for i, r := range stf {
		latestOffset[i] = r.CurrentOffsets
		starts[i] = r.Start
	}
	for msg := range results {
		// fmt.Println("DEBUG: Got update from Part", msg.Part_Id, "at offset", msg.CurrOffset)

		latestOffset[msg.Part_Id] = msg.CurrOffset
		var current_total int64
		for i := 0; i < con; i++ {
			current_total += latestOffset[i] - starts[i]
		}
		percentage := (float64(current_total) / float64(msg.Total_Size)) * 100

		fmt.Printf("\r\033[KCurrent Percentage : %.2f%%", percentage)
		os.Stdout.Sync()
	}
	// fmt.Println()
}

// Current Requirement for this to work nicely and do the task eassily for us
// Managing the incoming request and based on that  pass the request based on the availability of concurrent approach and all shit
func (d *DownloadInfo) Resolve(ctx context.Context, f_stf *StateFile) {

	ht := NewDualTransport()
	var chain http.RoundTripper = ht
	chain = &UTLSTransport{Next: ht}

	// chain = &LocalCookieTransport{Next: chain}
	// chain = &SolverTransport{Next: chain}
	client := &http.Client{Transport: chain, Timeout: 10 * time.Minute}

	req, err := http.NewRequestWithContext(ctx, "HEAD", d.Rs.Link, nil)
	if err != nil {
		slog.Error("[Downloader-Maximizer]: Error Ocurred <http Client GET req> ", slog.Any("error", err))
	}

	resp, err := client.Do(req)
	if err != nil {
		slog.Error("[Downloader-Maximizer]: Network Failure!!", "error: ", err)
		fmt.Println("Connection-Failure..")
		return
	}

	defer resp.Body.Close()

	if resp.StatusCode != 200 {

		slog.Error("[Downloader-Maximizer-ERROR]:", slog.Any("status-code", resp.StatusCode), slog.Any("status", resp.Status), slog.Any("url", resp.Request.URL.String()))
		return
	}

	upd_chan := make(chan ResultUpdate, 1000000)
	req_head := ServerResponse(resp.Header)
	// we need to pass the  variable  somewhere to there so that it happen here easily without  unecessary problem in
	if req_head.ConcurrentCheck() {
		// req_head is good for getting the concurrent_check
		// we need to check the stf is populated or not , if it is tghen  we ould go  and tghen there is one more thing
		// if stf.

		var conFlow concurrentFlow
		totalSize, _ := strconv.Atoi(req_head.Content_length)
		slog.Info(fmt.Sprintf("totalSize of the file %v and in the gb %v", totalSize, (float64(totalSize) / float64(1024*1024*1024))))

		batchSize := int64(math.Ceil(float64(totalSize) / float64(d.Rs.Con_n)))
		slog.Info("floated value ", ":", math.Ceil(float64(totalSize)/float64(d.Rs.Con_n)))
		start, limit := int64(0), batchSize
		f_stf.Stf.LastRanges[0] = Ranges{CurrentOffsets: start, ExpectedLimit: limit}

		if f_stf.Resume_stf.Url == "" {
			for i := 1; i < int(d.Rs.Con_n); i++ {

				start = limit
				limit = start + batchSize

				if limit%int64(totalSize) != limit {
					limit = limit - (limit % int64(totalSize))
				}
				f_stf.Stf.LastRanges[i] = Ranges{
					Start:          start,
					CurrentOffsets: start,
					ExpectedLimit:  limit,
				}
			}
			go update(ctx, upd_chan, f_stf.Stf.LastRanges)

			conFlow.client = *client
			conFlow.stf = f_stf.Stf
			conFlow.headers = req_head
			conFlow.ctx = ctx
			conFlow.total_size = int64(totalSize)
			conFlow.upd_chann = upd_chan

			slog.Info("[Downloader-Maximizer]: We are gonaa Fresh  Download (Concurrent), OLD n-concurrent will be used")
		} else {

			for i := 0; i < int(d.Rs.Con_n); i++ {

				// we should have same LastRange as of the Resume so that if any network error or goroutine doesnt schedule and
				// ctr+c happen rather than getting 0 (when we didnt use currentOffset of Resume STF.LastRanges )  we would be fallback
				// to the previously written byte state and may continue on antoher resume
				f_stf.Stf.LastRanges[i] = Ranges{
					Start:          f_stf.Resume_stf.LastRanges[i].Start,
					CurrentOffsets: f_stf.Resume_stf.LastRanges[i].CurrentOffsets,
					ExpectedLimit:  f_stf.Resume_stf.LastRanges[i].ExpectedLimit,
				}
			}

			go update(ctx, upd_chan, f_stf.Resume_stf.LastRanges)
			conFlow.client = *client
			conFlow.resumeStf = f_stf.Resume_stf
			conFlow.stf = f_stf.Stf
			conFlow.ctx = ctx
			conFlow.total_size = int64(totalSize)
			conFlow.upd_chann = upd_chan
			conFlow.isReady = true

			slog.Info("[Downloader-Maximizer]: We  are gonna Resume Download (Concurrent), OLD n-concurrent will be used")
		}
		d.ConcurrentDownloader(conFlow)
	} else {

		slog.Info("[Downloader-Maximizer]: We are gonna Fresh Download everytime (No concurrent)")

		d.DownloadNormal(req_head, client)
	}

}
