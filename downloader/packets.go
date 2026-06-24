package downloader

import (
	// "context"
	"fmt"
	"io"
	"log"
	"math"
	"net/http"
	"net/url"
	"os"
	"path"
	"strconv"
	"strings"
	"sync"
	"time"
	// "golang.org/x/net/http2"
)

const buffer_length int = 32 * 1024

type DownloadInfo struct {
	Rs *RequestServer
	// FileInfo *Responseheaders // i really dont feel now there is a use case for this cause there is no pre req thing happen
	cn conCurrentDet
}
type RequestServer struct {
	Link     string
	Con_n    int8 // we are thinking nobody gonna give more than this
	Location string
}

type Responseheaders struct {
	content_length     string
	content_type       string
	content_deposition string
	accept_ranges      string
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
func NewServerLink(link string, n int8, location string) *RequestServer {
	return &RequestServer{
		Link:     link,
		Con_n:    n,
		Location: location,
	}
}

// i want to create the request (GET) to the server link and  all shit ,  and based on the response i will do  a shit and all shit

func ServerResponse(headers http.Header) *Responseheaders {

	return &Responseheaders{content_length: headers.Get("Content-Length"), content_type: headers.Get("Content-Type"), accept_ranges: headers.Get("Accept-Ranges")}

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

	if r.content_deposition != "" {
		file_name := strings.Split(r.content_deposition, "filename=")[1]
		file_type := strings.Split(file_name, ".")[1]

		fmt.Println("THIS IS IT")
		return file_name, file_type

	}

	if r.content_type != "" {
		// file_type := strings.Split(r.content_type, "/")[1]
		file_type := mimeToExt[r.content_type]

		fmt.Println(r.content_type)
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
		log.Printf("[Downloader] Error Ocurred <http Client GET req> : %v\n", err)
	}
	resp, err := client.Do(req)
	if err != nil {
		log.Println("[Downloader] Network error", err)
	}

	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		log.Println("[Downloader] Failure :", resp.StatusCode)
		return
	}

	// req_head := ServerResponse(resp.Header)

	filename, filetype := req_head.getFileInfo(d.Rs.Link)
	fmt.Println(filename, filetype)
	var contentType string
	var preview []byte
	if filetype == "" {
		reader := resp.Body
		preview = make([]byte, 512)

		_, _ = reader.Read(preview)

		contentType = http.DetectContentType(preview)
		contentType = strings.Split(contentType, ";")[0]
		fmt.Println(contentType)
		filetype = mimeToExt[contentType]

		// fmt.Println(filetype)
	}
	// fmt.Println(filetype, filename)
	// create the buffer , like 8kb or something which get fill--up and then  call the write thing to the file anda all shit and that
	var fullpath string
	var filename_with_type string
	// fmt.Println(d.Rs.Location)

	// FileInfo: filenname with type
	if filename != "" && filetype == "" {
		filename_with_type = "/" + filename + ".bin"
	} else if filename == "" && filetype != "" {
		filename_with_type = "/download_file" + "." + filetype
	} else if filename != "" && filetype != "" {
		filename_with_type = "/" + filename + "." + filetype
	} else {
		filename_with_type = "/download_file.bin"
	}

	// LocationInfo: file location addition , for the fullpath creation
	if d.Rs.Location != "" {
		fullpath = d.Rs.Location + filename_with_type
	} else {
		current_dir, err := os.Getwd()
		if err != nil {
			fmt.Printf("[Downloader]: Error Ocurred <Current Directory>: %v\n", err)
			return
		}
		fullpath = current_dir + filename_with_type
		// fmt.Println("I am in else")
	}

	out, err := os.Create(fullpath)
	if err != nil {
		fmt.Printf("[Downloader]: Error occurred <File creation>: %v", err)
	}

	buffer_read := make([]byte, buffer_length) //buffer_lenght --> 32kb length
	contentLength := resp.ContentLength        // This is an int64
	var downloaded int64 = 0

	if preview != nil {

		out.Write(preview)
	}
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
const globalLimit int = 3

func (d *DownloadInfo) ConcurrentDownloader(headers *Responseheaders, client *http.Client) {

	d.cn.n = 4
	total_size, _ := strconv.Atoi(headers.content_length)
	log.Println("total_size of the file", total_size, "and in the gb", (float64(total_size) / float64(1024*1024*1024)))

	//**************current_Problem
	batch_size := int64(math.Ceil(float64(total_size) / float64(d.cn.n))) // size need to be clearl y round so that slicing doesnt give problems
	log.Printf("floated value : %v", math.Ceil(float64(total_size)/float64(d.cn.n)))
	d.cn.buffer = make([]byte, total_size)
	d.cn.passed = true

	start, limit := int64(0), batch_size

	wg := &sync.WaitGroup{}
	for i := 0; i < d.cn.n; i++ {
		// current outer variables limit, start, is the passed one , the outer is the globalLimit one

		log.Printf("GOROUTINE %d-->Start: %d, limit: %d", i, start, limit)

		// ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		wg.Add(1)
		go func(start int64, limit int64) {
			defer wg.Done()

			// there is a new client is being created here , and all shit
			// client := &http.Client{Transport: chain, Timeout: 30 * time.Second}
			req, err := http.NewRequest("GET", d.Rs.Link, nil) // new request , default http Transport with TLS , https support based on that
			if err != nil {
				log.Println("[Concurrent-ERROR]: ", err)
			}
			req.Header.Set("Range", fmt.Sprintf("bytes=%d-%d", start, limit-1))
			// request based on the range and allshit
			current := 0
			for {

				if current == globalLimit {
					d.cn.mw.Lock()
					d.cn.passed = false
					d.cn.mw.Unlock()
					log.Println("All Limit Crossed!! Exitting Goroutine..")
					return
				}

				resp, err := client.Do(req)
				if err != nil {
					log.Printf("[Concurrent-Error]: Connection Failed %v, ", err)
					// exit the goroutine thing...:_)
					current++
					// resp.Body.Close()
					time.Sleep(1 * time.Second)
					continue
				}

				if resp.StatusCode != http.StatusPartialContent && resp.StatusCode != http.StatusOK {
					log.Printf("[Concurrent-Error]: Unexpected status code : %d", resp.StatusCode)
					resp.Body.Close()
					current++
					continue
				}

				// read to the correct section of the buffer
				destBuffer := d.cn.buffer[start:limit]
				// Since each slice of the buffere is not overlapping , so  there is no need to put the lock over the buffer and we cango easily and it is the design which help it move
				n, err := io.ReadFull(resp.Body, destBuffer)
				resp.Body.Close()
				if n < 0 || err != nil {
					log.Printf("[Concurrent-Error]: BOOOM!!, start %d: limit %d | Read-up ERR-> %v", start, limit, err)
					current++
					continue
				}

				break
			}

		}(start, limit)
		start = limit
		limit = start + batch_size

		if limit%int64(total_size) != limit {
			limit = limit - (limit % int64(total_size))
		}
		// we need to wait till all the goroutines are complete  then proceed with lower ,  if done then based on the passed bool value we proceed like if it went well or not if not then we will just skip that and all shit
	}

	// i got one solution
	wg.Wait()

	if !d.cn.passed {
		log.Printf("[Concurrent-Error]: Concurrent Process Failed !!")
		return
	}
	filename, filetype := headers.getFileInfo(d.Rs.Link)
	// fmt.Println(filename, filetype)
	var contentType string
	var preview []byte
	if filetype == "" {
		// reader := resp.Body
		preview = d.cn.buffer[:513]

		// _, _ = reader.Read(preview)

		contentType = http.DetectContentType(preview)
		contentType = strings.Split(contentType, ";")[0]
		fmt.Println("Content-Type", contentType)
		filetype = mimeToExt[contentType]

		// fmt.Println(filetype)
	}
	// fmt.Println(filetype, filename)
	var fullpath string
	var filename_with_type string
	// fmt.Println(d.Rs.Location)

	// FileInfo: filenname with type
	if filename != "" && filetype == "" {
		filename_with_type = "/" + filename + ".bin"
	} else if filename == "" && filetype != "" {
		filename_with_type = "/download_file" + "." + filetype
	} else if filename != "" && filetype != "" {
		filename_with_type = "/" + filename + "." + filetype
	} else {
		filename_with_type = "/download_file.bin"
	}

	// LocationInfo: file location addition , for the fullpath creation
	if d.Rs.Location != "" {
		fullpath = d.Rs.Location + filename_with_type
	} else {
		current_dir, err := os.Getwd()
		if err != nil {
			fmt.Printf("[Concurrent-Downloader]: Error Ocurred <Current Directory>: %v\n", err)
			return
		}
		fullpath = current_dir + filename_with_type
		// fmt.Println("I am in else")
	}

	out, err := os.Create(fullpath)
	if err != nil {
		log.Printf("[Concurrent-Downloader]: Error occurred <File creation>: %v", err)
	}

	out.Write(d.cn.buffer)

}
