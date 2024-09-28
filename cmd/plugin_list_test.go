package cmd

import (
	"fmt"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_pluginListCmd(t *testing.T) {
	pluginTestConfigFile := "./test_plugins_pluginListCmd.yaml"
	// Test plugin list command.
	output, err := executeCommandC(rootCmd, "plugin", "init", "-p", pluginTestConfigFile)
	require.NoError(t, err, "plugin init command should not have returned an error")
	assert.Equal(t,
		fmt.Sprintf("Config file '%s' was created successfully.", pluginTestConfigFile),
		output,
		"plugin init command should have returned the correct output")
	assert.FileExists(t, pluginTestConfigFile, "plugin init command should have created a config file")

	output, err = executeCommandC(rootCmd, "plugin", "list", "-p", pluginTestConfigFile)
	require.NoError(t, err, "plugin list command should not have returned an error")
	assert.Equal(t,
		"No plugins found\n",
		output,
		"plugin list command should have returned empty output")

	// Clean up.
	err = os.Remove(pluginTestConfigFile)
	assert.Nil(t, err)
}

func Test_pluginListCmdWithPlugins(t *testing.T) {
	// Test plugin list command.
	// Read the plugin config file from the root directory.
	pluginTestConfigFile := "../gatewayd_plugins.yaml"
	output, err := executeCommandC(rootCmd, "plugin", "list", "-p", pluginTestConfigFile)
	require.NoError(t, err, "plugin list command should not have returned an error")
	assert.Contains(t,
		output,
		"Total plugins: 1",
		"plugin list command should have returned the correct output")
	assert.Contains(t,
		output,
		"Name: gatewayd-plugin-cache",
		"plugin list command should have returned the correct output")
}
