package cmd

import (
	"fmt"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_pluginInstallCmd(t *testing.T) {
	pluginTestConfigFile := "./test_plugins_pluginInstallCmd.yaml"

	// Create a test plugin config file.
	output, err := executeCommandC(rootCmd, "plugin", "init", "-p", pluginTestConfigFile)
	require.NoError(t, err, "plugin init should not return an error")
	assert.Equal(t,
		fmt.Sprintf("Config file '%s' was created successfully.", pluginTestConfigFile),
		output,
		"plugin init command should have returned the correct output")
	assert.FileExists(t, pluginTestConfigFile, "plugin init command should have created a config file")

	// Pull the plugin archive and install it.
	pluginArchivePath, err = mustPullPlugin()
	require.NoError(t, err, "mustPullPlugin should not return an error")
	assert.FileExists(t, pluginArchivePath, "mustPullPlugin should have downloaded the plugin archive")

	// Test plugin install command.
	output, err = executeCommandC(
		rootCmd, "plugin", "install", "-p", pluginTestConfigFile,
		"--update", "--backup", "--name", "gatewayd-plugin-cache", pluginArchivePath)
	require.NoError(t, err, "plugin install should not return an error")
	assert.Equal(t, output, "Installing plugin from CLI argument\nBackup completed successfully\nPlugin binary extracted to plugins/gatewayd-plugin-cache\nPlugin installed successfully\n") //nolint:lll

	// See if the plugin was actually installed.
	output, err = executeCommandC(rootCmd, "plugin", "list", "-p", pluginTestConfigFile)
	require.NoError(t, err, "plugin list should not return an error")
	assert.Contains(t, output, "Name: gatewayd-plugin-cache")

	// Clean up.
	assert.FileExists(t, "plugins/gatewayd-plugin-cache")
	assert.FileExists(t, fmt.Sprintf("%s.bak", pluginTestConfigFile))
	assert.NoFileExists(t, "gatewayd-plugin-cache-linux-amd64-v0.2.4.tar.gz")
	assert.NoFileExists(t, "checksums.txt")
	assert.NoFileExists(t, "plugins/LICENSE")
	assert.NoFileExists(t, "plugins/README.md")
	assert.NoFileExists(t, "plugins/checksum.txt")
	assert.NoFileExists(t, "plugins/gatewayd_plugin.yaml")

	require.NoError(t, os.RemoveAll("plugins/"))
	require.NoError(t, os.Remove(pluginTestConfigFile))
	require.NoError(t, os.Remove(fmt.Sprintf("%s.bak", pluginTestConfigFile)))
}

func Test_pluginInstallCmdAutomatedNoOverwrite(t *testing.T) {
	pluginTestConfigFile := "./testdata/gatewayd_plugins.yaml"

	// Test plugin install command.
	output, err := executeCommandC(
		rootCmd, "plugin", "install",
		"-p", pluginTestConfigFile, "--update", "--backup", "--overwrite-config=false")
	require.NoError(t, err, "plugin install should not return an error")
	assert.Contains(t, output, "/gatewayd-plugin-cache-linux-amd64-")
	assert.Contains(t, output, "/checksums.txt")
	assert.Contains(t, output, "Download completed successfully")
	assert.Contains(t, output, "Checksum verification passed")
	assert.Contains(t, output, "Plugin binary extracted to plugins/gatewayd-plugin-cache")
	assert.Contains(t, output, "Plugin installed successfully")

	// See if the plugin was actually installed.
	output, err = executeCommandC(rootCmd, "plugin", "list", "-p", pluginTestConfigFile)
	require.NoError(t, err, "plugin list should not return an error")
	assert.Contains(t, output, "Name: gatewayd-plugin-cache")

	// Clean up.
	assert.FileExists(t, "plugins/gatewayd-plugin-cache")
	assert.FileExists(t, fmt.Sprintf("%s.bak", pluginTestConfigFile))
	assert.NoFileExists(t, "plugins/LICENSE")
	assert.NoFileExists(t, "plugins/README.md")
	assert.NoFileExists(t, "plugins/checksum.txt")
	assert.NoFileExists(t, "plugins/gatewayd_plugin.yaml")

	require.NoError(t, os.RemoveAll("plugins/"))
	require.NoError(t, os.Remove(fmt.Sprintf("%s.bak", pluginTestConfigFile)))
}
