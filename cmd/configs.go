package cmd

import (
	"context"
	"encoding/json"
	"log"
	"os"

	"github.com/gatewayd-io/gatewayd/config"
	gerr "github.com/gatewayd-io/gatewayd/errors"
	jsonSchemaGenerator "github.com/invopop/jsonschema"
	"github.com/knadh/koanf"
	koanfJson "github.com/knadh/koanf/parsers/json"
	"github.com/knadh/koanf/parsers/yaml"
	jsonSchemaV5 "github.com/santhosh-tekuri/jsonschema/v5"
	"github.com/spf13/cobra"
)

// generateConfig generates a config file of the given type.
func generateConfig(
	cmd *cobra.Command, fileType configFileType, configFile string, forceRewriteFile bool,
) {
	logger := log.New(cmd.OutOrStdout(), "", 0)

	// Create a new config object and load the defaults.
	conf := &config.Config{
		GlobalKoanf: koanf.New("."),
		PluginKoanf: koanf.New("."),
	}
	if err := conf.LoadDefaults(context.TODO()); err != nil {
		logger.Fatal(err)
	}

	// Marshal the config file to YAML.
	var konfig *koanf.Koanf
	switch fileType {
	case Global:
		konfig = conf.GlobalKoanf
	case Plugins:
		konfig = conf.PluginKoanf
	default:
		logger.Fatal("Invalid config file type")
	}
	cfg, err := konfig.Marshal(yaml.Parser())
	if err != nil {
		logger.Fatal(err)
	}

	// Check if the config file already exists and if we should overwrite it.
	exists := false
	if _, err := os.Stat(configFile); err == nil && !forceRewriteFile {
		logger.Fatal(
			"Config file already exists. Use --force to overwrite or choose a different filename.")
	} else if err == nil {
		exists = true
	}

	// Create or overwrite the config file.
	if err := os.WriteFile(configFile, cfg, FilePermissions); err != nil {
		logger.Fatal(err)
	}

	verb := "created"
	if exists && forceRewriteFile {
		verb = "overwritten"
	}
	cmd.Printf("Config file '%s' was %s successfully.", configFile, verb)
}

// lintConfig lints the given config file of the given type.
func lintConfig(fileType configFileType, configFile string) *gerr.GatewayDError {
	// Load the config file and check it for errors.
	var conf *config.Config
	switch fileType {
	case Global:
		conf = config.NewConfig(context.TODO(), config.Config{GlobalConfigFile: configFile})
		if err := conf.LoadDefaults(context.TODO()); err != nil {
			return err
		}
		if err := conf.LoadGlobalConfigFile(context.TODO()); err != nil {
			return err
		}
		if err := conf.ConvertKeysToLowercase(context.TODO()); err != nil {
			return err
		}
		if err := conf.UnmarshalGlobalConfig(context.TODO()); err != nil {
			return err
		}
	case Plugins:
		conf = config.NewConfig(context.TODO(), config.Config{PluginConfigFile: configFile})
		if err := conf.LoadDefaults(context.TODO()); err != nil {
			return err
		}
		if err := conf.LoadPluginConfigFile(context.TODO()); err != nil {
			return err
		}
		if err := conf.UnmarshalPluginConfig(context.TODO()); err != nil {
			return err
		}
	default:
		return gerr.ErrLintingFailed
	}

	// Marshal the config to JSON.
	var jsonData []byte
	var err error
	switch fileType {
	case Global:
		jsonData, err = conf.GlobalKoanf.Marshal(koanfJson.Parser())
	case Plugins:
		jsonData, err = conf.PluginKoanf.Marshal(koanfJson.Parser())
	default:
		return gerr.ErrLintingFailed
	}
	if err != nil {
		return gerr.ErrLintingFailed.Wrap(err)
	}

	// Unmarshal the JSON data into a map.
	var jsonBytes map[string]interface{}
	err = json.Unmarshal(jsonData, &jsonBytes)
	if err != nil {
		return gerr.ErrLintingFailed.Wrap(err)
	}

	// Generate a JSON schema from the config struct.
	var generatedSchema *jsonSchemaGenerator.Schema
	switch fileType {
	case Global:
		generatedSchema = jsonSchemaGenerator.Reflect(&config.GlobalConfig{})
	case Plugins:
		generatedSchema = jsonSchemaGenerator.Reflect(&config.PluginConfig{})
	default:
		return gerr.ErrLintingFailed
	}

	// Marshal the schema to JSON.
	schemaBytes, err := json.Marshal(generatedSchema)
	if err != nil {
		return gerr.ErrLintingFailed.Wrap(err)
	}

	// Compile the schema for validation.
	schema, err := jsonSchemaV5.CompileString("", string(schemaBytes))
	if err != nil {
		return gerr.ErrLintingFailed.Wrap(err)
	}

	// Validate the config against the schema.
	err = schema.Validate(jsonBytes)
	if err != nil {
		return gerr.ErrLintingFailed.Wrap(err)
	}

	return nil
}
