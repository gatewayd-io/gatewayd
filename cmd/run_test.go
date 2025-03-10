package cmd

import (
	"os"
	"sync"
	"testing"
	"time"

	"github.com/gatewayd-io/gatewayd/testhelpers"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var (
	waitBeforeStop    = time.Second
	pluginArchivePath = ""
)

func Test_runCmd(t *testing.T) {
	previous := EnableTestMode()
	defer func() {
		testMode = previous
		testApp = nil
	}()

	postgresHostIP, postgresMappedPort := testhelpers.SetupPostgreSQLTestContainer(t.Context(), t)
	postgresAddress := postgresHostIP + ":" + postgresMappedPort.Port()
	t.Setenv("GATEWAYD_CLIENTS_DEFAULT_WRITES_ADDRESS", postgresAddress)

	tempDir := t.TempDir()
	t.Setenv("GATEWAYD_RAFT_DIRECTORY", tempDir)
	t.Setenv("GATEWAYD_RAFT_ADDRESS", "127.0.0.1:0")
	t.Setenv("GATEWAYD_RAFT_GRPCADDRESS", "127.0.0.1:0")

	globalTestConfigFile := "./test_global_runCmd.yaml"
	pluginTestConfigFile := "./test_plugins_runCmd.yaml"
	// Create a test plugins config file.
	_, err := executeCommandC(rootCmd, "plugin", "init", "--force", "-p", pluginTestConfigFile)
	require.NoError(t, err, "plugin init command should not have returned an error")
	assert.FileExists(t, pluginTestConfigFile, "plugin init command should have created a config file")

	// Create a test config file.
	_, err = executeCommandC(rootCmd, "config", "init", "--force", "-c", globalTestConfigFile)
	require.NoError(t, err, "configInitCmd should not return an error")
	// Check that the config file was created.
	assert.FileExists(t, globalTestConfigFile, "configInitCmd should create a config file")

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

	// Stop the server after a delay
	waitGroup.Add(1)
	go func(waitGroup *sync.WaitGroup) {
		time.Sleep(waitBeforeStop)
		testApp.stopGracefully(t.Context(), os.Interrupt)
		waitGroup.Done()
	}(&waitGroup)

	waitGroup.Wait()

	require.NoError(t, os.Remove(globalTestConfigFile))
	require.NoError(t, os.Remove(pluginTestConfigFile))
}

// Test_runCmdWithTLS tests the run command with TLS enabled on the server.
func Test_runCmdWithTLS(t *testing.T) {
	previous := EnableTestMode()
	defer func() {
		testMode = previous
		testApp = nil
	}()

	postgresHostIP, postgresMappedPort := testhelpers.SetupPostgreSQLTestContainer(t.Context(), t)
	postgresAddress := postgresHostIP + ":" + postgresMappedPort.Port()
	t.Setenv("GATEWAYD_CLIENTS_DEFAULT_WRITES_ADDRESS", postgresAddress)

	tempDir := t.TempDir()
	t.Setenv("GATEWAYD_RAFT_DIRECTORY", tempDir)
	t.Setenv("GATEWAYD_RAFT_ADDRESS", "127.0.0.1:0")
	t.Setenv("GATEWAYD_RAFT_GRPCADDRESS", "127.0.0.1:0")

	globalTLSTestConfigFile := "./testdata/gatewayd_tls.yaml"
	pluginTestConfigFile := "./test_plugins_runCmdWithTLS.yaml"
	// Create a test plugins config file.
	_, err := executeCommandC(rootCmd, "plugin", "init", "--force", "-p", pluginTestConfigFile)
	require.NoError(t, err, "plugin init command should not have returned an error")
	assert.FileExists(t, pluginTestConfigFile, "plugin init command should have created a config file")

	var waitGroup sync.WaitGroup
	// TODO: Test client certificate authentication.

	waitGroup.Add(1)
	go func(waitGroup *sync.WaitGroup) {
		// Test run command.
		output, err := executeCommandC(rootCmd, "run", "-c", globalTLSTestConfigFile, "-p", pluginTestConfigFile)
		require.NoError(t, err, "run command should not have returned an error")

		// Print the output for debugging purposes.
		runCmd.Print(output)

		// Check if GatewayD started and stopped correctly.
		assert.Contains(t, output, "GatewayD is running")
		assert.Contains(t, output, "TLS is enabled")
		assert.Contains(t, output, "Stopped all servers")

		waitGroup.Done()
	}(&waitGroup)

	// Stop the server after a delay
	waitGroup.Add(1)
	go func(waitGroup *sync.WaitGroup) {
		time.Sleep(waitBeforeStop)
		testApp.stopGracefully(t.Context(), os.Interrupt)
		waitGroup.Done()
	}(&waitGroup)

	waitGroup.Wait()

	require.NoError(t, os.Remove(pluginTestConfigFile))
}

// Test_runCmdWithMultiTenancy tests the run command with multi-tenancy enabled.
func Test_runCmdWithMultiTenancy(t *testing.T) {
	previous := EnableTestMode()
	defer func() {
		testMode = previous
		testApp = nil
	}()

	postgresHostIP, postgresMappedPort := testhelpers.SetupPostgreSQLTestContainer(t.Context(), t)
	postgresAddress := postgresHostIP + ":" + postgresMappedPort.Port()
	t.Setenv("GATEWAYD_CLIENTS_DEFAULT_WRITES_ADDRESS", postgresAddress)
	postgresHostIP2, postgresMappedPort2 := testhelpers.SetupPostgreSQLTestContainer(t.Context(), t)
	postgresAddress2 := postgresHostIP2 + ":" + postgresMappedPort2.Port()
	t.Setenv("GATEWAYD_CLIENTS_TEST_WRITE_ADDRESS", postgresAddress2)

	tempDir := t.TempDir()
	t.Setenv("GATEWAYD_RAFT_DIRECTORY", tempDir)
	t.Setenv("GATEWAYD_RAFT_ADDRESS", "127.0.0.1:0")
	t.Setenv("GATEWAYD_RAFT_GRPCADDRESS", "127.0.0.1:0")

	globalTestConfigFile := "./testdata/gatewayd.yaml"
	pluginTestConfigFile := "./test_plugins_runCmdWithMultiTenancy.yaml"
	// Create a test plugins config file.
	_, err := executeCommandC(rootCmd, "plugin", "init", "--force", "-p", pluginTestConfigFile)
	require.NoError(t, err, "plugin init command should not have returned an error")
	assert.FileExists(t, pluginTestConfigFile, "plugin init command should have created a config file")

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
		assert.Contains(t, output, "There are clients available in the pool count=10 group=default")
		assert.Contains(t, output, "There are clients available in the pool count=10 group=test")
		assert.Contains(t, output, "GatewayD is listening address=0.0.0.0:15432")
		assert.Contains(t, output, "GatewayD is listening address=0.0.0.0:15433")
		assert.Contains(t, output, "Stopped all servers")

		waitGroup.Done()
	}(&waitGroup)

	// Stop the server after a delay
	waitGroup.Add(1)
	go func(waitGroup *sync.WaitGroup) {
		time.Sleep(waitBeforeStop)
		testApp.stopGracefully(t.Context(), os.Interrupt)
		waitGroup.Done()
	}(&waitGroup)

	waitGroup.Wait()

	require.NoError(t, os.Remove(pluginTestConfigFile))
}

