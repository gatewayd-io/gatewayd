package config

import (
	"context"
	"testing"

	"github.com/knadh/koanf"
	"github.com/stretchr/testify/assert"
)

var parentDir = "../"

// TestNewConfig tests the NewConfig function.
func TestNewConfig(t *testing.T) {
	config := NewConfig(
		context.Background(), GlobalConfigFilename, PluginsConfigFilename)
	assert.NotNil(t, config)
	assert.Equal(t, GlobalConfigFilename, config.globalConfigFile)
	assert.Equal(t, PluginsConfigFilename, config.pluginConfigFile)
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
	config := NewConfig(ctx, parentDir+GlobalConfigFilename, parentDir+PluginsConfigFilename)
	config.InitConfig(ctx)
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
	assert.Equal(t, string(PassDown), config.PluginKoanf.String("verificationPolicy"))
	assert.NotNil(t, config.globalDefaults)
	assert.NotEqual(t, GlobalConfig{}, config.globalDefaults)
	assert.Contains(t, config.globalDefaults.Servers, Default)
	assert.NotNil(t, config.pluginDefaults)
	assert.NotEqual(t, PluginConfig{}, config.pluginDefaults)
	assert.Empty(t, config.pluginDefaults.Plugins)
}

// TestMergeGlobalConfig tests the MergeGlobalConfig function.
func TestMergeGlobalConfig(t *testing.T) {
	ctx := context.Background()
	config := NewConfig(ctx, parentDir+GlobalConfigFilename, parentDir+PluginsConfigFilename)
	config.InitConfig(ctx)
	// The default log level is info.
	assert.Equal(t, DefaultLogLevel, config.Global.Loggers[Default].Level)

	// Merge a config that sets the log level to debug.
	config.MergeGlobalConfig(ctx, map[string]interface{}{
		"loggers": map[string]interface{}{
			"default": map[string]interface{}{
				"level": "debug",
			},
		},
	})
	assert.NotNil(t, config.Global)
	assert.NotEqual(t, GlobalConfig{}, config.Global)
	// The log level should now be debug.
	assert.Equal(t, "debug", config.Global.Loggers[Default].Level)
}
