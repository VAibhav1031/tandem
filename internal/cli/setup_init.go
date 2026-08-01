package cli

import (
	_ "embed"
	"log/slog"
	"os"
	"os/exec"
)

//go:embed scripts/setup_script.sh
var setupString string

func RunSetup() {

	temp_file, err := os.CreateTemp("", "setup-*.sh")
	if err != nil {
		slog.Error("[CLI::SETUP-INIT]: Failed To Create Tempfile", slog.Any("error", err))
		return
	}

	defer os.Remove(temp_file.Name())

	if _, err := temp_file.WriteString(setupString); err != nil {
		slog.Error("[CLI::SETUP-INIT]: Failed to Write Script", slog.Any("error", err))
		return
	}
	temp_file.Close()

	if err := os.Chmod(temp_file.Name(), 0755); err != nil {
		slog.Error("[CLI::SETUP-INIT]: Failed to set persmission;", slog.Any("error", err))
		return
	}

	cmd := exec.Command("sh", temp_file.Name())
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		slog.Error("[CLI::SETUP-INIT]: Script execution failed", slog.Any("error", err))
		return
	}

	slog.Info("[CLI::SETUP-INIT]: Initial Setup is completed")

}
