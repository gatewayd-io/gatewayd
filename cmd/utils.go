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

type (
	configFileType string
)

var (
	Global configFileType = "global"
	Plugin configFileType = "plugin"
)

// generateConfig generates a config file of the given type.
func generateConfig(cmd *cobra.Command, fileType configFileType, configFile string, force bool) {
	logger := log.New(cmd.OutOrStdout(), "", 0)

	// Create a new config object and load the defaults.
	conf := &config.Config{
		GlobalKoanf: koanf.New("."),
		PluginKoanf: koanf.New("."),
	}
	conf.LoadDefaults(context.TODO())

	// Marshal the config file to YAML.
	var k *koanf.Koanf
	if fileType == Global {
		k = conf.GlobalKoanf
	} else if fileType == Plugin {
		k = conf.PluginKoanf
	} else {
		logger.Fatal("Invalid config file type")
	}
	cfg, err := k.Marshal(yaml.Parser())
	if err != nil {
		logger.Fatal(err)
	}

	// Check if the config file already exists and if we should overwrite it.
	exists := false
	if _, err := os.Stat(configFile); err == nil && !force {
		logger.Fatal("Config file already exists. Use --force to overwrite.")
	} else if err == nil {
		exists = true
	}

	// Create or overwrite the config file.
	if err := os.WriteFile(configFile, cfg, filePermissions); err != nil {
		logger.Fatal(err)
	}

	verb := "created"
	if exists && force {
		verb = "overwritten"
	}
	logger.Printf("Config file '%s' was %s successfully.", configFile, verb)
}
