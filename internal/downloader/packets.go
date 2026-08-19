package downloader

import (
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
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

func (d *DownloadInfo) DownloadNormal(nf normalFlow) {

	req, err := http.NewRequest("GET", d.Rs.Link, nil)
	if err != nil {
		slog.Error("[Downloader] Error Ocurred <http Client GET req>", slog.Any("error", err))
	}
	resp, err := nf.client.Do(req)
	if err != nil {
		slog.Error("[Downloader] Network error", slog.Any("error", err))
		fmt.Println("Connection-Failure..")
		return
	}

	// for closing of the channel from the sender side
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		slog.Error("[Downloader] Network/HTTP-Request Failure ", slog.Any("status-code", resp.StatusCode), slog.Any("status", resp.Status), slog.Any("url", resp.Request.URL.String()))
		return
	}

	out, err := os.Create(d.Rs.FileLocation)
	if err != nil {
		slog.Error("[Downloader]: Error occurred <File creation>", slog.Any("error", err))
		fmt.Println("Failed Creation of File ", d.Rs.FileLocation)
		return
	}

	buffer_read := make([]byte, buffer_length)
	var currBuff int64 = 0
	var currentOffset int64 = 0

	// if preview != nil {
	//
	// 	out.Write(preview)
	// }

	defer close(nf.upd_chann)
	var passed bool = true
	for {
		select {
		case <-nf.ctx.Done():
			fmt.Println("Normal Download cannot have the Pause Feature..")
			slog.Info("[Downloader-Normal]: Cancelled , Normal Download , no Pause state management")
			return
		default:
			for {
				// read from network into buffer_read (network Stream buffer
				n, err := resp.Body.Read(buffer_read)

				if n > 0 {
					// Writing the chunk to the disk (chunk by chunk)
					out.Write(buffer_read[:n])

					// Update the counter
					currentOffset += int64(n)
					currBuff += int64(n)

					if currBuff >= (512 * 1024) {
						nf.upd_chann <- ResultUpdate{Part_Id: 0, Total_Size: nf.total_size, CurrOffset: currentOffset, Start: 1}
					}

				}

				if err != nil {
					if err == io.EOF {
						break // Data finished!
					} else {
						passed = false
						fmt.Println(err)
						break
					}
				}
			}
			resp.Body.Close()
		}
		if !passed {
			slog.Error("[Download-Normal-Error]: Download Normal Failed !!", slog.Any("error", "Network Interruption"))
			fmt.Println("")
			return
		} else {

			slog.Info("[Downloader-Normal]: Downloaded Sucessfully")
			fmt.Println("\nSucesfully Done!!")
			return
		}
	}

}

/* ***Concurrent Download Section***  */

type conCurrentDet struct {
	bufferBlock [32 * 1024]byte
	passed      bool
	mw          sync.Mutex
}

const globalTryLimit int = 4

