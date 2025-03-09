package wsl

import (
	"os/exec"
	"runtime"

	"github.com/wsl-images/wslb/internal/logger"
)

// ShutdownAll terminates all running distributions and the WSL 2 lightweight virtual machine
func ShutdownAll() {
	if runtime.GOOS != "windows" {
		logger.Fatal("The shutdown command can only be executed on Windows.")
	}

	logger.Info("Shutting down all WSL distributions and the WSL 2 VM...")

	cmd := exec.Command("wsl.exe", "--shutdown")
	output, err := cmd.CombinedOutput()
	if err != nil {
		logger.Error("Failed to shutdown WSL: ", err)
		logger.Error(string(output))
		return
	}

	logger.Info("Successfully shut down all WSL distributions and the WSL 2 VM")
}
