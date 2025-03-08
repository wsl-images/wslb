package cmd

import (
	"github.com/spf13/cobra"
	"github.com/wsl-images/wslb/internal/wsl"
)

var (
	allFlag     bool
	runningFlag bool
	quietFlag   bool
	verboseFlag bool
	onlineFlag  bool
)

var lsCmd = &cobra.Command{
	Use:     "ls",
	Aliases: []string{"list"},
	Short:   "List WSL distributions",
	Long:    `Lists WSL distributions with various filtering and formatting options.`,
	Run: func(cmd *cobra.Command, args []string) {
		wsl.ListDistributions(allFlag, runningFlag, quietFlag, verboseFlag, onlineFlag)
	},
}

func init() {
	rootCmd.AddCommand(lsCmd)

	lsCmd.Flags().BoolVar(&allFlag, "all", false, "List all distributions, including distributions that are currently being installed or uninstalled")
	lsCmd.Flags().BoolVar(&runningFlag, "running", false, "List only distributions that are currently running")
	lsCmd.Flags().BoolVarP(&quietFlag, "quiet", "q", false, "Only show distribution names")
	lsCmd.Flags().BoolVarP(&verboseFlag, "verbose", "v", false, "Show detailed information about all distributions")
	lsCmd.Flags().BoolVarP(&onlineFlag, "online", "o", false, "Displays a list of available distributions for install")
}
