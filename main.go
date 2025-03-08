package main

import (
	"archive/tar"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/spf13/pflag"
)

func main() {
	if len(os.Args) < 2 {
		usage()
		os.Exit(1)
	}

	cmd := os.Args[1]
	switch cmd {
	case "build":
		// Build command: allow interspersed flags.
		buildCmd := pflag.NewFlagSet("build", pflag.ExitOnError)
		outputDir := buildCmd.StringP("output", "o", ".", "Output directory for the WSL file")
		buildCmd.Parse(os.Args[2:])

		args := buildCmd.Args()
		if len(args) < 1 {
			fmt.Println("Error: Docker image path is required")
			fmt.Println("Usage: wsl-builder build [-o output-dir] docker-image")
			os.Exit(1)
		}
		dockerImage := args[0]
		buildWSL(dockerImage, *outputDir)

	case "install":
		// Install command supports two optional flags: -n for name, -f for file.
		installCmd := pflag.NewFlagSet("install", pflag.ExitOnError)
		customName := installCmd.StringP("name", "n", "", "Custom name for installation")
		fileFlag := installCmd.StringP("file", "f", "", "Path to a pre-built .wsl file")
		installCmd.Parse(os.Args[2:])

		var wslFile string
		tmpDirUsed := false

		// If -f is provided, use that file.
		if *fileFlag != "" {
			wslFile = *fileFlag
		} else {
			// Otherwise, expect an image URL as a positional argument.
			args := installCmd.Args()
			if len(args) < 1 {
				fmt.Println("Error: Either a file must be provided with -f or an image URL must be specified.")
				fmt.Println("Usage: wsl-builder install [-n customName] [-f file] [image URL]")
				os.Exit(1)
			}
			imageURL := args[0]
			// Use a temporary directory for the build.
			tmpDir := "./tmp"
			buildWSL(imageURL, tmpDir)
			// The buildWSL function names the output file as "<distroName>.wsl"
			distroName := strings.Split(filepath.Base(imageURL), ":")[0]
			wslFile = filepath.Join(tmpDir, distroName+".wsl")
			tmpDirUsed = true
		}

		// Build the wsl install command.
		installArgs := []string{"--install", "--from-file", wslFile}
		if *customName != "" {
			installArgs = append(installArgs, "--name", *customName)
		}
		logStep("Installing WSL distro...")
		installCmdExec := exec.Command("wsl", installArgs...)
		installCmdExec.Stdout = os.Stdout
		installCmdExec.Stderr = os.Stderr
		if err := installCmdExec.Run(); err != nil {
			logError(fmt.Sprintf("WSL install command failed: %v", err))
			if tmpDirUsed {
				os.RemoveAll("./tmp")
			}
			os.Exit(1)
		}
		logSuccess("WSL distro installed successfully!")
		if tmpDirUsed {
			os.RemoveAll("./tmp")
		}

	default:
		usage()
		os.Exit(1)
	}
}

func usage() {
	fmt.Println("Usage:")
	fmt.Println("  wsl-builder build [-o output-dir] docker-image")
	fmt.Println("  wsl-builder install [-n customName] [-f file] [image URL]")
}

func buildWSL(dockerImage, outputDir string) {
	// Create output directory if it doesn't exist.
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		logError(fmt.Sprintf("Failed to create output directory: %v", err))
		os.Exit(1)
	}

	absOutputDir, err := filepath.Abs(outputDir)
	if err != nil {
		logError(fmt.Sprintf("Failed to resolve absolute path for output directory: %v", err))
		os.Exit(1)
	}

	// Extract distro name from the Docker image.
	distroName := strings.Split(filepath.Base(dockerImage), ":")[0]
	containerName := distroName

	logInfo(fmt.Sprintf("===== Starting build for %s WSL distro =====", distroName))

	// Step 1: Run container.
	logStep("Running container to prepare filesystem...")
	runCmd := exec.Command("docker", "run", "-t", "--name", containerName, dockerImage, "ls", "/")
	if err := runCmd.Run(); err != nil {
		logError(fmt.Sprintf("Failed to run container: %v", err))
		cleanupContainer(containerName)
		os.Exit(1)
	}

	// Step 2: Export container to a tarball while filtering out etc/resolv.conf.
	logStep("Exporting container to tarball...")
	tarPath := filepath.Join(absOutputDir, distroName+"-wsl.tar")
	outFile, err := os.Create(tarPath)
	if err != nil {
		logError(fmt.Sprintf("Failed to create tar file: %v", err))
		cleanupContainer(containerName)
		os.Exit(1)
	}

	tarWriter := tar.NewWriter(outFile)
	exportCmd := exec.Command("docker", "export", containerName)
	stdout, err := exportCmd.StdoutPipe()
	if err != nil {
		logError(fmt.Sprintf("Failed to get stdout from docker export: %v", err))
		cleanupContainer(containerName)
		os.Exit(1)
	}

	if err := exportCmd.Start(); err != nil {
		logError(fmt.Sprintf("Failed to start docker export: %v", err))
		cleanupContainer(containerName)
		os.Exit(1)
	}

	tarReader := tar.NewReader(stdout)
	for {
		header, err := tarReader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			logError(fmt.Sprintf("Error reading tar stream: %v", err))
			cleanupContainer(containerName)
			os.Exit(1)
		}

		// Skip "etc/resolv.conf"
		if header.Name == "etc/resolv.conf" {
			continue
		}

		if err := tarWriter.WriteHeader(header); err != nil {
			logError(fmt.Sprintf("Error writing tar header: %v", err))
			cleanupContainer(containerName)
			os.Exit(1)
		}

		if _, err := io.Copy(tarWriter, tarReader); err != nil {
			logError(fmt.Sprintf("Error writing file data to tar: %v", err))
			cleanupContainer(containerName)
			os.Exit(1)
		}
	}

	if err := exportCmd.Wait(); err != nil {
		logError(fmt.Sprintf("Docker export command failed: %v", err))
		cleanupContainer(containerName)
		os.Exit(1)
	}

	if err := tarWriter.Close(); err != nil {
		logError(fmt.Sprintf("Error closing tar writer: %v", err))
		cleanupContainer(containerName)
		os.Exit(1)
	}
	if err := outFile.Close(); err != nil {
		logError(fmt.Sprintf("Error closing tar file: %v", err))
		cleanupContainer(containerName)
		os.Exit(1)
	}

	// Step 3: Clean up container.
	logStep("Cleaning up container...")
	cleanupContainer(containerName)

	// Step 4: Rename the tarball to a .wsl file.
	logStep("Finalizing WSL image...")
	wslPath := filepath.Join(absOutputDir, distroName+".wsl")
	if err := os.Rename(tarPath, wslPath); err != nil {
		logError(fmt.Sprintf("Failed to rename file: %v", err))
		os.Exit(1)
	}

	logSuccess(fmt.Sprintf("===== %s WSL distro build completed successfully =====", distroName))
	logResult(fmt.Sprintf("Output file: %s", wslPath))
}

func cleanupContainer(containerName string) {
	exec.Command("docker", "rm", containerName).Run()
}

func logInfo(message string) {
	fmt.Printf("\033[1;34m%s\033[0m\n", message)
}

func logStep(message string) {
	fmt.Printf("\033[1;33m>> %s\033[0m\n", message)
}

func logSuccess(message string) {
	fmt.Printf("\033[1;32m%s\033[0m\n", message)
}

func logError(message string) {
	fmt.Printf("\033[1;31m%s\033[0m\n", message)
}

func logResult(message string) {
	fmt.Printf("\033[1;36m%s\033[0m\n", message)
}
