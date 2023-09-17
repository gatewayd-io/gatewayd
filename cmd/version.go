package cmd

import (
	"github.com/gatewayd-io/gatewayd/config"
	"github.com/spf13/cobra"
)

// versionCmd represents the version command.
var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Show version information",
	Run: func(cmd *cobra.Command, _ []string) {
		cmd.Println(config.VersionInfo())
	},
}

func init() {
	rootCmd.AddCommand(versionCmd)
}
