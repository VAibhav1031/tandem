package main

import (
	"fmt"
	"github.com/VAibhav1031/tandem/internal/cli"
	"github.com/VAibhav1031/tandem/internal/logger"
	"os"
	"runtime/pprof"
)

func main() {
	if len(os.Args) > 1 && os.Args[1] == "--setup" {
		cli.RunSetup()
		return
	}
	cpuLogPath := os.Getenv("PPROF_CPU")
	memLogPath := os.Getenv("PPROF_MEM")

	// 2. Handle CPU profile if env var is set
	if cpuLogPath != "" {
		f, err := os.Create(cpuLogPath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "pprof error: %v\n", err)
		} else {
			defer f.Close()
			pprof.StartCPUProfile(f)
			defer pprof.StopCPUProfile()
		}
	}
	logger.LoggerInitiator()
	cli.Execute()
	if memLogPath != "" {
		f, err := os.Create(memLogPath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "pprof error: %v\n", err)
		} else {
			defer f.Close()
			if err := pprof.WriteHeapProfile(f); err != nil {
				fmt.Fprintf(os.Stderr, "pprof error: %v\n", err)
			}
		}
	}
}
