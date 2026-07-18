package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"os/signal"

	"github.com/VAibhav1031/tandem/internal/downloader"
)

const DefaultConcurrency = 4

func Execute() {

	Banner()
	fmt.Println("==========TANDEM DOWNLOADER===========")
	// var link string = "https://filesamples.com/samples/document/csv/sample4.csv"
	// var link string = "https://pub-821312cfd07a4061bf7b99c1f23ed29b.r2.dev/3dicons-png-dynamic-1.0.0.zip"
	// var link string = "https://ash-speed.hetzner.com/100MB.bin"
	// var link string = "https://files.testfile.org/ZIPC/300MB-Corrupt-Testfile.Org.zip"

	f := &Flags{}

	// parse
	// we need the error to enforece it here nicely i think soo
	err := f.Parser()
	if err != nil {
		slog.Error("[CLI::EXECUTE]:Parsing Failed")
		os.Exit(1)
	}

	// we have to check we can resume , if so then it is okay , if not then we have to start again
	check := f.CheckResume()
	req := downloader.NewServerLink(f.Url_link, 0, check.Fullpath, check.HashStateFile)
	dow := downloader.DownloadWorker(req)

	concurrencyLimit := f.Concurrent_n
	if concurrencyLimit <= 0 {
		slog.Info("[CLI::EXECUTE]: No concurrency count provided. Falling back to default.", "workers", DefaultConcurrency)
		concurrencyLimit = DefaultConcurrency

	}
	// Adding thge concurrentcy Limit
	dow.Rs.Con_n = int8(concurrencyLimit)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	flowState := &downloader.StateFile{}

	if check.Result == "resume" {

		flowState.Resume_stf = check.StateFile
		flowState.Stf = &downloader.State_File_Format{
			Con:        check.StateFile.Con, // we need to use last state number of con
			Url:        f.Url_link,
			Filepath:   f.Filepath,
			LastRanges: make([]downloader.Ranges, concurrencyLimit)}
		dow.Resolve(ctx, flowState)

	} else if check.Result == "fresh" {
		flowState.Resume_stf = &downloader.State_File_Format{}
		flowState.Stf = &downloader.State_File_Format{
			Con:        dow.Rs.Con_n,
			Url:        f.Url_link,
			Filepath:   f.Filepath,
			LastRanges: make([]downloader.Ranges, concurrencyLimit)}

		dow.Resolve(ctx, flowState)
	} else {

		slog.Error("[CLI::EXECUTE]: ERROR NOTHING<'-'>")
		return
	}

	if ctx.Err() == context.Canceled {
		// we  have to save the state file
		// open the file

		statefile, err := os.OpenFile(check.HashStateFile, os.O_CREATE|os.O_WRONLY, 0644)

		if err != nil {
			slog.Error("[CLI::EXECUTE]:Unable to open the file")
		}

		json_format, err := json.Marshal(flowState.Stf)
		if err != nil {
			slog.Error("[CLI::EXECUTE]:JSON Marshalling Failed!!")
			return
		}

		statefile.Write(json_format)
	}
}
