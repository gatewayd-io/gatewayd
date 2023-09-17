package cmd

import (
	"fmt"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_pluginInitCmd(t *testing.T) {
	// Test plugin init command.
	output, err := executeCommandC(rootCmd, "plugin", "init", "-p", pluginTestConfigFile)
	assert.NoError(t, err, "plugin init command should not have returned an error")
	assert.Equal(t,
		fmt.Sprintf("Config file '%s' was created successfully.", pluginTestConfigFile),
		output,
		"plugin init command should have returned the correct output")
	assert.FileExists(t, pluginTestConfigFile, "plugin init command should have created a config file")

	// Clean up.
	err = os.Remove(pluginTestConfigFile)
	assert.NoError(t, err)
}
