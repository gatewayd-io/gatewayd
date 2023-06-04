package cmd

import (
	"github.com/gatewayd-io/gatewayd/config"
	"github.com/spf13/cobra"
)

// pluginInitCmd represents the plugin init command.
var pluginInitCmd = &cobra.Command{
	Use:   "init",
	Short: "Create or overwrite the GatewayD plugins config",
	Run: func(cmd *cobra.Command, args []string) {
		generateConfig(cmd, Plugins, pluginConfigFile, force)
	},
}

func init() {
	pluginCmd.AddCommand(pluginInitCmd)

	pluginInitCmd.Flags().BoolVarP(
		&force, "force", "f", false, "Force overwrite of existing config file")
	pluginInitCmd.Flags().StringVarP(
		&pluginConfigFile, // Already exists in run.go
		"plugin-config", "p", config.GetDefaultConfigFilePath(config.PluginsConfigFilename),
		"Plugin config file")
}
