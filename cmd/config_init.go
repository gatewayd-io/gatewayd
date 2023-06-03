package cmd

import (
	"os"

	"github.com/gatewayd-io/gatewayd/config"
	"github.com/spf13/cobra"
)

var (
	force           bool
	filePermissions os.FileMode = 0o644
)

// configInitCmd represents the plugin init command.
var configInitCmd = &cobra.Command{
	Use:   "init",
	Short: "Create or overwrite the GatewayD global config",
	Run: func(cmd *cobra.Command, args []string) {
		generateConfig(cmd, Global, globalConfigFile, force)
	},
}

func init() {
	configCmd.AddCommand(configInitCmd)

	configInitCmd.Flags().BoolVarP(
		&force, "force", "f", false, "Force overwrite of existing config file")
	configInitCmd.Flags().StringVarP(
		&globalConfigFile, // Already exists in run.go
		"config", "c", config.GetDefaultConfigFilePath(config.GlobalConfigFilename),
		"Global config file")
}
