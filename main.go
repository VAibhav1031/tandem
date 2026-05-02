package main

import (
	"fmt"
	"github.com/VAibhav1031/tandem/downloader"
)

func main() {

	fmt.Println("hey welcome brother")

	// currently no args will be given direct check will be given i  would try to do soo

	var link string = "https://filesamples.com/samples/document/csv/sample4.csv"

	req := downloader.NewServerLink(link, 0, "/home/necromancer/Downloads")
	res := req.ServerResponse()

	dow := downloader.DownloadWorker(res, req)

	dow.DownloadNormal()

	fmt.Println("Completed !!!")
}
