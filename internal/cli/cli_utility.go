package cli

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"path"
	"strings"
	"time"

	"log/slog"

	"github.com/VAibhav1031/tandem/internal/downloader"
)

func Banner() {
	asciiArt := `  $$$$$$$$\                          $$\                           
  \__$$  __|                         $$|                          
     $$ | $$$$$$\   $$$$$$$\   $$$$$$$ | $$$$$$\  $$$$$$\$$\$$\   
     $$ | \____$$\ $$  __$$\ $$  __$$ |$$  __$$\ $$  _$$  _ $$\  
     $$ | $$$$$$$ |$$ |  $$ |$$ /  $$ |$$$$$$$$ |$$ / $$ / $$ | 
     $$ |$$  __$$ |$$ |  $$ |$$ |  $$ |$$   ____|$$ | $$ | $$ | 
     $$ |\$$$$$$$ |$$ |  $$ |\$$$$$$$ |\$$$$$$$\ $$ | $$ | $$ | 
     \__| \_______|\__|  \__| \_______| \_______|\__| \__| \__| `

	fmt.Println(asciiArt)
}

type Flags struct {
	Url_link     string
	Concurrent_n int
	Filepath     string
}

type ResultFlow struct {
	Result        string
	Fullpath      string
	HashStateFile string
	StateFile     *downloader.State_File_Format
}

const (
	CanResume = "resume"
	CanStart  = "fresh"
	Nothing   = "no"
)

// dedicated json file area ~/.local/tandem/jsons

// states and file (means the file is partially  downloaded and it has the state , the best part is that the json doesnt need to store the data ,  it  just need to store the last info just that , and then we are just gonna append write something we can say soo
// file and no state ( means fully downloaded)

// but the problem is that how we check like we have to check  this detail state is there file and  one more then how we get to know about the file if therre is no ourput location , like we have the following flags
// -url <link> -concurrent <n> -output <fullpath>
// then soo

var state_file_path string

const state_file_location = "/.local/tandem/json_data/"

func init() {

	home_dir, err := os.UserHomeDir()
	if err != nil {
		slog.Error("[CLI::CLI-UTILITY]:Error unable to get the Home Directory")
		return
	}
	state_file_path = home_dir + state_file_location
}

