package cmd

import (
	"fmt"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_pluginInstallCmd(t *testing.T) {
	// Create a test config file.
	output, err := executeCommandC(rootCmd, "plugin", "init", "-p", pluginTestConfigFile)
	assert.NoError(t, err, "plugin init should not return an error")
	assert.Equal(t,
		fmt.Sprintf("Config file '%s' was created successfully.", pluginTestConfigFile),
		output,
		"plugin init command should have returned the correct output")
	assert.FileExists(t, pluginTestConfigFile, "plugin init command should have created a config file")

	// Test plugin install command.
	output, err = executeCommandC(
		rootCmd, "plugin", "install",
		"github.com/gatewayd-io/gatewayd-plugin-cache@v0.2.4", "-p", pluginTestConfigFile)
	assert.NoError(t, err, "plugin install should not return an error")
	assert.Contains(t, output, "Downloading https://github.com/gatewayd-io/gatewayd-plugin-cache/releases/download/v0.2.4/gatewayd-plugin-cache-linux-amd64-v0.2.4.tar.gz") //nolint:lll
	assert.Contains(t, output, "Downloading https://github.com/gatewayd-io/gatewayd-plugin-cache/releases/download/v0.2.4/checksums.txt")                                   //nolint:lll
	assert.Contains(t, output, "Download completed successfully")
	assert.Contains(t, output, "Checksum verification passed")
	assert.Contains(t, output, "Plugin binary extracted to plugins/gatewayd-plugin-cache")
	assert.Contains(t, output, "Plugin installed successfully")

	// See if the plugin was actually installed.
	output, err = executeCommandC(rootCmd, "plugin", "list", "-p", pluginTestConfigFile)
	assert.NoError(t, err, "plugin list should not return an error")
	assert.Contains(t, output, "Name: gatewayd-plugin-cache")

	// Clean up.
	assert.NoError(t, os.RemoveAll("plugins/"))
	assert.NoError(t, os.Remove("checksums.txt"))
	assert.NoError(t, os.Remove("gatewayd-plugin-cache-linux-amd64-v0.2.4.tar.gz"))
	assert.NoError(t, os.Remove(pluginTestConfigFile))
}