func (d *DownloadInfo) ConcurrentDownloader(ct conCurrentFlow) {

	var fd int
	var file *os.File

	_, err := os.Stat(d.Rs.FileLocation)
	if err == nil { // && check for the fallocate cause err!= nill  means there i

		file, err = os.OpenFile(d.Rs.FileLocation, os.O_RDWR|os.O_CREATE, 0666)
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
			slog.Error("Fallocate failed: ", slog.Any("error", err))
			return
		}
	}
	defer close(ct.upd_chann)

	// ALl goroutine Checker bool,helps in  checking that all passed or not
	d.cn.passed = true

	wg := &sync.WaitGroup{}
	for i := 0; i < int(d.Rs.Con_n); i++ {

		wg.Add(1)
		go func(Part_ID int) {
			defer wg.Done()

			var start int64
			var limit int64

			if ct.isReady {
				d.cn.mw.Lock()
				r_det := ct.resumeStf.LastRanges[Part_ID]
				d.cn.mw.Unlock()
				start, limit = r_det.CurrentOffsets, r_det.ExpectedLimit
			} else {
				d.cn.mw.Lock()
				det := ct.stf.LastRanges[Part_ID]
				d.cn.mw.Unlock()
				start, limit = det.CurrentOffsets, det.ExpectedLimit
			}
			slog.Info(fmt.Sprintf("GOROUTINE : %v, -->Start: %v, limit-->  %v", i, start, limit))
			currentOffset := start
			expectedLimit := limit

			current_try_limit := 0
			for {
				select {

				case <-ct.ctx.Done():

					d.cn.mw.Lock()
					if currentOffset > ct.stf.LastRanges[Part_ID].CurrentOffsets {
						ct.stf.LastRanges[Part_ID].CurrentOffsets = currentOffset

					}
					d.cn.mw.Unlock()

					slog.Info("[Concurrent]: Canceled.., Sucessfully Paused with current Offset States!!")
					return

				default:
					localBuffer := make([]byte, len(d.cn.bufferBlock))
					max_to_read := len(localBuffer)
					limit_to_read := int64(max_to_read)
					remainingBytes := expectedLimit - currentOffset

					if remainingBytes <= 0 {
						return
					}
					if remainingBytes < limit_to_read { // to make sure we read only the required number of byte of the current range
						limit_to_read = remainingBytes
					}

					if current_try_limit == globalTryLimit {
						d.cn.mw.Lock()
						d.cn.passed = false
						d.cn.mw.Unlock()
						slog.Warn("[Concurrent-ERROR]:All Limit Crossed!! Exitting Goroutine..")
						return
					}

					req, err := http.NewRequestWithContext(ct.ctx, "GET", d.Rs.Link, nil) // new request , default http Transport with TLS , https support based on that
					if err != nil {
						slog.Error("[Concurrent-ERROR]: Request Creation failed ", slog.Any("error", err))
						current_try_limit++
						continue
					}

					req.Header.Set("Range", fmt.Sprintf("bytes=%d-%d", currentOffset, expectedLimit-1))
					resp, err := ct.client.Do(req)
					if err != nil {
						slog.Error("[Concurrent-Error]: Connection Failed ", slog.Any("error", err))
						current_try_limit++
						time.Sleep(1 * time.Second)
						continue
					}
					if resp.StatusCode != http.StatusPartialContent && resp.StatusCode != http.StatusOK {
						slog.Error("[Concurrent-Error] Network/HTTP-Request Failure ", slog.Any("status-code", resp.StatusCode), slog.Any("status", resp.Status), slog.Any("url", resp.Request.URL.String()))
						resp.Body.Close()
						current_try_limit++
						continue
					}

					// Example in Go logic
					if resp.StatusCode != 206 {
						// Treat this as an error! Do not write to disk.
						// The server is denying chunking for this request.

						slog.Error("[Concurrent-Error] Partial Content", slog.Any("status-code", resp.StatusCode), slog.Any("url", resp.Request.URL.String()))
						resp.Body.Close()
						current_try_limit++
						continue
					}

					// Since each slice of the buffere is not overlapping , so  there is no need to put the lock over the buffer and we cango easily and it is the design which help it move

					// going for the block read , cause io.ReadFull() all or nothing , here we have to go in progressive way where if any error ocurr we can  store the till the read offset byte , not losing whole and retrying again

					var currentBuff int64
					for {

						nr, readErr := resp.Body.Read(localBuffer[:limit_to_read])
						if nr > 0 {
							_, err := file.WriteAt(localBuffer[:nr], currentOffset)

							if err != nil {
								d.cn.passed = false
								resp.Body.Close()
								slog.Error("[Concurrent-Error]: WriteAt Error ", slog.Any("error", err))
								return
							}

							currentOffset += int64(nr)

							// CurrentBuff For sending the data to the update groutine
							currentBuff += int64(nr)
							if currentBuff >= (512 * 1024) { // send only when 512 kb HAS  been read
								ct.upd_chann <- ResultUpdate{Part_Id: Part_ID, Total_Size: ct.total_size, CurrOffset: currentOffset, Start: start}
								currentBuff = 0
							}
						}

						if readErr != nil {
							if readErr == io.EOF {
								break // Read Completely successfully
							}
							slog.Error("[Concurrent-Error]:[Network-Interrupted]: Saved  Progress")
							// could be too harsh if i add the 'current' incrementor here
							current_try_limit++
							break
						}

					}
					// few byte maybe left less than (512kb<=) to send that also so wee see all the result nicely , which would be helpful across all the goroutines and giving nice result
					if currentBuff > 0 {
						ct.upd_chann <- ResultUpdate{
							Part_Id:    Part_ID,
							Total_Size: ct.total_size,
							CurrOffset: currentOffset,
							Start:      start,
						}
					}
					resp.Body.Close()

					// if currentOffset >= expectedLimit {
					// 	break
					// }
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

		}(i)

	}
	slog.Info("[Concurrent]:All goroutine are fired!!")

	wg.Wait()
	if !d.cn.passed {
		slog.Error("[Concurrent-Error]: Concurrent Process Failed !!")
		fmt.Println("Download Failed!!")
		return
	} else {
		// remove the state file if there

		if ct.ctx.Err() == nil { // then only the  file has either gone fully downloaded or not canceled
			_, err := os.Stat(d.Rs.StateFileLocation)
			if err == nil {
				err := os.Remove(d.Rs.StateFileLocation)
				if err != nil {
					slog.Error("[Concurrent-Error]: StateFile Deletion Failed on download completion")
					return
				}
			}
			time.Sleep(1 * time.Second)
			fmt.Println("\nSUCESSFULLY DONE :)")
			return
		}

	}

}
