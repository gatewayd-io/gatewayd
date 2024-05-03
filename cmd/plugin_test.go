package cmd

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_pluginCmd(t *testing.T) {
	// Test pluginCmd with no arguments.
	output, err := executeCommandC(rootCmd, "plugin")
	require.NoError(t, err, "pluginCmd should not return an error")
	assert.Equal(t, `Manage plugins and their configuration

Usage:
  gatewayd plugin [flags]
  gatewayd plugin [command]

Available Commands:
  help        Help about any command
  init        Create or overwrite the GatewayD plugins config
  install     Install a plugin from a local archive or a GitHub repository
  lint        Lint the GatewayD plugins config
  list        List the GatewayD plugins
  scaffold    Scaffold a plugin and store the files into a directory

Flags:
  -h, --help   help for plugin

Use "gatewayd plugin [command] --help" for more information about a command.
`,
		output,
		"pluginCmd should print the correct output")
}
