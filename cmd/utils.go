package cmd

import (
	"context"
	"encoding/json"
	"log"
	"os"

	"github.com/gatewayd-io/gatewayd/config"
	jsonSchemaGenerator "github.com/invopop/jsonschema"
	"github.com/knadh/koanf"
	koanfJson "github.com/knadh/koanf/parsers/json"
	"github.com/knadh/koanf/parsers/yaml"
	jsonSchemaV5 "github.com/santhosh-tekuri/jsonschema/v5"
	"github.com/spf13/cobra"
)

type (
	configFileType string
)

const (
	FilePermissions     os.FileMode = 0o644
	ExecFilePermissions os.FileMode = 0o755
	ExecFileMask        os.FileMode = 0o111
	MaxFileSize         int64       = 1024 * 1024 * 100 // 10MB
)

var (
	Global  configFileType = "global"
	Plugins configFileType = "plugins"

	DSN string = "https://e22f42dbb3e0433fbd9ea32453faa598@o4504550475038720.ingest.sentry.io/4504550481723392" //nolint:lll
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
	if _, err := os.Stat(configFile); err == nil && !force {
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
	if exists && force {
		verb = "overwritten"
	}
	logger.Printf("Config file '%s' was %s successfully.", configFile, verb)
}

func lintConfig(cmd *cobra.Command, fileType configFileType, configFile string) {
	logger := log.New(cmd.OutOrStdout(), "", 0)

	// Load the config file and check it for errors.
	var conf *config.Config
	switch fileType {
	case Global:
		conf = config.NewConfig(context.TODO(), configFile, "")
		conf.LoadDefaults(context.TODO())
		conf.LoadGlobalConfigFile(context.TODO())
		conf.UnmarshalGlobalConfig(context.TODO())
	case Plugins:
		conf = config.NewConfig(context.TODO(), "", configFile)
		conf.LoadDefaults(context.TODO())
		conf.LoadPluginConfigFile(context.TODO())
		conf.UnmarshalPluginConfig(context.TODO())
	default:
		logger.Fatal("Invalid config file type")
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
		logger.Fatal("Invalid config file type")
	}
	if err != nil {
		logger.Fatalf("Error marshalling %s config to JSON: %s\n", string(fileType), err)
	}

	// Unmarshal the JSON data into a map.
	var jsonBytes map[string]interface{}
	err = json.Unmarshal(jsonData, &jsonBytes)
	if err != nil {
		logger.Fatal("Error unmarshalling schema to JSON:\n", err)
	}

	// Generate a JSON schema from the config struct.
	var generatedSchema *jsonSchemaGenerator.Schema
	switch fileType {
	case Global:
		generatedSchema = jsonSchemaGenerator.Reflect(&config.GlobalConfig{})
	case Plugins:
		generatedSchema = jsonSchemaGenerator.Reflect(&config.PluginConfig{})
	default:
		logger.Fatal("Invalid config file type")
	}

	// Marshal the schema to JSON.
	schemaBytes, err := json.Marshal(generatedSchema)
	if err != nil {
		logger.Fatal("Error marshalling schema to JSON:\n", err)
	}

	// Compile the schema for validation.
	schema, err := jsonSchemaV5.CompileString("", string(schemaBytes))
	if err != nil {
		logger.Fatal("Error compiling schema:\n", err)
	}

	// Validate the config against the schema.
	err = schema.Validate(jsonBytes)
	if err != nil {
		logger.Fatalf("Error validating %s config: %s\n", string(fileType), err)
	}

	logger.Printf("%s config is valid\n", fileType)
}
