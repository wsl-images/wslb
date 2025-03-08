package docker

import (
	"github.com/wsl-images/wslb/internal/logger"
	"os/exec"
)

func RunContainer(containerName, image string) error {
	logger.Debug("Running container ", containerName, " for image ", image)
	cmd := exec.Command("docker", "run", "-t", "--name", containerName, image, "ls", "/")
	return cmd.Run()
}

func CleanupContainer(containerName string) {
	_ = exec.Command("docker", "rm", containerName).Run()
}
