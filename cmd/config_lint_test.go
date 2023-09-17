package cmd

import (
	"fmt"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_configLintCmd(t *testing.T) {
	// Reset globalConfigFile to avoid conflicts with other tests.
	globalConfigFile = "./test_config.yaml"

	// Test configInitCmd.
	output, err := executeCommandC(rootCmd, "config", "init", "-c", globalConfigFile)
	assert.NoError(t, err, "configInitCmd should not return an error")
	assert.Equal(t,
		fmt.Sprintf("Config file '%s' was created successfully.", globalConfigFile),
		output,
		"configInitCmd should print the correct output")
	// Check that the config file was created.
	assert.FileExists(t, globalConfigFile, "configInitCmd should create a config file")

	// Test configLintCmd.
	output, err = executeCommandC(rootCmd, "config", "lint", "-c", globalConfigFile)
	assert.NoError(t, err, "configLintCmd should not return an error")
	assert.Equal(t,
		"global config is valid\n",
		output,
		"configLintCmd should print the correct output")

	// Clean up.
	err = os.Remove(globalConfigFile)
	assert.NoError(t, err)
}
