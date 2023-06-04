package cmd

import (
	"github.com/gatewayd-io/gatewayd/config"
	"github.com/spf13/cobra"
)

// pluginLintCmd represents the plugin lint command.
var pluginLintCmd = &cobra.Command{
	Use:   "lint",
	Short: "Lint the GatewayD plugins config",
	Run: func(cmd *cobra.Command, args []string) {
		lintConfig(cmd, Plugins, pluginConfigFile)
	},
}

func init() {
	pluginCmd.AddCommand(pluginLintCmd)

	pluginLintCmd.Flags().StringVarP(
		&pluginConfigFile, // Already exists in run.go
		"plugin-config", "p", config.GetDefaultConfigFilePath(config.PluginsConfigFilename),
		"Plugin config file")
}
