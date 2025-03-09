package cmd

import (
	"fmt"
	"github.com/spf13/cobra"
	"github.com/wsl-images/wslb/internal/version"
)

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print the version information",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println(version.GetVersionInfo())
	},
}

func init() {
	rootCmd.AddCommand(versionCmd)
}
