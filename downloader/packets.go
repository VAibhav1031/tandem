package downloader

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path"
	"strings"
	"time"
)

const buffer_length int = 32 * 1024

type DownloadInfo struct {
	Rs       *RequestServer
	FileInfo *Responseheaders
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
	"text/plain":      "txt",
	"text/csv":        "csv",
	"application/pdf": "pdf",
	"image/jpeg":      "jpg",
	"image/png":       "png",
	"application/zip": "zip",
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

func (l *RequestServer) ServerResponse() *Responseheaders {
	resp, err := http.Head(l.Link)

	if err != nil {
		fmt.Printf("Errot Occurred %v \n", err)
	}

	defer resp.Body.Close()
	return &Responseheaders{content_length: resp.Header.Get("Content-Length"), content_type: resp.Header.Get("Content-Type"), accept_ranges: resp.Header.Get("Accept-Ranges")}

}

func DownloadWorker(response *Responseheaders, request *RequestServer) *DownloadInfo {
	return &DownloadInfo{Rs: request, FileInfo: response}
}
func (r *Responseheaders) ConcurrentCheck() bool {

	accept_range := r.accept_ranges

	if accept_range == "" {
		return false
	} else {
		return true
	}

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

		return file_name, file_type

	}

	if r.content_type != "" {
		// file_type := strings.Split(r.content_type, "/")[1]
		file_type := mimeToExt[r.content_type]
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

	// i really think
	tr := &http.Transport{
		// MaxIdleConns:       5,
		IdleConnTimeout: 30 * time.Second,
		// DisableCompression: true,
	}
	client := &http.Client{Transport: tr}

	resp, err := client.Get(d.Rs.Link)

	if err != nil {
		fmt.Printf("Error Ocurred: %v\n", err)
	}

	defer resp.Body.Close()

	filename, filetype := d.FileInfo.getFileInfo(d.Rs.Link)

	var contentType string
	var preview []byte
	if filetype == "" {
		reader := resp.Body
		preview = make([]byte, 512)

		_, _ = reader.Read(preview)

		contentType = http.DetectContentType(preview)
		filetype = mimeToExt[contentType]
	}

	fmt.Println(filetype)
	// create the buffer , like 8kb or something which get fill uyp and then that call the write thing to the file anda all shit and that
	var fullpath string
	if d.Rs.Location != "" {

		if filename != "" {
			fullpath = d.Rs.Location + "/" + filename + "." + filetype
		} else {
			fullpath = d.Rs.Location + "/download_file" + "." + filetype
		}
	} else {
		current_dir, err := os.Getwd()

		if err != nil {
			fmt.Printf("Error Ocurred: %v\n", err)
			return
		}

		if filetype == "" {
			fullpath = current_dir + "/download_file.bin"
		} else {
			fullpath = current_dir + "/download_file" + "." + filetype
		}
	}

	out, err := os.Create(fullpath)

	if err != nil {
		fmt.Printf("Error occurred: %v", err)
	}

	buffer_read := make([]byte, buffer_length) //buffer_lenght --> 32kb length

	contentLength := resp.ContentLength // This is an int64

	var downloaded int64 = 0

	if preview != nil {

		out.Write(preview)
	}
	for {
		// Read from network into buffer
		n, err := resp.Body.Read(buffer_read)

		if n > 0 {
			// Write this chunk to disk
			out.Write(buffer_read[:n])

			// Update the counter
			downloaded += int64(n)

			// Comparison works here!
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

func ConcurrentDownloader() {}
