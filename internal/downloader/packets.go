package downloader

import (
	"errors"
	"fmt"
	"io"
	"log/slog"
	"math"
	"net/http"
	"net/url"
	"os"
	"path"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"
)

const buffer_length int = 32 * 1024

type DownloadInfo struct {
	Rs *RequestServer
	cn conCurrentDet
}
type RequestServer struct {
	Link              string
	Con_n             int8 // we are thinking nobody gonna give more than this
	FileLocation      string
	StateFileLocation string
}

type Responseheaders struct {
	Content_length     string
	Content_type       string
	Content_deposition string
	Accept_ranges      string
	// headers            http.Header
}

var mimeToExt = map[string]string{
	"text/plain":                "txt",
	"text/html":                 "html",
	"text/csv":                  "csv",
	"application/pdf":           "pdf",
	"image/jpeg":                "jpg",
	"image/png":                 "png",
	"application/zip":           "zip",
	"application/octect-stream": "bin",
}

// we should strore the map only of the header nothing else
func NewServerLink(link string, n int8, location string, state string) *RequestServer {
	return &RequestServer{
		Link:              link,
		Con_n:             n,
		FileLocation:      location,
		StateFileLocation: state,
	}
}

func ServerResponse(headers http.Header) *Responseheaders {

	return &Responseheaders{Content_length: headers.Get("Content-Length"), Content_type: headers.Get("Content-Type"), Accept_ranges: headers.Get("Accept-Ranges")}

}

func DownloadWorker(request *RequestServer) *DownloadInfo {
	return &DownloadInfo{Rs: request}
}

func getExtensionFromUrl(rawUrl string) string {

	u, err := url.Parse(rawUrl)

	if err != nil {
		return ""
	}

	ext := path.Ext(u.Path)
	return ext
}

func (r *Responseheaders) getFileInfo(url string) (string, string) {

	if r.Content_deposition != "" {
		file_name := strings.Split(r.Content_deposition, "filename=")[1]
		file_type := strings.Split(file_name, ".")[1]

		fmt.Println("THIS IS IT")
		return file_name, file_type

	}

	if r.Content_type != "" {
		// file_type := strings.Split(r.content_type, "/")[1]
		file_type := mimeToExt[r.Content_type]

		fmt.Println(r.Content_type)
		return "", file_type
	}

	return "", getExtensionFromUrl(url)

	//
	//1 Deposition
	//2 Content Type
	//3 URL -Check
	//4 sniffing (initial packets)
	//5 fallback (default type .txt .bin or just default with no extensiongiven)
}

func (d *DownloadInfo) DownloadNormal(req_head *Responseheaders, client *http.Client) {

	req, err := http.NewRequest("GET", d.Rs.Link, nil)
	if err != nil {
		slog.Error("[Downloader] Error Ocurred <http Client GET req> : %v\n", err)
	}
	resp, err := client.Do(req)
	if err != nil {
		slog.Error("[Downloader] Network error", err)
	}

	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		slog.Error("[Downloader] Failure :", resp.StatusCode)
		return
	}

	out, err := os.Create(d.Rs.FileLocation)
	if err != nil {
		fmt.Printf("[Downloader]: Error occurred <File creation>: %v", err)
	}

	buffer_read := make([]byte, buffer_length) //buffer_lenght --> 32kb length
	contentLength := resp.ContentLength        // This is an int64
	var downloaded int64 = 0

	// if preview != nil {
	//
	// 	out.Write(preview)
	// }
	for {
		// read from network into buffer_read (network Stream buffer
		n, err := resp.Body.Read(buffer_read)

		if n > 0 {
			// Writing the chunk to the disk (chunk by chunk)
			out.Write(buffer_read[:n])

			// Update the counter
			downloaded += int64(n)

			if contentLength > 0 {
				percent := (float64(downloaded) / float64(contentLength)) * 100
				fmt.Printf("\rProgress: %.2f%% \n", percent)
			}
		}

		if err == io.EOF {
			break // Data finished!
		}
	}
}

