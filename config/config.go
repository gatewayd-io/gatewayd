package config

import (
	"fmt"
	"os"
	"strings"

	gerr "github.com/gatewayd-io/gatewayd/errors"
	"github.com/knadh/koanf"
	"github.com/knadh/koanf/parsers/yaml"
	"github.com/knadh/koanf/providers/confmap"
	"github.com/knadh/koanf/providers/env"
	"github.com/knadh/koanf/providers/file"
	"github.com/rs/zerolog"
)

type IConfig interface {
	LoadDefaults()
	LoadPluginEnvVars()
	LoadGlobalEnvVars()
	LoadGlobalConfigFile()
	LoadPluginConfigFile()
	MergeGlobalConfig(map[string]interface{})
}

type Config struct {
	globalDefaults   map[string]interface{}
	pluginDefaults   map[string]interface{}
	globalConfigFile string
	pluginConfigFile string
	logger           zerolog.Logger

	GlobalKoanf *koanf.Koanf
	PluginKoanf *koanf.Koanf
	Global      GlobalConfig
	Plugin      PluginConfig
}

var _ IConfig = &Config{}

func NewConfig(globalConfigFile, pluginConfigFile string, logger zerolog.Logger) *Config {
	config := Config{
		GlobalKoanf:      koanf.New("."),
		PluginKoanf:      koanf.New("."),
		globalDefaults:   make(map[string]interface{}),
		pluginDefaults:   make(map[string]interface{}),
		globalConfigFile: globalConfigFile,
		pluginConfigFile: pluginConfigFile,
		logger:           logger,
	}

	config.LoadDefaults()

	config.LoadPluginConfigFile()
	config.LoadPluginEnvVars()
	config.UnmarshalPluginConfig()

	config.LoadGlobalConfigFile()
	config.LoadGlobalEnvVars()
	config.UnmarshalGlobalConfig()

	return &config
}

// LoadDefaults loads the default configuration before loading the config files.
func (c *Config) LoadDefaults() {
	c.globalDefaults = map[string]interface{}{
		"loggers": map[string]interface{}{
			"default": map[string]interface{}{
				"output":            DefaultLogOutput,
				"level":             DefaultLogLevel,
				"timeFormat":        DefaultTimeFormat,
				"consoleTimeFormat": DefaultConsoleTimeFormat,
				"fileName":          DefaultLogFileName,
				"maxSize":           DefaultMaxSize,
				"maxBackups":        DefaultMaxBackups,
				"maxAge":            DefaultMaxAge,
				"compress":          DefaultCompress,
				"localTime":         DefaultLocalTime,
			},
		},
		"clients": map[string]interface{}{
			"default": map[string]interface{}{
				"receiveBufferSize":  DefaultBufferSize,
				"receiveChunkSize":   DefaultChunkSize,
				"tcpKeepAlivePeriod": DefaultTCPKeepAlivePeriod.String(),
			},
		},
		"pools": map[string]interface{}{
			"default": map[string]interface{}{
				"size": DefaultPoolSize,
			},
		},
		"proxy": map[string]interface{}{
			"default": map[string]interface{}{
				"elastic":             false,
				"reuseElasticClients": false,
				"healthCheckPeriod":   DefaultHealthCheckPeriod.String(),
			},
		},
		"server": map[string]interface{}{
			"network":          DefaultListenNetwork,
			"address":          DefaultListenAddress,
			"softLimit":        0,
			"hardLimit":        0,
			"enableTicker":     false,
			"multiCore":        true,
			"lockOSThread":     false,
			"reuseAddress":     true,
			"reusePort":        true,
			"loadBalancer":     DefaultLoadBalancer,
			"readBufferCap":    DefaultBufferSize,
			"writeBufferCap":   DefaultBufferSize,
			"socketRecvBuffer": DefaultBufferSize,
			"socketSendBuffer": DefaultBufferSize,
		},
		"metrics": map[string]interface{}{
			"default": map[string]interface{}{
				"enabled": true,
				"address": DefaultMetricsAddress,
				"path":    DefaultMetricsPath,
			},
		},
	}

	c.pluginDefaults = map[string]interface{}{
		"plugins": map[string]interface{}{
			"verificationPolicy":  "passdown",
			"compatibilityPolicy": "strict",
			"acceptancePolicy":    "accept",
			"metricsMergerPeriod": DefaultMetricsMergerPeriod.String(),
		},
	}

	if err := c.GlobalKoanf.Load(confmap.Provider(c.globalDefaults, ""), nil); err != nil {
		panic(fmt.Errorf("failed to load default global configuration: %w", err))
	}

	if err := c.PluginKoanf.Load(confmap.Provider(c.pluginDefaults, ""), nil); err != nil {
		panic(fmt.Errorf("failed to load default plugin configuration: %w", err))
	}
}

