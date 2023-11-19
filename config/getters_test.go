package config

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestGetOutput tests the GetOutput function.
func TestGetOutput(t *testing.T) {
	logger := Logger{}
	assert.Equal(t, []LogOutput{Console}, logger.GetOutput())
}

// TestGetPlugins tests the GetPlugins function.
func TestGetPlugins(t *testing.T) {
	plugin := Plugin{Name: "plugin1"}
	pluginConfig := PluginConfig{Plugins: []Plugin{plugin}}
	assert.Equal(t, []Plugin{plugin}, pluginConfig.GetPlugins("plugin1"))
}

// TestGetDefaultConfigFilePath tests the GetDefaultConfigFilePath function.
func TestGetDefaultConfigFilePath(t *testing.T) {
	assert.Equal(t, GlobalConfigFilename, GetDefaultConfigFilePath(GlobalConfigFilename))
}

// TestFilter tests the Filter function.
func TestFilter(t *testing.T) {
	// Load config from the default config file.
	conf := NewConfig(context.TODO(), "../gatewayd.yaml", "../gatewayd_plugins.yaml")
	conf.InitConfig(context.TODO())
	assert.NotEmpty(t, conf.Global)

	// Filter the config.
	defaultGroup := conf.Global.Filter(Default)
	assert.NotEmpty(t, defaultGroup)
	assert.Contains(t, defaultGroup.Clients, Default)
	assert.Contains(t, defaultGroup.Servers, Default)
	assert.Contains(t, defaultGroup.Pools, Default)
	assert.Contains(t, defaultGroup.Proxies, Default)
	assert.Contains(t, defaultGroup.Metrics, Default)
	assert.Contains(t, defaultGroup.Loggers, Default)
}
