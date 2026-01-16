package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

var (
	Version   = "dev"
	GitCommit = "unknown"
	BuildTime = "unknown"
)

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print version information",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("phpx %s (commit: %s, built: %s)\n", Version, GitCommit, BuildTime)
	},
}

func init() {
	rootCmd.AddCommand(versionCmd)
}
