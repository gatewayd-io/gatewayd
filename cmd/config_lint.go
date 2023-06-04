package cmd

import (
	"github.com/gatewayd-io/gatewayd/config"
	"github.com/spf13/cobra"
)

// configLintCmd represents the config lint command.
var configLintCmd = &cobra.Command{
	Use:   "lint",
	Short: "Lint the GatewayD global config",
	Run: func(cmd *cobra.Command, args []string) {
		lintConfig(cmd, Global, globalConfigFile)
	},
}

func init() {
	configCmd.AddCommand(configLintCmd)

	configLintCmd.Flags().StringVarP(
		&globalConfigFile, // Already exists in run.go
		"config", "c", config.GetDefaultConfigFilePath(config.GlobalConfigFilename),
		"Global config file")
}
