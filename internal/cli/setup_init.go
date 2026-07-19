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
		slog.Error("[CLI::SETUP-INIT]: Failed To Create Tempfile")
		return
	}

	defer os.Remove(temp_file.Name())

	if _, err := temp_file.WriteString(setupString); err != nil {
		slog.Error("[CLI::SETUP-INIT]: Failed to Write Script")
		return
	}
	temp_file.Close()

	if err := os.Chmod(temp_file.Name(), 0755); err != nil {
		slog.Error("[CLI::SETUP-INIT]: Failed to set persmission;")
		return
	}

	cmd := exec.Command("bash", temp_file.Name())
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		slog.Error("[CLI::SETUP-INIT]: Script execution failed")
	}

	slog.Info("[CLI::SETUP-INIT]: Initial Setup is completed")

}
