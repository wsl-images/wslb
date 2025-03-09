package cmd

import (
	"github.com/spf13/cobra"
	"github.com/wsl-images/wslb/internal/wsl"
)

var shutdownCmd = &cobra.Command{
	Use:   "shutdown",
	Short: "Shutdown all WSL distributions",
	Long:  `Immediately terminates all running distributions and the WSL 2 lightweight utility virtual machine.`,
	Run: func(cmd *cobra.Command, args []string) {
		wsl.ShutdownAll()
	},
}

func init() {
	rootCmd.AddCommand(shutdownCmd)
}
