package cmd

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/zenizh/go-capturer"
)

//nolint:lll
const rootHelp string = `GatewayD is a cloud-native database gateway and framework for building data-driven applications. It sits between your database servers and clients and proxies all their communication.

Usage:
  gatewayd [command]

Available Commands:
  completion  Generate the autocompletion script for the specified shell
  config      Manage GatewayD global configuration
  help        Help about any command
  plugin      Manage plugins and their configuration
  run         Run a GatewayD instance
  version     Show version information

Flags:
  -h, --help   help for gatewayd

Use "gatewayd [command] --help" for more information about a command.
`

func Test_rootCmd(t *testing.T) {
	output, err := executeCommandC(rootCmd)
	require.NoError(t, err, "rootCmd should not return an error")
	assert.Equal(t,
		rootHelp,
		output,
		"rootCmd should print the correct output")
}

func Test_Execute(t *testing.T) {
	stdout := capturer.CaptureStdout(func() {
		Execute()
	})
	assert.Equal(t, rootHelp, stdout, "Execute should print the correct output")
}