func Test_runCmdWithCachePlugin(t *testing.T) {
	previous := EnableTestMode()
	defer func() {
		testMode = previous
		testApp = nil
	}()

	postgresHostIP, postgresMappedPort := testhelpers.SetupPostgreSQLTestContainer(t.Context(), t)
	postgresAddress := postgresHostIP + ":" + postgresMappedPort.Port()
	t.Setenv("GATEWAYD_CLIENTS_DEFAULT_WRITES_ADDRESS", postgresAddress)

	tempDir := t.TempDir()
	t.Setenv("GATEWAYD_RAFT_DIRECTORY", tempDir)
	t.Setenv("GATEWAYD_RAFT_ADDRESS", "127.0.0.1:0")
	t.Setenv("GATEWAYD_RAFT_GRPCADDRESS", "127.0.0.1:0")

	globalTestConfigFile := "./test_global_runCmdWithCachePlugin.yaml"
	pluginTestConfigFile := "./test_plugins_runCmdWithCachePlugin.yaml"

	// Create a test plugins config file.
	_, err := executeCommandC(rootCmd, "plugin", "init", "--force", "-p", pluginTestConfigFile)
	require.NoError(t, err, "plugin init command should not have returned an error")
	assert.FileExists(t, pluginTestConfigFile, "plugin init command should have created a config file")

	// Create a test config file.
	_, err = executeCommandC(rootCmd, "config", "init", "--force", "-c", globalTestConfigFile)
	require.NoError(t, err, "configInitCmd should not return an error")
	// Check that the config file was created.
	assert.FileExists(t, globalTestConfigFile, "configInitCmd should create a config file")

	// Pull the plugin archive and install it.
	pluginArchivePath, err = mustPullPlugin()
	require.NoError(t, err, "mustPullPlugin should not return an error")
	assert.FileExists(t, pluginArchivePath, "mustPullPlugin should have downloaded the plugin archive")

	// Test plugin install command.
	output, err := executeCommandC(
		rootCmd, "plugin", "install", "-p", pluginTestConfigFile, "--update", "--backup",
		"--overwrite-config=true", "--name", "gatewayd-plugin-cache", pluginArchivePath)
	require.NoError(t, err, "plugin install should not return an error")
	assert.Contains(t, output, "Installing plugin from CLI argument")
	assert.Contains(t, output, "Backup completed successfully")
	assert.Contains(t, output, "Plugin binary extracted to plugins/gatewayd-plugin-cache")
	assert.Contains(t, output, "Plugin installed successfully")

	// See if the plugin was actually installed.
	output, err = executeCommandC(rootCmd, "plugin", "list", "-p", pluginTestConfigFile)
	require.NoError(t, err, "plugin list should not return an error")
	assert.Contains(t, output, "Name: gatewayd-plugin-cache")

	var waitGroup sync.WaitGroup

	waitGroup.Add(1)
	go func(waitGroup *sync.WaitGroup) {
		// Test run command.
		output, err := executeCommandC(rootCmd, "run", "-c", globalTestConfigFile, "-p", pluginTestConfigFile)
		require.NoError(t, err, "run command should not have returned an error")

		// Print the output for debugging purposes.
		runCmd.Print(output)
		// Check if GatewayD started and stopped correctly.
		assert.Contains(t, output, "GatewayD is running")
		assert.Contains(t, output, "Stopped all servers")

		waitGroup.Done()
	}(&waitGroup)

	// Stop the server after a delay
	waitGroup.Add(1)
	go func(waitGroup *sync.WaitGroup) {
		time.Sleep(waitBeforeStop)
		testApp.stopGracefully(t.Context(), os.Interrupt)
		waitGroup.Done()
	}(&waitGroup)

	waitGroup.Wait()

	require.NoError(t, os.RemoveAll("plugins/"))
	require.NoError(t, os.Remove(globalTestConfigFile))
	require.NoError(t, os.Remove(pluginTestConfigFile))
	require.NoError(t, os.Remove(pluginTestConfigFile+BackupFileExt))
}
