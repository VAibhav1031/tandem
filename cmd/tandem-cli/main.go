package main

import (
	"github.com/VAibhav1031/tandem/internal/cli"
	"github.com/VAibhav1031/tandem/internal/logger"
	"os"
)

func main() {
	if len(os.Args) > 1 && os.Args[1] == "--setup" {
		cli.RunSetup()
		return
	}
	logger.LoggerInitiator()
	cli.Execute()
}
