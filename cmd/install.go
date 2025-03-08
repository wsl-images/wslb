package cmd

import (
	"github.com/wsl-images/wslb/internal/build"
	"github.com/wsl-images/wslb/internal/logger"
	"github.com/wsl-images/wslb/internal/wsl"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/spf13/cobra"
)

var (
	customName string
	fileFlag   string
)

var installCmd = &cobra.Command{
	Use:   "install [image URL]",
	Short: "Install a WSL distribution",
	Args:  cobra.MaximumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		if runtime.GOOS == "linux" {
			logger.Fatal("The install command can only be executed on Windows. WSL installation is not supported on Linux.")
		}

		var wslFile string
		tmpDirUsed := false

		if fileFlag != "" {
			wslFile = fileFlag
		} else {
			if len(args) < 1 {
				logger.Error("Either a file must be provided with -f or an image URL must be specified.")
				cmd.Usage()
				os.Exit(1)
			}
			imageURL := args[0]
			tmpDir := "./tmp"
			logger.Info("Installing WSL distribution from ", imageURL)
			build.BuildWSL(imageURL, tmpDir, false)
			distroName := strings.Split(filepath.Base(imageURL), ":")[0]
			wslFile = filepath.Join(tmpDir, distroName+".wsl")
			tmpDirUsed = true
		}

		wsl.InstallDistribution(wslFile, customName)

		if tmpDirUsed {
			_ = os.RemoveAll("./tmp")
		}
	},
}

func init() {
	rootCmd.AddCommand(installCmd)
	installCmd.Flags().StringVarP(&customName, "name", "n", "", "Custom name for installation")
	installCmd.Flags().StringVarP(&fileFlag, "file", "f", "", "Path to a pre-built .wsl file")
}
