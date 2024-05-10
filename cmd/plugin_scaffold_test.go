package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/codingsince1985/checksum"
	"github.com/gatewayd-io/gatewayd/plugin"
	"github.com/spf13/cast"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	yamlv3 "gopkg.in/yaml.v3"
)

func Test_pluginScaffoldCmd(t *testing.T) {
	globalTestConfigFile := filepath.Join("testdata", "gatewayd.yaml")
	plugin.IsPluginTemplateEmbedded()
	pluginTestScaffoldInputFile := "./testdata/scaffold_input.yaml"

	output, err := executeCommandC(
		rootCmd, "plugin", "scaffold",
		"-i", pluginTestScaffoldInputFile)
	require.NoError(t, err, "plugin scaffold should not return an error")
	assert.Contains(t, output, "scaffold done")
	assert.Contains(t, output, "created files:")
	assert.Contains(t, output, "test-gatewayd-plugin/.github/issue_template.md")
	assert.Contains(t, output, "test-gatewayd-plugin/.github/pull_request_template.md")
	assert.Contains(t, output, "test-gatewayd-plugin/.github/workflows/commits-signed.yaml")

	pluginsConfig, err := os.ReadFile(filepath.Join("plugins", "test-gatewayd-plugin", "gatewayd_plugin.yaml"))
	require.NoError(t, err, "reading plugins config file should not return an error")

	var localPluginsConfig map[string]interface{}
	err = yamlv3.Unmarshal(pluginsConfig, &localPluginsConfig)
	require.NoError(t, err, "unmarshalling yaml file should not return error")

	// TODO: build the plugin binary and check that it is valid.

	_, err = os.ReadFile(filepath.Join("plugins", "test-gatewayd-plugin", "test-gatewayd-plugin"))
	require.NoError(t, err, "reading plugin binary file should not return an error")

	pluginsList := cast.ToSlice(localPluginsConfig["plugins"])
	plugin := cast.ToStringMap(pluginsList[0])
	pluginsList[0] = plugin
	plugin["localPath"] = filepath.Join("cmd", "plugins", "test-gatewayd-plugin", "test-gatewayd-plugin")
	sum, err := checksum.SHA256sum(filepath.Join("plugins", "test-gatewayd-plugin", "test-gatewayd-plugin"))
	require.NoError(t, err, "marshalling yaml file should not return error")
	plugin["checksum"] = sum

	pluginsList[0] = plugin
	plugins := make(map[string]interface{})
	plugins["plugins"] = pluginsList

	updatedPluginConfig, err := yamlv3.Marshal(plugins)
	require.NoError(t, err, "marshalling yaml file should not return error")

	err = os.WriteFile(filepath.Join("plugins", "test-gatewayd-plugin", "gatewayd_plugins.yaml"), updatedPluginConfig, FilePermissions)
	require.NoError(t, err, "writingh to yaml file should not return error")

	output, err = executeCommandC(rootCmd, "run", "-p", filepath.Join("plugins", "test-gatewayd-plugin", "gatewayd_plugins.yaml"), "-c", globalTestConfigFile)
	require.NoError(t, err, "run command should not have returned an error")
	fmt.Println(output)

	// var waitGroup sync.WaitGroup
	// waitGroup.Add(1)
	// go func(waitGroup *sync.WaitGroup) {
	// 	fmt.Println("sasasaas")
	// 	// Test run command.
	// 	_, err := executeCommandC(rootCmd, "run", "-p", filepath.Join("plugins", "test-gatewayd-plugin", "gatewayd_plugins.yaml"), "-c", globalTestConfigFile)
	// 	require.NoError(t, err, "run command should not have returned an error")
	// 	fmt.Println("2121")

	// 	// // Print the output for debugging purposes.
	// 	// runCmd.Print(output)
	// 	// Check if GatewayD started and stopped correctly.
	// 	// assert.Contains(t, output, "GatewayD is running")
	// 	// assert.Contains(t, output, "Stopped all servers")

	// 	waitGroup.Done()
	// }(&waitGroup)

	// waitGroup.Wait()

	// waitGroup.Add(1)
	// go func(waitGroup *sync.WaitGroup) {
	// 	time.Sleep(waitBeforeStop * 2)

	// 	StopGracefully(
	// 		context.Background(),
	// 		nil,
	// 		nil,
	// 		metricsServer,
	// 		nil,
	// 		loggers[config.Default],
	// 		servers,
	// 		stopChan,
	// 		nil,
	// 		nil,
	// 	)

	// 	waitGroup.Done()
	// }(&waitGroup)

	// waitGroup.Wait()
}
