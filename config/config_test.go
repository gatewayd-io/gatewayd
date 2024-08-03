package config

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/knadh/koanf"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var parentDir = "../"

// TestNewConfig tests the NewConfig function.
func TestNewConfig(t *testing.T) {
	config := NewConfig(
		context.Background(), Config{GlobalConfigFile: GlobalConfigFilename, PluginConfigFile: PluginsConfigFilename})
	assert.NotNil(t, config)
	assert.Equal(t, GlobalConfigFilename, config.GlobalConfigFile)
	assert.Equal(t, PluginsConfigFilename, config.PluginConfigFile)
	assert.Equal(t, GlobalConfig{}, config.globalDefaults)
	assert.Equal(t, PluginConfig{}, config.pluginDefaults)
	assert.Equal(t, GlobalConfig{}, config.Global)
	assert.Equal(t, PluginConfig{}, config.Plugin)
	assert.Equal(t, koanf.New("."), config.GlobalKoanf)
	assert.Equal(t, koanf.New("."), config.PluginKoanf)
}

// TestInitConfig tests the InitConfig function, which practically tests all
// the other functions.
func TestInitConfig(t *testing.T) {
	ctx := context.Background()
	config := NewConfig(ctx,
		Config{
			GlobalConfigFile: parentDir + "cmd/testdata/gatewayd.yaml",
			PluginConfigFile: parentDir + PluginsConfigFilename,
		},
	)
	err := config.InitConfig(ctx)
	require.Nil(t, err)
	assert.NotNil(t, config.Global)
	assert.NotEqual(t, GlobalConfig{}, config.Global)
	assert.Contains(t, config.Global.Servers, Default)
	assert.Contains(t, config.Global.Servers, "test") // Test the multi-tenant configuration.
	assert.NotNil(t, config.Plugin)
	assert.NotEqual(t, PluginConfig{}, config.Plugin)
	assert.Len(t, config.Plugin.Plugins, 1)
	assert.NotNil(t, config.GlobalKoanf)
	assert.NotEqual(t, config.GlobalKoanf, koanf.New("."))
	assert.Equal(t, DefaultLogLevel, config.GlobalKoanf.String("loggers.default.level"))
	assert.NotNil(t, config.PluginKoanf)
	assert.NotEqual(t, config.PluginKoanf, koanf.New("."))
	assert.NotNil(t, config.globalDefaults)
	assert.NotEqual(t, GlobalConfig{}, config.globalDefaults)
	assert.Contains(t, config.globalDefaults.Servers, Default)
	assert.Contains(t, config.globalDefaults.Servers, "test")
	assert.NotNil(t, config.pluginDefaults)
	assert.NotEqual(t, PluginConfig{}, config.pluginDefaults)
	assert.Empty(t, config.pluginDefaults.Plugins)
}

// TestInitConfigMultiTenant tests the InitConfig function with a multi-tenant configuration.
func TestInitConfigMultiTenant(t *testing.T) {
	ctx := context.Background()
	config := NewConfig(ctx,
		Config{GlobalConfigFile: parentDir + GlobalConfigFilename, PluginConfigFile: parentDir + PluginsConfigFilename})
	err := config.InitConfig(ctx)
	require.Nil(t, err)
	assert.NotNil(t, config.Global)
	assert.NotEqual(t, GlobalConfig{}, config.Global)
	assert.Contains(t, config.Global.Servers, Default)
	assert.NotNil(t, config.Plugin)
	assert.NotEqual(t, PluginConfig{}, config.Plugin)
	assert.Len(t, config.Plugin.Plugins, 1)
	assert.NotNil(t, config.GlobalKoanf)
	assert.NotEqual(t, config.GlobalKoanf, koanf.New("."))
	assert.Equal(t, DefaultLogLevel, config.GlobalKoanf.String("loggers.default.level"))
	assert.NotNil(t, config.PluginKoanf)
	assert.NotEqual(t, config.PluginKoanf, koanf.New("."))
	assert.NotNil(t, config.globalDefaults)
	assert.NotEqual(t, GlobalConfig{}, config.globalDefaults)
	assert.Contains(t, config.globalDefaults.Servers, Default)
	assert.NotNil(t, config.pluginDefaults)
	assert.NotEqual(t, PluginConfig{}, config.pluginDefaults)
	assert.Empty(t, config.pluginDefaults.Plugins)
}

// TestInitConfigMissingFile tests the InitConfig function with a missing file.
func TestInitConfigMissingKeys(t *testing.T) {
	ctx := context.Background()
	config := NewConfig(ctx,
		Config{
			GlobalConfigFile: "./testdata/missing_keys.yaml",
			PluginConfigFile: parentDir + PluginsConfigFilename,
		},
	)
	err := config.InitConfig(ctx)
	assert.Error(t, err)
	assert.Contains(t, err.Error(),
		"validation failed, OriginalError: failed to validate global configuration")
}

// TestInitConfigMissingFile tests the InitConfig function with a missing file.
func TestInitConfigMissingFile(t *testing.T) {
	ctx := context.Background()
	config := NewConfig(ctx,
		Config{
			GlobalConfigFile: "./testdata/missing_file.yaml",
			PluginConfigFile: parentDir + PluginsConfigFilename,
		},
	)
	err := config.InitConfig(ctx)
	assert.Error(t, err)
	assert.Contains(
		t,
		err.Error(),
		"error parsing config, OriginalError: failed to load global configuration: "+
			"open testdata/missing_file.yaml: no such file or directory")
}

