package config

import (
	"context"
	"testing"

	"github.com/knadh/koanf"
	"github.com/stretchr/testify/assert"
)

// TestNewConfig tests the NewConfig function.
func TestNewConfig(t *testing.T) {
	config := NewConfig(
		context.Background(), GlobalConfigFilename, PluginsConfigFilename)
	assert.NotNil(t, config)
	assert.Equal(t, config.globalConfigFile, GlobalConfigFilename)
	assert.Equal(t, config.pluginConfigFile, PluginsConfigFilename)
	assert.Equal(t, config.globalDefaults, GlobalConfig{})
	assert.Equal(t, config.pluginDefaults, PluginConfig{})
	assert.Equal(t, config.Global, GlobalConfig{})
	assert.Equal(t, config.Plugin, PluginConfig{})
	assert.Equal(t, config.GlobalKoanf, koanf.New("."))
	assert.Equal(t, config.PluginKoanf, koanf.New("."))
}

// TestInitConfig tests the InitConfig function, which practically tests all
// the other functions.
func TestInitConfig(t *testing.T) {
	ctx := context.Background()
	config := NewConfig(ctx, "../"+GlobalConfigFilename, "../"+PluginsConfigFilename)
	config.InitConfig(ctx)
	assert.NotNil(t, config.Global)
	assert.NotEqual(t, config.Global, GlobalConfig{})
	assert.Contains(t, config.Global.Servers, Default)
	assert.NotNil(t, config.Plugin)
	assert.NotEqual(t, config.Plugin, PluginConfig{})
	assert.Len(t, config.Plugin.Plugins, 1)
	assert.NotNil(t, config.GlobalKoanf)
	assert.NotEqual(t, config.GlobalKoanf, koanf.New("."))
	assert.Equal(t, DefaultLogLevel, config.GlobalKoanf.String("loggers.default.level"))
	assert.NotNil(t, config.PluginKoanf)
	assert.NotEqual(t, config.PluginKoanf, koanf.New("."))
	assert.Equal(t, string(PassDown), config.PluginKoanf.String("verificationPolicy"))
	assert.NotNil(t, config.globalDefaults)
	assert.NotEqual(t, config.globalDefaults, GlobalConfig{})
	assert.Contains(t, config.globalDefaults.Servers, Default)
	assert.NotNil(t, config.pluginDefaults)
	assert.NotEqual(t, config.pluginDefaults, PluginConfig{})
	assert.Len(t, config.pluginDefaults.Plugins, 0)
}

// TestMergeGlobalConfig tests the MergeGlobalConfig function.
func TestMergeGlobalConfig(t *testing.T) {
	ctx := context.Background()
	config := NewConfig(ctx, "../"+GlobalConfigFilename, "../"+PluginsConfigFilename)
	config.InitConfig(ctx)
	// The default log level is info.
	assert.Equal(t, config.Global.Loggers[Default].Level, DefaultLogLevel)

	// Merge a config that sets the log level to debug.
	config.MergeGlobalConfig(ctx, map[string]interface{}{
		"loggers": map[string]interface{}{
			"default": map[string]interface{}{
				"level": "debug",
			},
		},
	})
	assert.NotNil(t, config.Global)
	assert.NotEqual(t, config.Global, GlobalConfig{})
	// The log level should now be debug.
	assert.Equal(t, config.Global.Loggers[Default].Level, "debug")
}
