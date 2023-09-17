package cmd

import (
	"fmt"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_pluginInitCmd(t *testing.T) {
	// Test plugin init command.
	pluginConfigFile := "./test.yaml"
	output, err := executeCommandC(rootCmd, "plugin", "init", "-p", pluginConfigFile)
	assert.NoError(t, err, "plugin init command should not have returned an error")
	assert.Equal(t,
		fmt.Sprintf("Config file '%s' was created successfully.", pluginConfigFile),
		output,
		"plugin init command should have returned the correct output")
	assert.FileExists(t, pluginConfigFile, "plugin init command should have created a config file")

	// Clean up.
	err = os.Remove(pluginConfigFile)
	assert.NoError(t, err)
}
