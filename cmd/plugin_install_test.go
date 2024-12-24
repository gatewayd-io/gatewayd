package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_pluginInstallCmdWithFile(t *testing.T) {
	pluginTestConfigFile := "./test_plugins_pluginInstallCmdWithFile.yaml"

	// Create a test plugin config file.
	output, err := executeCommandC(rootCmd, "plugin", "init", "-p", pluginTestConfigFile)
	require.NoError(t, err, "plugin init should not return an error")
	assert.Equal(t,
		fmt.Sprintf("Config file '%s' was created successfully.", pluginTestConfigFile),
		output,
		"plugin init command should have returned the correct output")
	assert.FileExists(
		t, pluginTestConfigFile, "plugin init command should have created a config file")

	// Pull the plugin archive and install it.
	pluginArchivePath, err = mustPullPlugin()
	require.NoError(t, err, "mustPullPlugin should not return an error")
	assert.FileExists(t, pluginArchivePath, "mustPullPlugin should have downloaded the plugin archive")

	// Test plugin install command.
	output, err = executeCommandC(
		rootCmd, "plugin", "install", "-p", pluginTestConfigFile,
		"--update", "--backup", "--name=gatewayd-plugin-cache", pluginArchivePath)
	require.NoError(t, err, "plugin install should not return an error")
	assert.Contains(t, output, "Installing plugin from CLI argument")
	assert.Contains(t, output, "Backup completed successfully")
	assert.Contains(t, output, "Plugin binary extracted to plugins/gatewayd-plugin-cache")
	assert.Contains(t, output, "Plugin installed successfully")

	// See if the plugin was actually installed.
	output, err = executeCommandC(rootCmd, "plugin", "list", "-p", pluginTestConfigFile)
	require.NoError(t, err, "plugin list should not return an error")
	assert.Contains(t, output, "Name: gatewayd-plugin-cache")

	// Clean up.
	assert.FileExists(t, "plugins/gatewayd-plugin-cache")
	assert.FileExists(t, pluginTestConfigFile+BackupFileExt)
	assert.NoFileExists(t, "gatewayd-plugin-cache-linux-amd64-v0.2.4.tar.gz")
	assert.NoFileExists(t, "checksums.txt")
	assert.NoFileExists(t, "plugins/LICENSE")
	assert.NoFileExists(t, "plugins/README.md")
	assert.NoFileExists(t, "plugins/checksum.txt")
	assert.NoFileExists(t, "plugins/gatewayd_plugin.yaml")

	require.NoError(t, os.RemoveAll("plugins/"))
	require.NoError(t, os.Remove(pluginArchivePath))
	require.NoError(t, os.Remove(pluginTestConfigFile))
	require.NoError(t, os.Remove(pluginTestConfigFile+BackupFileExt))
}

func Test_pluginInstallCmdAutomatedNoOverwrite(t *testing.T) {
	pwd, err := os.Getwd()
	require.NoError(t, err, "os.Getwd should not return an error")

	pluginTestConfigFile := "./testdata/gatewayd_plugins.yaml"

	// Test plugin install command with overwrite disabled
	output, err := executeCommandC(
		rootCmd, "plugin", "install", "-p", pluginTestConfigFile,
		"--update", "--backup", "--overwrite-config=false")
	require.NoError(t, err, "plugin install should not return an error")

	// Verify expected output for no-overwrite case
	assert.Contains(t, output, "Installing plugins from plugins configuration file")
	assert.Contains(t, output, fmt.Sprintf("gatewayd-plugin-cache-%s-%s-", runtime.GOOS, runtime.GOARCH))
	assert.Contains(t, output, "checksums.txt")
	assert.Contains(t, output, "Download completed successfully")
	assert.Contains(t, output, "Checksum verification passed")
	assert.Contains(t, output, "Plugin binary extracted to plugins/gatewayd-plugin-cache")
	assert.Contains(t, output, "Plugin installed successfully")

	// Cleanup
	require.NoError(t, os.RemoveAll("plugins/"))
	checksumFile := filepath.Join(pwd, "checksums.txt")
	if _, err := os.Stat(checksumFile); err == nil {
		require.NoError(t, os.Remove(checksumFile))
	}
	require.NoError(t, os.Remove(pluginTestConfigFile+BackupFileExt))
}

func Test_pluginInstallCmdPullOnly(t *testing.T) {
	pwd, err := os.Getwd()
	require.NoError(t, err, "os.Getwd should not return an error")

	pluginTestConfigFile := "./testdata/gatewayd_plugins.yaml"

	// Test plugin install command in pull-only mode
	output, err := executeCommandC(
		rootCmd, "plugin", "install", "-p", pluginTestConfigFile,
		"--update", "--backup", "--pull-only", "--overwrite-config=false")
	require.NoError(t, err, "plugin install should not return an error")

	// Verify pull-only behavior
	assert.Contains(t, output, "Installing plugins from plugins configuration file")
	assert.Contains(t, output, fmt.Sprintf("gatewayd-plugin-cache-%s-%s-", runtime.GOOS, runtime.GOARCH))
	assert.Contains(t, output, "Download completed successfully")

	// Should not contain installation messages in pull-only mode
	assert.NotContains(t, output, "Plugin binary extracted")
	assert.NotContains(t, output, "Plugin installed successfully")

	// Cleanup downloaded files
	pattern := filepath.Join(pwd, fmt.Sprintf("gatewayd-plugin-cache-%s-%s-*.tar.gz", runtime.GOOS, runtime.GOARCH))
	matches, err := filepath.Glob(pattern)
	require.NoError(t, err, "failed to glob plugin archives")
	for _, match := range matches {
		require.NoError(t, os.Remove(match))
	}

	checksumFile := filepath.Join(pwd, "checksums.txt")
	if _, err := os.Stat(checksumFile); err == nil {
		require.NoError(t, os.Remove(checksumFile))
	}
}
