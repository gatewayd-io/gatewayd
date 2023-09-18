package cmd

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_pluginCmd(t *testing.T) {
	// Test pluginCmd with no arguments.
	output, err := executeCommandC(rootCmd, "plugin")
	assert.NoError(t, err, "pluginCmd should not return an error")
	assert.Equal(t, `Manage plugins and their configuration

Usage:
  gatewayd plugin [flags]
  gatewayd plugin [command]

Available Commands:
  init        Create or overwrite the GatewayD plugins config
  install     Install a plugin from a local archive or a GitHub repository
  lint        Lint the GatewayD plugins config
  list        List the GatewayD plugins

Flags:
  -h, --help   help for plugin

Use "gatewayd plugin [command] --help" for more information about a command.
`,
		output,
		"pluginCmd should print the correct output")
}
