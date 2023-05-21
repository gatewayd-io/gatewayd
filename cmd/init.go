/*
Copyright © 2023 NAME HERE <EMAIL ADDRESS>
*/
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
	configFile      string
	filePermissions os.FileMode = 0o644
)

// initCmd represents the init command.
var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Create or overwrite the GatewayD global config",
	Run: func(cmd *cobra.Command, args []string) {
		// Create a new config object and load the defaults.
		conf := &config.Config{
			GlobalKoanf: koanf.New("."),
		}
		conf.LoadDefaults(context.Background())

		// Marshal the global config to YAML.
		globalCfg, err := conf.GlobalKoanf.Marshal(yaml.Parser())
		if err != nil {
			log.Fatal(err)
		}

		// Check if the config file already exists and if we should overwrite it.
		exists := false
		if _, err := os.Stat(configFile); err == nil && !force {
			log.Fatal("Config file already exists. Use --force to overwrite.")
		} else if err == nil {
			exists = true
		}

		// Create or overwrite the global config file.
		if err := os.WriteFile(configFile, globalCfg, filePermissions); err != nil {
			log.Fatal(err)
		}

		verb := "created"
		if exists && force {
			verb = "overwritten"
		}
		log.Printf("Config file '%s' was %s successfully.", configFile, verb)
	},
}

func init() {
	configCmd.AddCommand(initCmd)

	initCmd.Flags().BoolVarP(
		&force, "force", "f", false, "Force overwrite of existing config file")
	initCmd.Flags().StringVarP(
		&configFile, "config", "c", "gatewayd.yaml", "Config file to write to")
}
