package install

import (
	"github.com/wsl-images/wslb/internal/logger"
	"os"
	"os/exec"
)

func InstallWSL(wslFile, customName string) {
	logger.Info("Installing WSL distro...")
	installArgs := []string{"--install", "--from-file", wslFile}
	if customName != "" {
		installArgs = append(installArgs, "--name", customName)
	}
	installCmd := exec.Command("wsl", installArgs...)
	installCmd.Stdout = os.Stdout
	installCmd.Stderr = os.Stderr
	if err := installCmd.Run(); err != nil {
		logger.Error("WSL install command failed: ", err)
		os.Exit(1)
	}
	logger.Info("WSL distro installed successfully!")
}
