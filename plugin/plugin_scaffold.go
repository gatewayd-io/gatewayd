package plugin

import (
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"unicode"

	"github.com/cybercyst/go-scaffold/pkg/scaffold"
	"gopkg.in/yaml.v2"
)

var templatePath = filepath.Join("plugin", "template")

// Scaffold generates a gatewayd plugin based on the provided input file
// and stores the generated files in the specified output directory.
// The expected input file should be a YAML file containing the following fields:
//
// ```yaml
// remote_url: https://github.com/gatewayd-io/test-gatewayd-plugin
// version: 0.1
// description: This is test plugin
// license: MIT
// authors:
//   - GatewayD Team
//
// ```
func Scaffold(inputFile string, outputDir string) ([]string, error) {
	tempDir, err := os.MkdirTemp("", "gatewayd-plugin-template")
	if err != nil {
		return nil, err
	}

	defer func() {
		os.RemoveAll(tempDir)
	}()

	// Copy the embedded directory and its contents as "go-scaffold" library only accepts files on os filesystem
	err = fs.WalkDir(pluginTemplate, pluginTemplateRootDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		relativePath, err := filepath.Rel(pluginTemplateRootDir, path)
		if err != nil {
			return err
		}
		destPath := filepath.Join(tempDir, relativePath)

		if d.IsDir() {
			return os.MkdirAll(destPath, 0755)
		}

		fileContent, err := pluginTemplate.ReadFile(path)
		if err != nil {
			return err
		}

		if err := os.MkdirAll(filepath.Dir(destPath), 0755); err != nil {
			return err
		}

		return os.WriteFile(destPath, fileContent, 0644)
	})

	if err != nil {
		return nil, err
	}

	input, err := readTemplateInput(inputFile)
	if err != nil {
		return nil, err
	}
	pluginName := getLastSegment(input["remote_url"].(string))
	input["plugin_name"] = pluginName
	input["pascal_case_plugin_name"] = toPascalCase(pluginName)
	// set go_mod as template variable because the go.mod file is not embedable.
	// so we would name it as {{ go_mod }} and rename it to go.mod when scaffolding to circumvent this issue
	input["go_mod"] = "go.mod"

	template, err := scaffold.Download(tempDir)
	if err != nil {
		return nil, err
	}

	metadata, err := scaffold.Generate(template, &input, outputDir)
	if err != nil {
		return nil, err
	}

	metadataYaml, err := yaml.Marshal(metadata)
	if err != nil {
		return nil, err
	}

	err = os.WriteFile(filepath.Join(outputDir, ".metadata.yaml"), metadataYaml, 0644)
	if err != nil {
		return nil, err
	}

	return *metadata.CreatedFiles, err
}

// Reads the template input file in YAML format and returns a map containing the parsed data.
//
// This function opens the provided input file path, reads its contents, and unmarshals
// the YAML data into a Go map[string]interface{}. It returns the parsed map and any encountered error.
func readTemplateInput(inputFilePath string) (map[string]interface{}, error) {
	inputBytes, err := os.ReadFile(inputFilePath)
	if err != nil {
		return nil, err
	}

	inputJson := make(map[string]interface{})
	err = yaml.Unmarshal(inputBytes, &inputJson)
	if err != nil {
		return nil, err
	}

	return inputJson, nil
}

// toPascalCase converts a string to PascalCase format, suitable for use as a plugin class name in templates.
func toPascalCase(input string) string {
	trimmed := strings.TrimSpace(input)
	words := strings.FieldsFunc(trimmed, func(r rune) bool {
		return r == ' ' || r == '-'
	})
	var pascalCase string
	for _, word := range words {
		pascalCase += strings.Title(word)
	}

	// Remove any non-alphanumeric characters except underscores
	cleaned := strings.Map(func(r rune) rune {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			return r
		}
		if r == '_' {
			return r
		}
		return -1
	}, pascalCase)

	return cleaned
}

// getLastSegment extracts the last path segment from a given string, assuming a forward slash ('/') separator.
func getLastSegment(input string) string {
	parts := strings.Split(input, "/")
	if len(parts) == 0 {
		return ""
	}
	return parts[len(parts)-1]
}
