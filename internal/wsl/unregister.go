package wsl

import (
	"os/exec"
	"runtime"

	"github.com/wsl-images/wslb/internal/logger"
)

// UnregisterDistribution unregisters the specified WSL distribution and deletes its root filesystem
func UnregisterDistribution(distroName string) {
	if runtime.GOOS != "windows" {
		logger.Fatal("The rm command can only be executed on Windows.")
	}

	logger.Info("Unregistering WSL distribution: ", distroName)

	cmd := exec.Command("wsl.exe", "--unregister", distroName)
	output, err := cmd.CombinedOutput()
	if err != nil {
		logger.Error("Failed to unregister WSL distribution: ", err)
		logger.Error(string(output))
		return
	}

	logger.Info("Successfully unregistered WSL distribution: ", distroName)
}
