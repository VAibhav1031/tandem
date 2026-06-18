package main

import (
	"fmt"

	// cookies "github.com/VAibhav1031/tandem/cookiesManager"
	"github.com/VAibhav1031/tandem/downloader"
)

func main() {

	fmt.Println("=========TANDEM DOWNLOADER===========")

	// currently no args will be given direct check will be given i  would try to do soo

	// var link string = "https://filesamples.com/samples/document/csv/sample4.csv"
	var link string = "https://cdn.hotelnearmedanta.com/testfile.org/testfile.org-5GB.dat"
	// var link string = "https://link.testfile.org/250MB"
	// var link string = "https://files.testfile.org/ZIPC/300MB-Corrupt-Testfile.Org.zip"
	req := downloader.NewServerLink(link, 0, "/home/necromancer/Downloads")

	dow := downloader.DownloadWorker(req)
	// dow.DownloadNormal()
	dow.ConcurrentDownloader()
	fmt.Println("Completed !!!")

}
