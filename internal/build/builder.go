package build

import (
	"archive/tar"
	"github.com/wsl-images/wslb/internal/docker"
	"github.com/wsl-images/wslb/internal/logger"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// BuildWSL builds a WSL distribution from a Docker image with specified verbosity
func BuildWSL(dockerImage, outputDir string, verbose bool) string {
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		logger.Error("Failed to create output directory: ", err)
		os.Exit(1)
	}

	absOutputDir, err := filepath.Abs(outputDir)
	if err != nil {
		logger.Error("Failed to resolve absolute path: ", err)
		os.Exit(1)
	}

	distroName := strings.Split(filepath.Base(dockerImage), ":")[0]
	containerName := distroName

	if verbose {
		logger.Info("Starting build for ", distroName, " WSL distro")
	} else {
		logger.Debug("Starting build for ", distroName, " WSL distro")
	}

	if err := docker.RunContainer(containerName, dockerImage); err != nil {
		logger.Error("Failed to run container: ", err)
		docker.CleanupContainer(containerName)
		os.Exit(1)
	}

	tarPath := filepath.Join(absOutputDir, distroName+"-wsl.tar")
	outFile, err := os.Create(tarPath)
	if err != nil {
		logger.Error("Failed to create tar file: ", err)
		docker.CleanupContainer(containerName)
		os.Exit(1)
	}
	tarWriter := tar.NewWriter(outFile)

	exportCmd := exec.Command("docker", "export", containerName)
	stdout, err := exportCmd.StdoutPipe()
	if err != nil {
		logger.Error("Failed to get stdout from docker export: ", err)
		docker.CleanupContainer(containerName)
		os.Exit(1)
	}
	if err := exportCmd.Start(); err != nil {
		logger.Error("Failed to start docker export: ", err)
		docker.CleanupContainer(containerName)
		os.Exit(1)
	}

	tarReader := tar.NewReader(stdout)
	for {
		header, err := tarReader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			logger.Error("Error reading tar stream: ", err)
			docker.CleanupContainer(containerName)
			os.Exit(1)
		}

		if header.Name == "etc/resolv.conf" {
			continue
		}

		if err := tarWriter.WriteHeader(header); err != nil {
			logger.Error("Error writing tar header: ", err)
			docker.CleanupContainer(containerName)
			os.Exit(1)
		}
		if _, err := io.Copy(tarWriter, tarReader); err != nil {
			logger.Error("Error copying file data to tar: ", err)
			docker.CleanupContainer(containerName)
			os.Exit(1)
		}
	}

	if err := exportCmd.Wait(); err != nil {
		logger.Error("Docker export command failed: ", err)
		docker.CleanupContainer(containerName)
		os.Exit(1)
	}
	_ = tarWriter.Close()
	_ = outFile.Close()

	docker.CleanupContainer(containerName)

	wslPath := filepath.Join(absOutputDir, distroName+".wsl")
	if err := os.Rename(tarPath, wslPath); err != nil {
		logger.Error("Failed to rename file: ", err)
		os.Exit(1)
	}

	if verbose {
		logger.Info("WSL distro build completed successfully. Output file: ", wslPath)
	} else {
		logger.Debug("WSL distro build completed successfully. Output file: ", wslPath)
	}

	return wslPath
}
