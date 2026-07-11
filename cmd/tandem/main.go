package main

import (
	"github.com/VAibhav1031/tandem/internal/cli"
	"github.com/VAibhav1031/tandem/internal/logger"
)

func main() {
	logger.LoggerInitiator()
	cli.Execute()
}
