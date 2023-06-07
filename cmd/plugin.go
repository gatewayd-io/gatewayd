package cmd

import (
	"log"

	"github.com/spf13/cobra"
)

// pluginCmd represents the plugin command.
var pluginCmd = &cobra.Command{
	Use:   "plugin",
	Short: "Manage plugins and their configuration",
	Run: func(cmd *cobra.Command, args []string) {
		if err := cmd.Help(); err != nil {
			log.New(cmd.OutOrStdout(), "", 0).Fatal(err)
		}
	},
}

func init() {
	rootCmd.AddCommand(pluginCmd)
}
