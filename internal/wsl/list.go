package wsl

import (
	"fmt"
	"os/exec"
	"runtime"
	"strings"

	"github.com/wsl-images/wslb/internal/logger"
)

// ListDistributions lists the WSL distributions based on the provided flags
func ListDistributions(all, running, quiet, verbose, online bool) {
	if runtime.GOOS != "windows" {
		logger.Fatal("The list command can only be executed on Windows.")
	}

	args := []string{"--list"}

	if all {
		args = append(args, "--all")
	}
	if running {
		args = append(args, "--running")
	}
	if quiet {
		args = append(args, "--quiet")
	}
	if verbose {
		args = append(args, "--verbose")
	}
	if online {
		args = append(args, "--online")
	}

	cmd := exec.Command("wsl.exe", args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		logger.Error("Failed to list WSL distributions: ", err)
		logger.Error(string(output))
		return
	}

	fmt.Println(strings.TrimSpace(string(output)))
}
