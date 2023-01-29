package cmd

import (
	"fmt"

	"github.com/gatewayd-io/gatewayd/config"
	"github.com/spf13/cobra"
)

// versionCmd represents the version command.
var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Show version information",
	Run: func(_ *cobra.Command, _ []string) {
		fmt.Println(config.VersionInfo()) //nolint:forbidigo
	},
}

func init() {
	rootCmd.AddCommand(versionCmd)
}
