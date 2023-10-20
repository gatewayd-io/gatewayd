package cmd

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_pluginLintCmd(t *testing.T) {
	// Test plugin lint command.
	output, err := executeCommandC(rootCmd, "plugin", "lint", "-p", "../gatewayd_plugins.yaml")
	require.NoError(t, err, "plugin lint command should not have returned an error")
	assert.Equal(t,
		"plugins config is valid\n",
		output,
		"plugin lint command should have returned the correct output")
}
