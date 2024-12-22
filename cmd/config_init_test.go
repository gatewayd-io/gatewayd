package cmd

import (
	"fmt"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_configInitCmd(t *testing.T) {
	globalTestConfigFile := "./test_global_configInitCmd.yaml"
	// Test configInitCmd.
	output, err := executeCommandC(rootCmd, "config", "init", "-c", globalTestConfigFile)
	require.NoError(t, err, "configInitCmd should not return an error")
	assert.Equal(t,
		fmt.Sprintf("Config file '%s' was created successfully.", App.GlobalConfigFile),
		output,
		"configInitCmd should print the correct output")
	// Check that the config file was created.
	assert.FileExists(t, App.GlobalConfigFile, "configInitCmd should create a config file")

	// Test configInitCmd with the --force flag to overwrite the config file.
	output, err = executeCommandC(rootCmd, "config", "init", "--force", "-c", globalTestConfigFile)
	require.NoError(t, err, "configInitCmd should not return an error")
	assert.Equal(t,
		fmt.Sprintf("Config file '%s' was overwritten successfully.", App.GlobalConfigFile),
		output,
		"configInitCmd should print the correct output")

	// Clean up.
	err = os.Remove(App.GlobalConfigFile)
	assert.Nil(t, err)
}
