package cmd

import (
	"log"

	"github.com/spf13/cobra"
)

// configCmd represents the config command.
var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Manage GatewayD global configuration",
	Run: func(cmd *cobra.Command, _ []string) {
		if err := cmd.Help(); err != nil {
			log.New(cmd.OutOrStdout(), "", 0).Fatal(err)
		}
	},
}

func init() {
	rootCmd.AddCommand(configCmd)
}
