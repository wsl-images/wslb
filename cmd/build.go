package cmd

import (
	"github.com/spf13/cobra"
	"github.com/wsl-images/wslb/internal/build"
)

var outputDir string

var buildCmd = &cobra.Command{
	Use:   "build [docker-image]",
	Short: "Build a WSL distribution from a Docker image",
	Args:  cobra.MinimumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		dockerImage := args[0]
		build.BuildWSL(dockerImage, outputDir, true)
	},
}

func init() {
	rootCmd.AddCommand(buildCmd)
	buildCmd.Flags().StringVarP(&outputDir, "output", "o", ".", "Output directory for the WSL file")
}
