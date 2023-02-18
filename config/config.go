package config

import (
	"context"
	"fmt"
	"log"
	"strings"

	"github.com/knadh/koanf"
	"github.com/knadh/koanf/parsers/yaml"
	"github.com/knadh/koanf/providers/confmap"
	"github.com/knadh/koanf/providers/env"
	"github.com/knadh/koanf/providers/file"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
)

type IConfig interface {
	InitConfig(context.Context)
	LoadDefaults(context.Context)
	LoadPluginEnvVars(context.Context)
	LoadGlobalEnvVars(context.Context)
	LoadGlobalConfigFile(context.Context)
	LoadPluginConfigFile(context.Context)
	MergeGlobalConfig(context.Context, map[string]interface{})
}

type Config struct {
	globalDefaults   map[string]interface{}
	pluginDefaults   map[string]interface{}
	globalConfigFile string
	pluginConfigFile string

	GlobalKoanf *koanf.Koanf
	PluginKoanf *koanf.Koanf
	Global      GlobalConfig
	Plugin      PluginConfig
}

var _ IConfig = &Config{}

func NewConfig(ctx context.Context, globalConfigFile, pluginConfigFile string) *Config {
	_, span := otel.Tracer(TracerName).Start(ctx, "Create new config")
	defer span.End()
	span.SetAttributes(attribute.String("globalConfigFile", globalConfigFile))
	span.SetAttributes(attribute.String("pluginConfigFile", pluginConfigFile))

	return &Config{
		GlobalKoanf:      koanf.New("."),
		PluginKoanf:      koanf.New("."),
		globalDefaults:   make(map[string]interface{}),
		pluginDefaults:   make(map[string]interface{}),
		globalConfigFile: globalConfigFile,
		pluginConfigFile: pluginConfigFile,
	}
}

func (c *Config) InitConfig(ctx context.Context) {
	newCtx, span := otel.Tracer(TracerName).Start(ctx, "Initialize config")
	defer span.End()

	c.LoadDefaults(newCtx)

	c.LoadPluginConfigFile(newCtx)
	c.LoadPluginEnvVars(newCtx)
	c.UnmarshalPluginConfig(newCtx)

	c.LoadGlobalConfigFile(newCtx)
	c.LoadGlobalEnvVars(newCtx)
	c.UnmarshalGlobalConfig(newCtx)
}

// LoadDefaults loads the default configuration before loading the config files.
func (c *Config) LoadDefaults(ctx context.Context) {
	_, span := otel.Tracer(TracerName).Start(ctx, "Load defaults")
	defer span.End()

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
				"rsyslogNetwork":    DefaultRSyslogNetwork,
				"rsyslogAddress":    DefaultRSyslogAddress,
				"syslogPriority":    DefaultSyslogPriority,
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
		"servers": map[string]interface{}{
			"default": map[string]interface{}{
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
		"verificationPolicy":  "passdown",
		"compatibilityPolicy": "strict",
		"acceptancePolicy":    "accept",
		"enableMetricsMerger": true,
		"metricsMergerPeriod": DefaultMetricsMergerPeriod.String(),
		"healthCheckPeriod":   DefaultPluginHealthCheckPeriod.String(),
	}

	if err := c.GlobalKoanf.Load(confmap.Provider(c.globalDefaults, ""), nil); err != nil {
		log.Fatal(fmt.Errorf("failed to load default global configuration: %w", err))
		span.RecordError(err)
	}

	if err := c.PluginKoanf.Load(confmap.Provider(c.pluginDefaults, ""), nil); err != nil {
		log.Fatal(fmt.Errorf("failed to load default plugin configuration: %w", err))
		span.RecordError(err)
	}
}

// LoadGlobalEnvVars loads the environment variables into the global configuration with the
// given prefix, "GATEWAYD_".
func (c *Config) LoadGlobalEnvVars(ctx context.Context) {
	_, span := otel.Tracer(TracerName).Start(ctx, "Load global environment variables")
	defer span.End()

	if err := c.GlobalKoanf.Load(loadEnvVars(), nil); err != nil {
		log.Fatal(fmt.Errorf("failed to load environment variables: %w", err))
		span.RecordError(err)
	}
}

// LoadPluginEnvVars loads the environment variables into the plugins configuration with the
// given prefix, "GATEWAYD_".
func (c *Config) LoadPluginEnvVars(ctx context.Context) {
	_, span := otel.Tracer(TracerName).Start(ctx, "Load plugin environment variables")
	defer span.End()

	if err := c.PluginKoanf.Load(loadEnvVars(), nil); err != nil {
		log.Fatal(fmt.Errorf("failed to load environment variables: %w", err))
		span.RecordError(err)
	}
}

func loadEnvVars() *env.Env {
	return env.Provider(EnvPrefix, ".", func(env string) string {
		return strings.ReplaceAll(strings.ToLower(strings.TrimPrefix(env, EnvPrefix)), "_", ".")
	})
}

// LoadGlobalConfig loads the plugin configuration file.
func (c *Config) LoadGlobalConfigFile(ctx context.Context) {
	_, span := otel.Tracer(TracerName).Start(ctx, "Load global config file")
	defer span.End()

	if err := c.GlobalKoanf.Load(file.Provider(c.globalConfigFile), yaml.Parser()); err != nil {
		log.Fatal(fmt.Errorf("failed to load global configuration: %w", err))
		span.RecordError(err)
	}
}

// LoadPluginConfig loads the plugin configuration file.
func (c *Config) LoadPluginConfigFile(ctx context.Context) {
	_, span := otel.Tracer(TracerName).Start(ctx, "Load plugin config file")
	defer span.End()

	if err := c.PluginKoanf.Load(file.Provider(c.pluginConfigFile), yaml.Parser()); err != nil {
		log.Fatal(fmt.Errorf("failed to load plugin configuration: %w", err))
		span.RecordError(err)
	}
}

// UnmarshalGlobalConfig unmarshals the global configuration for easier access.
func (c *Config) UnmarshalGlobalConfig(ctx context.Context) {
	_, span := otel.Tracer(TracerName).Start(ctx, "Unmarshal global config")
	defer span.End()

	if err := c.GlobalKoanf.Unmarshal("", &c.Global); err != nil {
		log.Fatal(fmt.Errorf("failed to unmarshal global configuration: %w", err))
		span.RecordError(err)
	}
}

// UnmarshalPluginConfig unmarshals the plugin configuration for easier access.
func (c *Config) UnmarshalPluginConfig(ctx context.Context) {
	_, span := otel.Tracer(TracerName).Start(ctx, "Unmarshal plugin config")
	defer span.End()

	if err := c.PluginKoanf.Unmarshal("", &c.Plugin); err != nil {
		log.Fatal(fmt.Errorf("failed to unmarshal plugin configuration: %w", err))
		span.RecordError(err)
	}
}

func (c *Config) MergeGlobalConfig(
	ctx context.Context, updatedGlobalConfig map[string]interface{},
) {
	_, span := otel.Tracer(TracerName).Start(ctx, "Merge global config from plugins")
	defer span.End()

	if err := c.GlobalKoanf.Load(confmap.Provider(updatedGlobalConfig, "."), nil); err != nil {
		log.Fatal(fmt.Errorf("failed to merge global configuration: %w", err))
		span.RecordError(err)
	}

	if err := c.GlobalKoanf.Unmarshal("", &c.Global); err != nil {
		log.Fatal(fmt.Errorf("failed to unmarshal global configuration: %w", err))
		span.RecordError(err)
	}
}
