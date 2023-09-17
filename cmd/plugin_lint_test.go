package cmd

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_pluginLintCmd(t *testing.T) {
	// Test plugin lint command.
	pluginConfigFile := "../gatewayd_plugins.yaml"
	output, err := executeCommandC(rootCmd, "plugin", "lint", "-p", pluginConfigFile)
	assert.NoError(t, err, "plugin lint command should not have returned an error")
	assert.Equal(t,
		"plugins config is valid\n",
		output,
		"plugin lint command should have returned the correct output")
}
