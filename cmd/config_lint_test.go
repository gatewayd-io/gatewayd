package cmd

import (
	"fmt"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_configLintCmd(t *testing.T) {
	globalTestConfigFile := "./test_global_configLintCmd.yaml"
	// Test configInitCmd.
	output, err := executeCommandC(rootCmd, "config", "init", "-c", globalTestConfigFile)
	require.NoError(t, err, "configInitCmd should not return an error")
	assert.Equal(t,
		fmt.Sprintf("Config file '%s' was created successfully.", globalTestConfigFile),
		output,
		"configInitCmd should print the correct output")
	// Check that the config file was created.
	assert.FileExists(t, globalTestConfigFile, "configInitCmd should create a config file")

	// Test configLintCmd.
	output, err = executeCommandC(rootCmd, "config", "lint", "-c", globalTestConfigFile)
	require.NoError(t, err, "configLintCmd should not return an error")
	assert.Equal(t,
		"global config is valid\n",
		output,
		"configLintCmd should print the correct output")

	// Clean up.
	err = os.Remove(globalTestConfigFile)
	assert.Nil(t, err)
}
