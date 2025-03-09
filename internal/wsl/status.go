package wsl

import (
	"fmt"
	"os/exec"
	"runtime"
	"strings"

	"github.com/wsl-images/wslb/internal/logger"
)

// ShowStatus displays the status of Windows Subsystem for Linux
func ShowStatus() {
	if runtime.GOOS != "windows" {
		logger.Fatal("The status command can only be executed on Windows.")
	}

	logger.Info("Checking WSL status...")

	cmd := exec.Command("wsl.exe", "--status")
	output, err := cmd.CombinedOutput()
	if err != nil {
		logger.Error("Failed to get WSL status: ", err)
		logger.Error(string(output))
		return
	}

	fmt.Println(strings.TrimSpace(string(output)))
}
