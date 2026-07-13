package cli

import (
	"context"
	"fmt"
	"github.com/VAibhav1031/tandem/internal/downloader"
	"os"
	"os/signal"
)

func Execute() {

	Banner()
	fmt.Println("==========TANDEM DOWNLOADER===========")
	// var link string = "https://filesamples.com/samples/document/csv/sample4.csv"
	// var link string = "https://pub-821312cfd07a4061bf7b99c1f23ed29b.r2.dev/3dicons-png-dynamic-1.0.0.zip"
	// var link string = "https://ash-speed.hetzner.com/100MB.bin"
	// var link string = "https://files.testfile.org/ZIPC/300MB-Corrupt-Testfile.Org.zip"
	// req := downloader.NewServerLink(link, 0, "/home/necromancer/Downloads")
	//
	// dow := downloader.DownloadWorker(req)
	//
	// dow.Maxim()
	// fmt.Println("Completed !!!")

	f := &Flags{}

	// parser
	f.Parser()
	// we have to check we can resume , if so then it is okay , if not then we have to start again
	check := f.CheckResume()
	req := downloader.NewServerLink(f.Url_link, 0, check.Fullpath, check.HashStateFile)
	dow := downloader.DownloadWorker(req)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	// for the use case wher we have to use the resume STF data and pause details thing , just to make sure the this manage the pause and continyue thing
	flowState := &downloader.StateFile{}
	if check.Result == "resume" {
		// recall the stored bytes , but how with the current details and start , but there is a thing  called server provide accept-range thing andall shit , in the download function if doesnt have then we have to update that shit and all thing on the terminal

		// give to the maxim  and that will decide whether to see and move

		// there is a state file  ,  there is a file path ,  means there is both ,  we have to continue , we have to  go with that thing

		flowState.Resume_stf = check.StateFile
		flowState.Stf = &downloader.State_File_Format{}
		dow.Maxim(ctx, flowState)

	} else if check.Result == "fresh" {
		flowState.Stf = &downloader.State_File_Format{}
		dow.Maxim(ctx, flowState)
	} else {

		fmt.Println("ERROR NOTHING<'-'>")
	}

	// must use the "resume" "start" "no"  so we can go easily make it formal to the top somethinglike that would be better and nice to goo

}
