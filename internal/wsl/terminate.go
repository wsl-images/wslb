package wsl

import (
	"os/exec"
	"runtime"

	"github.com/wsl-images/wslb/internal/logger"
)

func TerminateDistribution(distroName string) {
	if runtime.GOOS != "windows" {
		logger.Fatal("The stop command can only be executed on Windows.")
	}

	logger.Info("Terminating WSL distribution: ", distroName)

	cmd := exec.Command("wsl.exe", "--terminate", distroName)
	output, err := cmd.CombinedOutput()
	if err != nil {
		logger.Error("Failed to terminate WSL distribution: ", err)
		logger.Error(string(output))
		return
	}

	logger.Info("Successfully terminated WSL distribution: ", distroName)
}
