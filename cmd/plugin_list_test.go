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
	assert.Equal(t, `Total plugins: 1
Plugins:
  Name: gatewayd-plugin-cache
  Enabled: true
  Path: ../gatewayd-plugin-cache/gatewayd-plugin-cache
  Args: --log-level debug
  Env:
    MAGIC_COOKIE_KEY=GATEWAYD_PLUGIN
    MAGIC_COOKIE_VALUE=5712b87aa5d7e9f9e9ab643e6603181c5b796015cb1c09d6f5ada882bf2a1872
    REDIS_URL=redis://localhost:6379/0
    EXPIRY=1h
    METRICS_ENABLED=True
    METRICS_UNIX_DOMAIN_SOCKET=/tmp/gatewayd-plugin-cache.sock
    METRICS_PATH=/metrics
    PERIODIC_INVALIDATOR_ENABLED=True
    PERIODIC_INVALIDATOR_INTERVAL=1m
    PERIODIC_INVALIDATOR_START_DELAY=1m
    API_ADDRESS=localhost:18080
    EXIT_ON_STARTUP_ERROR=False
    SENTRY_DSN=https://70eb1abcd32e41acbdfc17bc3407a543@o4504550475038720.ingest.sentry.io/4505342961123328
  Checksum: 054e7dba9c1e3e3910f4928a000d35c8a6199719fad505c66527f3e9b1993833
`,
		output,
		"plugin list command should have returned the correct output")
}
