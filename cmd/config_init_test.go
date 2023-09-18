package cmd

import (
	"fmt"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_configInitCmd(t *testing.T) {
	// Test configInitCmd.
	output, err := executeCommandC(rootCmd, "config", "init", "-c", globalTestConfigFile)
	assert.NoError(t, err, "configInitCmd should not return an error")
	assert.Equal(t,
		fmt.Sprintf("Config file '%s' was created successfully.", globalTestConfigFile),
		output,
		"configInitCmd should print the correct output")
	// Check that the config file was created.
	assert.FileExists(t, globalTestConfigFile, "configInitCmd should create a config file")

	// Test configInitCmd with the --force flag to overwrite the config file.
	output, err = executeCommandC(rootCmd, "config", "init", "--force")
	assert.NoError(t, err, "configInitCmd should not return an error")
	assert.Equal(t,
		fmt.Sprintf("Config file '%s' was overwritten successfully.", globalTestConfigFile),
		output,
		"configInitCmd should print the correct output")

	// Clean up.
	err = os.Remove(globalTestConfigFile)
	assert.NoError(t, err)
}