// TestMergeGlobalConfig tests the MergeGlobalConfig function.
func TestMergeGlobalConfig(t *testing.T) {
	ctx := context.Background()
	config := NewConfig(ctx,
		Config{GlobalConfigFile: parentDir + GlobalConfigFilename, PluginConfigFile: parentDir + PluginsConfigFilename})
	err := config.InitConfig(ctx)
	require.Nil(t, err)
	// The default log level is info.
	assert.Equal(t, DefaultLogLevel, config.Global.Loggers[Default].Level)

	// Merge a config that sets the log level to debug.
	err = config.MergeGlobalConfig(ctx, map[string]interface{}{
		"loggers": map[string]interface{}{
			"default": map[string]interface{}{
				"level": "debug",
			},
		},
	})
	require.Nil(t, err)
	assert.NotNil(t, config.Global)
	assert.NotEqual(t, GlobalConfig{}, config.Global)
	// The log level should now be debug.
	assert.Equal(t, "debug", config.Global.Loggers[Default].Level)
}

// initializeConfig initializes the configuration with the given context.
// It returns a pointer to the Config struct. If configuration initialization fails,
// the test will fail with an error message.
func initializeConfig(ctx context.Context, t *testing.T) *Config {
	t.Helper()
	config := NewConfig(ctx, Config{
		GlobalConfigFile: parentDir + GlobalConfigFilename,
		PluginConfigFile: parentDir + PluginsConfigFilename,
	})
	err := config.InitConfig(ctx)
	require.Nil(t, err)
	return config
}

// serverLoadBalancerStrategyOverwrite sets the environment variable for server nested configuration
// and verifies that the configuration is correctly loaded with the expected value.
func ServerLoadBalancerStrategyOverwrite(t *testing.T) {
	t.Helper()
	ctx := context.Background()
	// Convert to uppercase
	upperDefaultGroup := strings.ToUpper(Default)

	// Format environment variable name
	envVarName := fmt.Sprintf("GATEWAYD_SERVERS_%s_LOADBALANCER_STRATEGY", upperDefaultGroup)

	// Set the environment variable
	t.Setenv(envVarName, "test")
	config := initializeConfig(ctx, t)
	assert.Equal(t, "test", config.Global.Servers[Default].LoadBalancer.Strategy)
}

// pluginDefaultPolicyOverwrite sets the environment variable for plugin configuration
// and verifies that the configuration is correctly loaded with the expected value.
func pluginDefaultPolicyOverwrite(t *testing.T) {
	t.Helper()
	ctx := context.Background()

	// Set the environment variable
	t.Setenv("GATEWAYD_DEFAULTPOLICY", "test")
	config := initializeConfig(ctx, t)
	assert.Equal(t, "test", config.Plugin.DefaultPolicy)
}

// clientNetworkOverwrite sets the environment variable for client network configuration
// and verifies that the configuration is correctly loaded with the expected value.
func clientNetworkOverwrite(t *testing.T) {
	t.Helper()
	ctx := context.Background()
	// Convert to uppercase
	upperDefaultGroup := strings.ToUpper(Default)
	upperDefaultBlock := strings.ToUpper(DefaultConfigurationBlock)

	// Format environment variable name
	envVarName := fmt.Sprintf("GATEWAYD_CLIENTS_%s_%s_NETWORK", upperDefaultGroup, upperDefaultBlock)

	// Set the environment variable
	t.Setenv(envVarName, "udp")
	config := initializeConfig(ctx, t)
	assert.Equal(t, "udp", config.Global.Clients[Default][DefaultConfigurationBlock].Network)
}

// serverNetworkOverwrite sets the environment variable for server network configuration
// and verifies that the configuration is correctly loaded with the expected value.
func serverNetworkOverwrite(t *testing.T) {
	t.Helper()
	ctx := context.Background()
	// Convert to uppercase
	upperDefaultGroup := strings.ToUpper(Default)

	// Format environment variable name
	envVarName := fmt.Sprintf("GATEWAYD_SERVERS_%s_NETWORK", upperDefaultGroup)

	// Set the environment variable
	t.Setenv(envVarName, "udp")
	config := initializeConfig(ctx, t)
	assert.Equal(t, "udp", config.Global.Servers[Default].Network)
}

// TestLoadEnvVariables runs a suite of tests to verify that environment variables are correctly
// loaded into the configuration. Each test scenario sets a specific environment variable and
// checks if the configuration reflects the expected value.
func TestLoadEnvVariables(t *testing.T) {
	scenarios := map[string]func(t *testing.T){
		"serverLoadBalancerStrategyOverwrite": ServerLoadBalancerStrategyOverwrite,
		"pluginLocalPathOverwrite":            pluginDefaultPolicyOverwrite,
		"ClientNetworkOverwrite":              clientNetworkOverwrite,
		"ServerNetworkOverwrite":              serverNetworkOverwrite,
	}

	for scenario, fn := range scenarios {
		t.Run(scenario, func(t *testing.T) {
			fn(t)
		})
	}
}

// TestConvertKeysToLowercaseSuccess verifies that after calling ConvertKeysToLowercase,
// all keys in the config.Global.Clients map are converted to lowercase.
func TestConvertKeysToLowercaseSuccess(t *testing.T) {
	ctx := context.Background()
	config := NewConfig(ctx,
		Config{GlobalConfigFile: parentDir + GlobalConfigFilename, PluginConfigFile: parentDir + PluginsConfigFilename})

	err := config.ConvertKeysToLowercase(ctx)
	require.Nil(t, err)
	for configurationGroupName, configurationGroup := range config.Global.Clients {
		assert.Equal(t, configurationGroupName, strings.ToLower(configurationGroupName))
		for configuraionBlockName := range configurationGroup {
			assert.Equal(t, configuraionBlockName, strings.ToLower(configuraionBlockName))
		}
	}
}
