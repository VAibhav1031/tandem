package main

import (
	"fmt"
	"github.com/VAibhav1031/tandem/downloader"
)

func main() {

	fmt.Println("=========TANDEM DOWNLOADER===========")

	// var link string = "https://filesamples.com/samples/document/csv/sample4.csv"
	var link string = "https://pub-821312cfd07a4061bf7b99c1f23ed29b.r2.dev/3dicons-png-dynamic-1.0.0.zip"
	// var link string = "https://ash-speed.hetzner.com/100MB.bin"
	// var link string = "https://files.testfile.org/ZIPC/300MB-Corrupt-Testfile.Org.zip"
	req := downloader.NewServerLink(link, 0, "/home/necromancer/Downloads")

	dow := downloader.DownloadWorker(req)

	dow.Maxim()
	fmt.Println("Completed !!!")

}
