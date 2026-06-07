package downloader

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"path"
	"strings"
	"time"

	"golang.org/x/net/http2"
)

const buffer_length int = 32 * 1024

type DownloadInfo struct {
	Rs *RequestServer
	// FileInfo *Responseheaders // i really dont feel now there is a use case for this cause there is no pre req thing happen
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

func (d *DownloadInfo) DownloadNormal() {

	// h2t := &http2.Transport{
	// 	DialTLSContext: dialUTLS,
	// 	// AllowHTTP: false, // keep false for real use
	// }

	h := &DualTransport{
		H1: &http.Transport{},
		H2: &http2.Transport{},
	}

	ht := NewDualTransport(h)

	// DefaultTransport is also RoundTripper casuse it has the RoundTrip method with it
	// so now in this condition it was like we have to  have to remove that for teh internet request andd add the new one her eit is the h2t
	// which is also a Transport but not the DefaultTransport one , but it satisfy condition ,plus with our custom TLS thing ,  and the uTLSuTLSTransport struct will create the request to the h2t with those uTuTLSTransport RoundTrip request added Headers
	var chain http.RoundTripper = ht

	chain = &uTLSTransport{Next: h}
	chain = &LocalCookieTransport{Next: chain}
	chain = &SolverTransport{Next: chain}

	client := &http.Client{Transport: chain, Timeout: 30 * time.Second}

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

	req_head := ServerResponse(resp.Header)
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
	fmt.Println(d.Rs.Location)

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

func (d *DownloadInfo) ConcurrentDownloader() {

}