type headersDetails struct {
	headers *downloader.Responseheaders
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

func getExtensionFromUrl(rawUrl string) string {

	u, err := url.Parse(rawUrl)
	if err != nil {
		return ""
	}
	ext := path.Ext(u.Path)

	return ext
}
func (r *headersDetails) getFileInfo(url string) (string, string) {

	if r.headers.Content_deposition != "" {
		file_name := strings.Split(r.headers.Content_deposition, "filename=")[1]
		file_type := strings.Split(file_name, ".")[1]

		return file_name, file_type

	}

	if r.headers.Content_type != "" {
		// file_type := strings.Split(r.content_type, "/")[1]
		file_type := mimeToExt[r.headers.Content_type]

		// fmt.Println(r.headers.Content_type)
		return "", file_type
	}

	slog.Info("[CLI::CLI-UTILITY]:Filename, file_type choosen correctly")
	return "", getExtensionFromUrl(url)

	//
	//1 Deposition
	//2 Content Type
	//3 URL -Check
	//4 sniffing (initial packets)
	//5 fallback (default type .txt .bin or just default with no extensiongiven)
}

// if the output is not provided then we go with this

// same file name then we
func (f *Flags) dynamicResolution() (string, string, string) {
	// fmt.Println(filename, filetype)
	// http request get thing we need the GET , that is when we can do the

	ht := downloader.NewDualTransport()
	var chain http.RoundTripper = ht
	chain = &downloader.UTLSTransport{Next: ht}

	client := &http.Client{Transport: chain, Timeout: 60 * time.Second}
	req, err := http.NewRequest("GET", f.Url_link, nil)
	if err != nil {

		slog.Error("[CLI::CLI-UTILITY]: Error Ocurred <http Client GET req>", "error: ", err)
		return "", "", ""
	}
	resp, err := client.Do(req)
	if err != nil {

		slog.Error("[CLI::CLI-UTILITY]: Network Failure ", "error: ", err)
		fmt.Println("Connection Failure..")
		return "", "", ""
	}
	defer resp.Body.Close()

	server_details := downloader.ServerResponse(resp.Header)
	h := &headersDetails{headers: server_details}
	filename, filetype := h.getFileInfo(f.Url_link)
	var contentType string
	var preview []byte

	if filetype == "" {
		reader := resp.Body
		preview = make([]byte, 512)

		_, _ = reader.Read(preview)

		contentType = http.DetectContentType(preview)
		contentType = strings.Split(contentType, ";")[0]
		// fmt.Println(contentType)
		filetype = mimeToExt[contentType]

		// fmt.Println(filetype)
	}
	var fullpath string
	var filename_with_type string

	// one problem we have to save the same filename thing , like
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
	if f.Filepath != "" {
		fullpath = f.Filepath + filename_with_type
	} else {
		current_dir, err := os.Getwd()
		if err != nil {
			fmt.Printf("[CLI::CLI-UTILITY]: Error Ocurred <Current Directory>: %v\n", err)
			return "", "", ""
		}
		fullpath = current_dir + filename_with_type
		// fmt.Println("I am in else")
	}

	// // client request
	//  based on the header check for the
	return filename, filetype, fullpath
}

func filePathResolution(filePath string, increment_value int) string {
	splited_value := strings.Split(filePath, "/")
	type_splitted := strings.Split(splited_value[len(splited_value)-1], ".")
	type_extract := type_splitted[len(type_splitted)-1]
	to_be_joined_folder := splited_value[:len(splited_value)-1]
	fullPath := strings.Join(to_be_joined_folder, "/") + fmt.Sprintf("/download_file(%d)."+type_extract, increment_value)

	return fullPath
}

func (f *Flags) CheckResume() ResultFlow {
	// check for the flag filepath (included with filename at the end)
	var result_flow ResultFlow
	var fullPath string
	if f.Filepath == "" {
		_, _, fullPath = f.dynamicResolution()
		if fullPath == "" {
			return ResultFlow{} // Connection-failure
		}
		f.Filepath = fullPath

	}
	hash := sha256.Sum256([]byte(fullPath))
	hash_file_path := state_file_path + hex.EncodeToString(hash[:]) + ".json"

	increment := 0
	for {

		// not ver much good in the working of the os.Stat
		file_stat, err := os.Stat(hash_file_path)
		if err == nil {
			// file present
			// we have to read adn then we have to
			file_hash, err := os.OpenFile(hash_file_path, os.O_RDONLY, 0644)
			if err != nil {
				slog.Error("[CLI::CLI-UTILITY]: Error in File Opening", slog.Any("error", err))
				break
			}
			buffer := make([]byte, file_stat.Size())
			file_hash.Read(buffer)
			file_hash.Close() // Closing Time....

			var json_dedact downloader.State_File_Format
			err = json.Unmarshal(buffer, &json_dedact)
			if err != nil {
				slog.Error("[CLI::CLI-UTILITY]: Error in the State file Unmarshalling State", slog.Any("error", err))
				break
			}

			// check is it the same url it has or not if not then
			if json_dedact.Url != f.Url_link {
				// no it is nto se // means the hash opened here is of the duplicate path of the download  we need to  increment that
				increment++
				fullPath = filePathResolution(fullPath, increment)
				continue // check again for this filepath

			} else {
				//Resume safely
				fmt.Println(fullPath)
				result_flow.Result = CanResume
				result_flow.Fullpath = fullPath
				result_flow.HashStateFile = hash_file_path
				result_flow.StateFile = &json_dedact

				return result_flow
			}

		} else if errors.Is(err, os.ErrNotExist) {
			// not present
			_, err := os.Stat(fullPath)
			if err == nil {
				// filepath of same name file  exist but no state_file
				increment++
				fullPath = filePathResolution(fullPath, increment)
				continue // check again for this filepath

			} else if errors.Is(err, os.ErrNotExist) {

				// Start Fresh  filepath doesnt exist , means it is freash to start
				// return start_fresh

				fmt.Println(fullPath)
				result_flow.Result = CanStart
				result_flow.Fullpath = fullPath
				result_flow.HashStateFile = hash_file_path

				return result_flow

			}
		} else {
			slog.Error("[CLI::CLI-UTILITY]: Error in getting the stat of the Hash File")
		}
	}

	result_flow.Result = Nothing
	return result_flow
}

func update(ctx context.Context, results chan downloader.ResultUpdate, con int) {
	latestOffset := make([]int64, con)
	starts := make([]int64, con)
	seen := make([]bool, con)
	for msg := range results {
		// fmt.Println("DEBUG: Got update from Part", msg.Part_Id, "at offset", msg.CurrOffset)
		if !seen[msg.Part_Id] {
			starts[msg.Part_Id] = msg.Start
			seen[msg.Part_Id] = true
		}
		latestOffset[msg.Part_Id] = msg.CurrOffset

		var current_total int64
		for i := 0; i < con; i++ {
			if seen[i] { // why this cause we would include  only those value which are present and contributinng , means which are downloading and sedn the data to update goroutine no to  who are in-process or in the transit to send
				current_total += latestOffset[i] - starts[i]
			}
		}
		percentage := (float64(current_total) / float64(msg.Total_Size)) * 100

		fmt.Printf("\r\033[Kcurrent Percentage : %.2f%%", percentage)
		// time.Sleep(200 * timeMillisecond)
		os.Stdout.Sync()
	}
	fmt.Println()
}

func Usage() {
	// need to
	multi_usage := `
Usage:
	tandem -u <url> -c <concurrency> -o <output path>

TheCommands are :
	help   Give usage Message 
	setup  For initial setup of the essential directories of logs and state-files

Insight:
	help <sub-command> :
		Will give the usage details  of the sub-command.
		Here sub command related to the arguments/command provided.
	
	`
	fmt.Println(multi_usage)
}

func con_usage() {
	// help/usage for the subcommand

	con := `
"concurrent" Usage:
	
	-c <value> / -concurrent <value> / -C <value> 
	
	This command  is used to  provide the   concurrent downloading workers,
	Which will be used for the downloading  of the object. 
	

<value>:
	Only from  0 to 9 are allowed.

DEFAULT:
	By default 4 is used for better usage.

`

	fmt.Println(con)
}

func url_usage() {
	con := `
"url" Usage:
	
	-u <url> / -url <url> / -U <url> 
	
	url command used to provide the url for which download request has to be
	taken.
	
<url>:
	A perfect url with https:// is allowed 

DEFAULT:
	There  is no default url , it is necessary to provide url.
	Else it would be error

	`

	fmt.Println(con)
}