// LoadGlobalEnvVars loads the environment variables into the global configuration with the
// given prefix, "GATEWAYD_".
func (c *Config) LoadGlobalEnvVars() {
	if err := c.GlobalKoanf.Load(loadEnvVars(), nil); err != nil {
		panic(fmt.Errorf("failed to load environment variables: %w", err))
	}
}

// LoadPluginEnvVars loads the environment variables into the plugins configuration with the
// given prefix, "GATEWAYD_".
func (c *Config) LoadPluginEnvVars() {
	if err := c.PluginKoanf.Load(loadEnvVars(), nil); err != nil {
		panic(fmt.Errorf("failed to load environment variables: %w", err))
	}
}

func loadEnvVars() *env.Env {
	return env.Provider(EnvPrefix, ".", func(env string) string {
		return strings.ReplaceAll(strings.ToLower(strings.TrimPrefix(env, EnvPrefix)), "_", ".")
	})
}

// LoadGlobalConfig loads the plugin configuration file.
func (c *Config) LoadGlobalConfigFile() {
	if err := c.GlobalKoanf.Load(file.Provider(c.globalConfigFile), yaml.Parser()); err != nil {
		c.logger.Fatal().Err(err).Msg("Failed to load plugin configuration")
		os.Exit(gerr.FailedToLoadPluginConfig) // panic here?
	}
}

// LoadPluginConfig loads the plugin configuration file.
func (c *Config) LoadPluginConfigFile() {
	if err := c.PluginKoanf.Load(file.Provider(c.pluginConfigFile), yaml.Parser()); err != nil {
		c.logger.Fatal().Err(err).Msg("Failed to load plugin configuration")
		os.Exit(gerr.FailedToLoadPluginConfig) // panic here?
	}
}

// UnmarshalGlobalConfig unmarshals the global configuration for easier access.
func (c *Config) UnmarshalGlobalConfig() {
	if err := c.GlobalKoanf.Unmarshal("", &c.Global); err != nil {
		c.logger.Fatal().Err(err).Msg("Failed to unmarshal global configuration")
		os.Exit(gerr.FailedToLoadPluginConfig) // panic here?
	}
}

// UnmarshalPluginConfig unmarshals the plugin configuration for easier access.
func (c *Config) UnmarshalPluginConfig() {
	if err := c.PluginKoanf.Unmarshal("", &c.Plugin); err != nil {
		c.logger.Fatal().Err(err).Msg("Failed to unmarshal plugin configuration")
		os.Exit(gerr.FailedToLoadPluginConfig) // panic here?
	}
}

func (c *Config) MergeGlobalConfig(updatedGlobalConfig map[string]interface{}) {
	if err := c.GlobalKoanf.Load(confmap.Provider(updatedGlobalConfig, "."), nil); err != nil {
		c.logger.Fatal().Err(err).Msg("Failed to merge configuration")
	}

	if err := c.GlobalKoanf.Unmarshal("", &c.Global); err != nil {
		c.logger.Fatal().Err(err).Msg("Failed to unmarshal plugin configuration")
	}
}
