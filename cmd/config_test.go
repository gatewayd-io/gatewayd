package cmd

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_configCmd(t *testing.T) {
	// Test configCmd with no arguments.
	output, err := executeCommandC(rootCmd, "config")
	require.NoError(t, err, "configCmd should not return an error")
	assert.Equal(t,
		`Manage GatewayD global configuration

Usage:
  gatewayd config [flags]
  gatewayd config [command]

Available Commands:
  init        Create or overwrite the GatewayD global config
  lint        Lint the GatewayD global config

Flags:
  -h, --help   help for config

Use "gatewayd config [command] --help" for more information about a command.
`,
		output,
		"configCmd should print the correct output")
}
