package main

import (
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

func main() {
	// Define the main command
	buildCmd := flag.NewFlagSet("build", flag.ExitOnError)
	outputDir := buildCmd.String("o", ".", "Output directory for the WSL file")

	// Check if command is provided
	if len(os.Args) < 2 {
		fmt.Println("Usage: wsl-builder build [docker-image] [-o output-dir]")
		os.Exit(1)
	}

	// Parse the command
	switch os.Args[1] {
	case "build":
		buildCmd.Parse(os.Args[2:])

		// Get the Docker image name
		args := buildCmd.Args()
		if len(args) < 1 {
			fmt.Println("Error: Docker image path is required")
			fmt.Println("Usage: wsl-builder build [docker-image] [-o output-dir]")
			os.Exit(1)
		}

		dockerImage := args[0]
		buildWSL(dockerImage, *outputDir)

	default:
		fmt.Printf("Unknown command: %s\n", os.Args[1])
		fmt.Println("Usage: wsl-builder build [docker-image] [-o output-dir]")
		os.Exit(1)
	}
}

func buildWSL(dockerImage, outputDir string) {
	// Create output directory if it doesn't exist
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		logError(fmt.Sprintf("Failed to create output directory: %v", err))
		os.Exit(1)
	}

	// Extract distro name from docker image
	distroName := strings.Split(filepath.Base(dockerImage), ":")[0]
	containerName := distroName

	logInfo(fmt.Sprintf("===== Starting build for %s WSL distro =====", distroName))

	// Step 1: Run container
	logStep("Running container to prepare filesystem...")
	cmd := exec.Command("docker", "run", "-t", "--name", containerName, dockerImage, "ls", "/")
	if err := cmd.Run(); err != nil {
		logError(fmt.Sprintf("Failed to run container: %v", err))
		cleanupContainer(containerName)
		os.Exit(1)
	}

	// Step 2: Export container to tarball
	logStep("Exporting container to tarball...")
	tarPath := filepath.Join(outputDir, distroName+"-wsl.tar")
	exportCmd := fmt.Sprintf("docker export %s | tar --delete --wildcards \"etc/resolv.conf\" > %s",
		containerName, tarPath)

	cmd = exec.Command("sh", "-c", exportCmd)
	if err := cmd.Run(); err != nil {
		logError(fmt.Sprintf("Failed to export container: %v", err))
		cleanupContainer(containerName)
		os.Exit(1)
	}

	// Step 3: Clean up container
	logStep("Cleaning up container...")
	cleanupContainer(containerName)

	// Step 4: Rename tar to wsl
	logStep("Finalizing WSL image...")
	wslPath := filepath.Join(outputDir, distroName+".wsl")
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

// Logging functions with colors
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
