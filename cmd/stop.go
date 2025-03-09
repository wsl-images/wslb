package cmd

import (
	"github.com/spf13/cobra"
	"github.com/wsl-images/wslb/internal/wsl"
)

var stopCmd = &cobra.Command{
	Use:   "stop <Distro>",
	Short: "Terminate a WSL distribution",
	Long:  `Terminates the specified WSL distribution.`,
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		distroName := args[0]
		wsl.TerminateDistribution(distroName)
	},
}

func init() {
	rootCmd.AddCommand(stopCmd)
}
