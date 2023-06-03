package cmd

import (
	"context"
	"log"
	"os"

	"github.com/gatewayd-io/gatewayd/config"
	"github.com/knadh/koanf"
	"github.com/knadh/koanf/parsers/yaml"
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
		logger := log.New(cmd.OutOrStdout(), "", 0)

		// Create a new config object and load the defaults.
		conf := &config.Config{
			GlobalKoanf: koanf.New("."),
		}
		conf.LoadDefaults(context.TODO())

		// Marshal the global config to YAML.
		globalCfg, err := conf.GlobalKoanf.Marshal(yaml.Parser())
		if err != nil {
			logger.Fatal(err)
		}

		// Check if the config file already exists and if we should overwrite it.
		exists := false
		if _, err := os.Stat(globalConfigFile); err == nil && !force {
			logger.Fatal("Config file already exists. Use --force to overwrite.")
		} else if err == nil {
			exists = true
		}

		// Create or overwrite the global config file.
		if err := os.WriteFile(globalConfigFile, globalCfg, filePermissions); err != nil {
			logger.Fatal(err)
		}

		verb := "created"
		if exists && force {
			verb = "overwritten"
		}
		logger.Printf("Config file '%s' was %s successfully.", globalConfigFile, verb)
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
