package cmd

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/codingsince1985/checksum"
	"github.com/gatewayd-io/gatewayd/config"
	"github.com/gatewayd-io/gatewayd/plugin"
	"github.com/gatewayd-io/gatewayd/testhelpers"
	"github.com/spf13/cast"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	yamlv3 "gopkg.in/yaml.v3"
)

func Test_pluginScaffoldCmd(t *testing.T) {
	// Start the test containers.
	ctx := context.Background()
	postgresHostIP1, postgresMappedPort1 := testhelpers.SetupPostgreSQLTestContainer(ctx, t)
	postgresHostIP2, postgresMappedPort2 := testhelpers.SetupPostgreSQLTestContainer(ctx, t)
	postgresAddress1 := postgresHostIP1 + ":" + postgresMappedPort1.Port()
	t.Setenv("GATEWAYD_CLIENTS_DEFAULT_WRITES_ADDRESS", postgresAddress1)
	postgresAddress2 := postgresHostIP2 + ":" + postgresMappedPort2.Port()
	t.Setenv("GATEWAYD_CLIENTS_TEST_WRITE_ADDRESS", postgresAddress2)

	globalTestConfigFile := filepath.Join("testdata", "gatewayd.yaml")
	plugin.IsPluginTemplateEmbedded()
	pluginTestScaffoldInputFile := "./testdata/scaffold_input.yaml"

	output, err := executeCommandC(
		rootCmd, "plugin", "scaffold", "-i", pluginTestScaffoldInputFile)
	require.NoError(t, err, "plugin scaffold should not return an error")
	assert.Contains(t, output, "scaffold done")
	assert.Contains(t, output, "created files:")
	assert.Contains(t, output, "test-gatewayd-plugin/.github/issue_template.md")
	assert.Contains(t, output, "test-gatewayd-plugin/.github/pull_request_template.md")
	assert.Contains(t, output, "test-gatewayd-plugin/.github/workflows/commits-signed.yaml")

	pluginsConfig, err := os.ReadFile(
		filepath.Join("plugins", "test-gatewayd-plugin", "gatewayd_plugin.yaml"))
	require.NoError(t, err, "reading plugins config file should not return an error")

	var localPluginsConfig map[string]interface{}
	err = yamlv3.Unmarshal(pluginsConfig, &localPluginsConfig)
	require.NoError(t, err, "unmarshalling yaml file should not return error")

	tidy := exec.Command("go", "mod", "tidy")
	tidy.Dir = filepath.Join("plugins", "test-gatewayd-plugin")
	err = tidy.Run()
	assert.NoError(t, err)

	build := exec.Command("make", "build-dev")
	build.Dir = filepath.Join("plugins", "test-gatewayd-plugin")
	err = build.Run()
	assert.NoError(t, err)

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

	err = os.WriteFile(
		filepath.Join("plugins", "test-gatewayd-plugin", "gatewayd_plugins.yaml"),
		updatedPluginConfig, FilePermissions)
	require.NoError(t, err, "writingh to yaml file should not return error")

	pluginTestConfigFile := filepath.Join(
		"plugins", "test-gatewayd-plugin", "gatewayd_plugins.yaml")

	stopChan = make(chan struct{})

	var waitGroup sync.WaitGroup

	waitGroup.Add(1)
	go func(waitGroup *sync.WaitGroup) {
		// Test run command.
		output, err := executeCommandC(
			rootCmd, "run", "-c", globalTestConfigFile, "-p", pluginTestConfigFile)
		require.NoError(t, err, "run command should not have returned an error")

		// Print the output for debugging purposes.
		runCmd.Print(output)
		// Check if GatewayD started and stopped correctly.
		assert.Contains(t, output, "GatewayD is running")
		assert.Contains(t, output, "Stopped all servers")

		waitGroup.Done()
	}(&waitGroup)

	waitGroup.Add(1)
	go func(waitGroup *sync.WaitGroup) {
		time.Sleep(waitBeforeStop)

		StopGracefully(
			context.Background(),
			nil,
			nil,
			metricsServer,
			nil,
			loggers[config.Default],
			servers,
			stopChan,
			nil,
			nil,
		)

		waitGroup.Done()
	}(&waitGroup)

	waitGroup.Wait()

	assert.NoError(t, os.RemoveAll("plugins"))
}
