package plugin

import (
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"unicode"

	"github.com/cybercyst/go-scaffold/pkg/scaffold"
	gerr "github.com/gatewayd-io/gatewayd/errors"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
	"gopkg.in/yaml.v3"
)

const (
	FolderPermissions os.FileMode = 0o755
	FilePermissions   os.FileMode = 0o644
)

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
func Scaffold(inputFile string, outputDir string) ([]string, error) {
	tempDir, err := os.MkdirTemp("", "gatewayd-plugin-template")
	if err != nil {
		return nil, gerr.ErrFailedToScaffoldPlugin.Wrap(err)
	}

	defer func() {
		os.RemoveAll(tempDir)
	}()

	// Copy the embedded directory and its contents as "go-scaffold" library
	// only accepts files on os filesystem
	err = fs.WalkDir(pluginTemplate, pluginTemplateRootDir, func(path string, dir fs.DirEntry, err error) error {
		if err != nil {
			return gerr.ErrFailedToCopyEmbeddedFiles.Wrap(err)
		}

		relativePath, err := filepath.Rel(pluginTemplateRootDir, path)
		if err != nil {
			return gerr.ErrFailedToCopyEmbeddedFiles.Wrap(err)
		}
		destPath := filepath.Join(tempDir, relativePath)

		if dir.IsDir() {
			return os.MkdirAll(destPath, FolderPermissions)
		}

		fileContent, err := pluginTemplate.ReadFile(path)
		if err != nil {
			return gerr.ErrFailedToCopyEmbeddedFiles.Wrap(err)
		}

		if err := os.MkdirAll(filepath.Dir(destPath), FolderPermissions); err != nil {
			return gerr.ErrFailedToCopyEmbeddedFiles.Wrap(err)
		}

		return os.WriteFile(destPath, fileContent, FilePermissions)
	})
	if err != nil {
		return nil, gerr.ErrFailedToScaffoldPlugin.Wrap(err)
	}

	input, err := readScaffoldInputFile(inputFile)
	if err != nil {
		return nil, gerr.ErrFailedToScaffoldPlugin.Wrap(err)
	}

	var pluginName string
	if url, ok := input["remote_url"].(string); ok {
		pluginName = getLastSegment(url)
	} else {
		return nil, gerr.ErrFailedToScaffoldPlugin.Wrap(err)
	}

	input["plugin_name"] = pluginName
	input["pascal_case_plugin_name"] = toPascalCase(pluginName)
	// set go_mod as template variable because the go.mod file is not embedable.
	// so we would name it as {{ go_mod }} and rename it to go.mod when scaffolding to circumvent this issue
	input["go_mod"] = "go.mod"

	template, err := scaffold.Download(tempDir)
	if err != nil {
		return nil, gerr.ErrFailedToScaffoldPlugin.Wrap(err)
	}

	metadata, err := scaffold.Generate(template, &input, outputDir)
	if err != nil {
		return nil, gerr.ErrFailedToScaffoldPlugin.Wrap(err)
	}

	metadataYaml, err := yaml.Marshal(metadata)
	if err != nil {
		return nil, gerr.ErrFailedToScaffoldPlugin.Wrap(err)
	}

	err = os.WriteFile(filepath.Join(outputDir, ".metadata.yaml"), metadataYaml, FilePermissions)
	if err != nil {
		return nil, gerr.ErrFailedToScaffoldPlugin.Wrap(err)
	}

	return *metadata.CreatedFiles, nil
}

// readScaffoldInputFile reads the template input file in YAML format and
// returns a map containing the parsed data.
//
// This function opens the provided input file path, reads its contents, and unmarshals
// the YAML data into a Go map[string]interface{}. It returns the parsed map and any encountered error.
func readScaffoldInputFile(inputFilePath string) (map[string]interface{}, error) {
	inputBytes, err := os.ReadFile(inputFilePath)
	if err != nil {
		return nil, gerr.ErrFailedToReadPluginScaffoldInputFile.Wrap(err)
	}

	inputJSON := make(map[string]interface{})
	err = yaml.Unmarshal(inputBytes, &inputJSON)
	if err != nil {
		return nil, gerr.ErrFailedToReadPluginScaffoldInputFile.Wrap(err)
	}

	return inputJSON, nil
}

// toPascalCase converts a string to PascalCase format, suitable for use as a plugin class name in templates.
func toPascalCase(input string) string {
	trimmed := strings.TrimSpace(input)
	words := strings.FieldsFunc(trimmed, func(r rune) bool {
		return r == ' ' || r == '-'
	})
	var pascalCase string
	for _, word := range words {
		caser := cases.Title(language.English)
		pascalCase += caser.String(word)
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
