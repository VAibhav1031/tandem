package logger

import (
	"log/slog"
	"os"
)

func LoggerInitiator() {

	homeDir, err := os.UserHomeDir()
	if err != nil {

		slog.Error("Failed to open the log file", "error", err)
		return
	}
	logFilepath := homeDir + "/.local/tandem/logs/unilog.log"
	file, err := os.OpenFile(logFilepath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		slog.Error("Failed to open the log file", "error", err)
	}

	handler := slog.NewJSONHandler(file, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	})

	logger := slog.New(handler)
	slog.SetDefault(logger)
}