type conCurrentDet struct {
	n      int
	buffer []byte
	passed bool
	mw     sync.Mutex
}

// current situation we facing is we need to know the total  size of the thing , without that we are unable to initate the concurrent download thing , Response Header is kind of the solution to this i would says oo , to get the total  byte thing i would say soo , currently we are not thinking about the  pre download part here , link would be directly okay

// each go-routine will have the  retyr logic with limit , like if the request fail , and if the copy fails something ... like that we have to have something so that we can have great sucess or increase sucess rate ..

// there should be loop if thinsg is done in go then loop would break , else increas but the counter will also increase and with reached limit it will break and with check of counter value eq tro the the total limit will return the error andallshit

// mostly it would something like  http request and the copy  there is some problem to occur
const globalTryLimit int = 4

func (d *DownloadInfo) ConcurrentDownloader(ct concurrentFlow) {

	// we have to check whether the
	// if we have the resume thing then we have to go with edifferent start and end
	var start int64
	var limit int64
	var total_size int
	var batch_size int64
	var fd int
	var file *os.File

	// concurrent n check
	if d.cn.n == 0 {
		d.cn.n = 4
	}

	if !ct.isReady {
		total_size, _ = strconv.Atoi(ct.headers.Content_length)
		slog.Info("total_size of the file", total_size, "and in the gb", (float64(total_size) / float64(1024*1024*1024)))

		batch_size := int64(math.Ceil(float64(total_size) / float64(d.cn.n))) // size need to be clearl y round so that slicing doesnt give problems
		slog.Info("floated value : %v", math.Ceil(float64(total_size)/float64(d.cn.n)))
		d.cn.buffer = make([]byte, total_size) // make changes to the buffer  with condition and open the last populated version
		start, limit = int64(0), batch_size

	}
	_, err := os.Stat(d.Rs.FileLocation)
	if err != nil { // && check for the fallocate cause err!= nill  means there i

		// file already exist no problem , if that exist and thenm we havce to populate , buyt that thing is nto required noq mann , we know that and we will use the offset adn all shit  to write the thing nothing else  is needed now

	} else if errors.Is(err, os.ErrNotExist) {

		file, err = os.OpenFile(d.Rs.FileLocation, os.O_RDWR|os.O_CREATE, 0666)
		if err != nil {
			panic(err)
		}
		defer file.Close()

		var size int64 = 10 * 1024 * 1024
		fd = int(file.Fd())

		err = syscall.Fallocate(fd, 0, 0, size)
		if err != nil {
			slog.Error("Fallocate failed: %v\n", err)
			return
		}
	}

	// pre passed thing
	d.cn.passed = true

	wg := &sync.WaitGroup{}
	for i := 0; i < d.cn.n; i++ {
		// current outer variables limit, start, is the passed one , the outer is the globalLimit one
		// this need to be universal means  just the start and limit
		// what condition needed to go
		// my idea just smpple check here  and we will just use the array of those start and end

		if ct.isReady {
			r_det := ct.stf.LastRanges[i]
			start, limit = r_det.CurrentOffsets, r_det.ExpectedLimit

		}
		slog.Info("GOROUTINE %d-->Start: %d, limit: %d", i, start, limit)

		wg.Add(1)
		go func(chunkStart int64, chunkLimit int64) {
			defer wg.Done()

			var currentOffset = chunkStart
			expectedLimit := chunkLimit

			current := 0
			for {
				select {

				case <-ct.ctx.Done():

					// here you have to give the current offset to the

					ct.stf.LastRanges = append(ct.stf.LastRanges, ranges{CurrentOffsets: currentOffset, ExpectedLimit: expectedLimit})
					// ct.stf.Filepath = d.Rs.FileLocation ///
					return
				default:
					remainingBytes := expectedLimit - currentOffset
					if remainingBytes < 0 {
						return
					}

					if current == globalTryLimit {
						d.cn.mw.Lock()
						d.cn.passed = false
						d.cn.mw.Unlock()
						slog.Error("All Limit Crossed!! Exitting Goroutine..")
						return
					}

					req, err := http.NewRequest("GET", d.Rs.Link, nil) // new request , default http Transport with TLS , https support based on that
					if err != nil {
						slog.Error("[Concurrent-ERROR]: ", err)
					}
					req.Header.Set("Range", fmt.Sprintf("bytes=%d-%d", currentOffset, expectedLimit-1))
					resp, err := ct.client.Do(req)
					if err != nil {
						slog.Error("[Concurrent-Error]: Connection Failed %v, ", err)
						// exit the goroutine thing...:_)
						current++
						// resp.Body.Close()
						time.Sleep(1 * time.Second)
						continue
					}

					if resp.StatusCode != http.StatusPartialContent && resp.StatusCode != http.StatusOK {
						slog.Error("[Concurrent-Error]: Unexpected status code : %d", resp.StatusCode)
						resp.Body.Close()
						current++
						continue
					}

					// Since each slice of the buffere is not overlapping , so  there is no need to put the lock over the buffer and we cango easily and it is the design which help it move

					// going for the block read , cause io.ReadFull() all or nothing , here we have to go in progressive way where if any error ocurr we can  store the till the read offset byte , not losing whole and retrying again
					bufferBlock := make([]byte, 32*1024)
					for {
						nr, readErr := resp.Body.Read(bufferBlock)
						if nr > 0 {
							//  here we will be using the writeAt thing with the offset provided
							_, err := file.WriteAt(bufferBlock, currentOffset)

							if err != nil {
								d.cn.passed = false
								slog.Error("[Concurrent-Error]: WriteAt Error %v", err)
								return
							}
							// we have the read  body which is like block read thing  which we dont need to have the probelm and now we have use that to write the  thing , cause the WriteAt usually take the buffer byte and currentOFfset from where we have to write andall shit nothing else

							// now the thing is how to update teh currentOffset it should be based on the the number of byte it has written

							// copy(d.cn.buffer[currentOffset:currentOffset+int64(nr)], bufferBlock)
							currentOffset += int64(nr)
						}

						if readErr != nil {
							if readErr == io.EOF {
								break // Read Completely successfully
							}
							slog.Error("[Concurrent-Error]:[Network-Interrupted]: Saved  Progress")
							// few thoughts : could have be the continue andallshit , but yeah we are under the another loop and we have only one was is to exit then close the resp streaming , then  yeah your thinking error and continue thing , that is nice , but we are already saved by the anmother timeout per goroutine client

							// could be too harsh if i add the 'current' incrementor here
							current++
							break
						}

					}

					resp.Body.Close()

					if currentOffset >= expectedLimit {
						break
						// read whole segment not needed anymore
					}
					// // read to the correct section of the buffer
					// destBuffer := d.cn.buffer[currentOffset:expectedLimit]
					// n, err := io.ReadFull(resp.Body, destBuffer)
					// if n < 0 || err != nil {
					// 	slog.Error("[Concurrent-Error]: BOOOM!!, start %d: limit %d | Read-up ERR-> %v", start, limit, err)
					// 	current++
					// 	resp.Body.Close()
					// 	// time.Sleep(1 * time.Second) // could be network lag or something we get interr
					// 	continue
					// }
					//
					// break
				}
			}

		}(start, limit)
		if !ct.isReady {
			start = limit
			limit = start + batch_size

			fmt.Println(total_size)
			if limit%int64(total_size) != limit {
				limit = limit - (limit % int64(total_size))
			}
		}
		// we need to wait till all the goroutines are complete  then proceed with lower ,  if done then based on the passed bool value we proceed like if it went well or not if not then we will just skip that and all shit
		slog.Info("All goroutine are fired!!")
	}

	wg.Wait()

	if !d.cn.passed {
		slog.Error("[Concurrent-Error]: Concurrent Process Failed !!")
		return
	}

	// out, err := os.Create(d.Rs.FileLocation)
	// if err != nil {
	// 	slog.Error("[Concurrent-Downloader]: Error occurred <File creation>: %v", err)
	// }
	//
	// out.Write(d.cn.buffer)

}
