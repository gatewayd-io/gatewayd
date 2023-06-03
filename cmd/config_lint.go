package cmd

import (
	"context"
	"encoding/json"
	"log"

	"github.com/gatewayd-io/gatewayd/config"
	jsonSchemaGenerator "github.com/invopop/jsonschema"
	koanfJson "github.com/knadh/koanf/parsers/json"
	jsonSchemaV5 "github.com/santhosh-tekuri/jsonschema/v5"
	"github.com/spf13/cobra"
)

// lintCmd represents the lint command.
var lintCmd = &cobra.Command{
	Use:   "lint",
	Short: "Lint the GatewayD global config",
	Run: func(cmd *cobra.Command, args []string) {
		logger := log.New(cmd.OutOrStdout(), "", 0)

		// Load the global configuration file and check it for errors.
		conf := config.NewConfig(context.TODO(), globalConfigFile, "")
		conf.LoadDefaults(context.TODO())
		conf.LoadGlobalConfigFile(context.TODO())
		conf.UnmarshalGlobalConfig(context.TODO())

		// Marshal the global config to JSON.
		jsonData, err := conf.GlobalKoanf.Marshal(koanfJson.Parser())
		if err != nil {
			logger.Fatal("Error marshalling global config to JSON:\n", err)
		}

		// Unmarshal the JSON data into a map.
		var jsonBytes map[string]interface{}
		err = json.Unmarshal(jsonData, &jsonBytes)
		if err != nil {
			logger.Fatal("Error unmarshalling schema to JSON:\n", err)
		}

		// Generate a JSON schema from the global config struct.
		generatedSchema := jsonSchemaGenerator.Reflect(&config.GlobalConfig{})

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

		// Validate the global config against the schema.
		err = schema.Validate(jsonBytes)
		if err != nil {
			logger.Fatal("Error validating global config:\n", err)
		}

		logger.Print("Global config is valid")
	},
}

func init() {
	configCmd.AddCommand(lintCmd)

	lintCmd.Flags().StringVarP(
		&globalConfigFile, // Already exists in run.go
		"config", "c", config.GetDefaultConfigFilePath(config.GlobalConfigFilename),
		"Global config file")
}
