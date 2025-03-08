package cmd

import (
	"github.com/spf13/cobra"
	"github.com/wsl-images/wslb/internal/wsl"
)

var rmCmd = &cobra.Command{
	Use:   "rm <Distro>",
	Short: "Remove a WSL distribution",
	Long:  `Unregisters the distribution and deletes the root filesystem.`,
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		distroName := args[0]
		wsl.UnregisterDistribution(distroName)
	},
}

func init() {
	rootCmd.AddCommand(rmCmd)
}
