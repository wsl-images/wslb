package cmd

import (
	"github.com/spf13/cobra"
	"github.com/wsl-images/wslb/internal/wsl"
)

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show the status of Windows Subsystem for Linux",
	Long:  `Show the status of Windows Subsystem for Linux.`,
	Run: func(cmd *cobra.Command, args []string) {
		wsl.ShowStatus()
	},
}

func init() {
	rootCmd.AddCommand(statusCmd)
}
