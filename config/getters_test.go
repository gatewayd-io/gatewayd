package config

import (
	"context"
	"testing"
	"time"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
)

// TestGetVerificationPolicy tests the GetVerificationPolicy function.
func TestGetVerificationPolicy(t *testing.T) {
	pluginConfig := PluginConfig{}
	assert.Equal(t, PassDown, pluginConfig.GetVerificationPolicy())
}

// TestGetPluginCompatibilityPolicy tests the GetPluginCompatibilityPolicy function.
func TestGetPluginCompatibilityPolicy(t *testing.T) {
	pluginConfig := PluginConfig{}
	assert.Equal(t, Strict, pluginConfig.GetPluginCompatibilityPolicy())
}

// TestGetAcceptancePolicy tests the GetAcceptancePolicy function.
func TestGetAcceptancePolicy(t *testing.T) {
	pluginConfig := PluginConfig{}
	assert.Equal(t, Accept, pluginConfig.GetAcceptancePolicy())
}

// TestGetTerminationPolicy tests the GetTerminationPolicy function.
func TestGetTerminationPolicy(t *testing.T) {
	pluginConfig := PluginConfig{}
	assert.Equal(t, Stop, pluginConfig.GetTerminationPolicy())
}

// TestGetTCPKeepAlivePeriod tests the GetTCPKeepAlivePeriod function.
func TestGetTCPKeepAlivePeriod(t *testing.T) {
	client := Client{}
	assert.Equal(t, client.GetTCPKeepAlivePeriod(), time.Duration(0))
}

// TestGetReceiveDeadline tests the GetReceiveDeadline function.
func TestGetReceiveDeadline(t *testing.T) {
	client := Client{}
	assert.Equal(t, time.Duration(0), client.GetReceiveDeadline())
}

// TestGetReceiveTimeout tests the GetReceiveTimeout function.
func TestGetReceiveTimeout(t *testing.T) {
	client := Client{}
	assert.Equal(t, time.Duration(0), client.GetReceiveTimeout())
}

// TestGetSendDeadline tests the GetSendDeadline function.
func TestGetSendDeadline(t *testing.T) {
	client := Client{}
	assert.Equal(t, time.Duration(0), client.GetSendDeadline())
}

// TestGetReceiveChunkSize tests the GetReceiveChunkSize function.
func TestGetReceiveChunkSize(t *testing.T) {
	client := Client{}
	assert.Equal(t, DefaultChunkSize, client.GetReceiveChunkSize())
}

// TestGetHealthCheckPeriod tests the GetHealthCheckPeriod function.
func TestGetHealthCheckPeriod(t *testing.T) {
	proxy := Proxy{}
	assert.Equal(t, DefaultHealthCheckPeriod, proxy.GetHealthCheckPeriod())
}

// TestGetTickInterval tests the GetTickInterval function.
func TestGetTickInterval(t *testing.T) {
	server := Server{}
	assert.Equal(t, server.GetTickInterval(), time.Duration(0))
}

// TestGetSize tests the GetSize function.
func TestGetSize(t *testing.T) {
	pool := Pool{}
	assert.Equal(t, DefaultPoolSize, pool.GetSize())
}

// TestGetOutput tests the GetOutput function.
func TestGetOutput(t *testing.T) {
	logger := Logger{}
	assert.Equal(t, []LogOutput{Console}, logger.GetOutput())
}

// TestGetTimeFormat tests the GetTimeFormat function.
func TestGetTimeFormat(t *testing.T) {
	logger := Logger{}
	assert.Equal(t, zerolog.TimeFormatUnix, logger.GetTimeFormat())
}

// TestGetConsoleTimeFormat tests the GetConsoleTimeFormat function.
func TestGetConsoleTimeFormat(t *testing.T) {
	logger := Logger{}
	assert.Equal(t, time.RFC3339, logger.GetConsoleTimeFormat())
}

// TestGetLevel tests the GetLevel function.
func TestGetLevel(t *testing.T) {
	logger := Logger{}
	assert.Equal(t, zerolog.InfoLevel, logger.GetLevel())
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

// TestGetReadTimeout tests the GetReadTimeout function.
func TestGetReadHeaderTimeout(t *testing.T) {
	metrics := Metrics{}
	assert.Equal(t, metrics.GetReadHeaderTimeout(), time.Duration(0))
}

// TestGetTimeout tests the GetTimeout function of the metrics server.
func TestGetTimeout(t *testing.T) {
	metrics := Metrics{}
	assert.Equal(t, metrics.GetTimeout(), time.Duration(0))
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
