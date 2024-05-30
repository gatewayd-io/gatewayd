package config

import (
	"context"
	goerrors "errors"
	"fmt"
	"log"
	"os"
	"reflect"
	"sort"
	"strings"

	gerr "github.com/gatewayd-io/gatewayd/errors"
	"github.com/knadh/koanf"
	"github.com/knadh/koanf/parsers/yaml"
	"github.com/knadh/koanf/providers/confmap"
	"github.com/knadh/koanf/providers/env"
	"github.com/knadh/koanf/providers/file"
	"github.com/knadh/koanf/providers/structs"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"golang.org/x/exp/slices"
)

type IConfig interface {
	InitConfig(ctx context.Context) *gerr.GatewayDError
	LoadDefaults(ctx context.Context) *gerr.GatewayDError
	LoadPluginEnvVars(ctx context.Context) *gerr.GatewayDError
	LoadGlobalEnvVars(ctx context.Context) *gerr.GatewayDError
	LoadGlobalConfigFile(ctx context.Context) *gerr.GatewayDError
	LoadPluginConfigFile(ctx context.Context) *gerr.GatewayDError
	MergeGlobalConfig(
		ctx context.Context, updatedGlobalConfig map[string]interface{}) *gerr.GatewayDError
}

type Config struct {
	globalDefaults GlobalConfig
	pluginDefaults PluginConfig

	GlobalConfigFile string
	PluginConfigFile string

	GlobalKoanf *koanf.Koanf
	PluginKoanf *koanf.Koanf

	Global GlobalConfig
	Plugin PluginConfig
}

var _ IConfig = (*Config)(nil)

func NewConfig(ctx context.Context, config Config) *Config {
	_, span := otel.Tracer(TracerName).Start(ctx, "Create new config")
	defer span.End()
	span.SetAttributes(attribute.String("GlobalConfigFile", config.GlobalConfigFile))
	span.SetAttributes(attribute.String("PluginConfigFile", config.PluginConfigFile))

	return &Config{
		GlobalKoanf:      koanf.New("."),
		PluginKoanf:      koanf.New("."),
		globalDefaults:   GlobalConfig{},
		pluginDefaults:   PluginConfig{},
		GlobalConfigFile: config.GlobalConfigFile,
		PluginConfigFile: config.PluginConfigFile,
	}
}

func (c *Config) InitConfig(ctx context.Context) *gerr.GatewayDError {
	newCtx, span := otel.Tracer(TracerName).Start(ctx, "Initialize config")
	defer span.End()

	if err := c.LoadDefaults(newCtx); err != nil {
		return err
	}

	if err := c.LoadPluginConfigFile(newCtx); err != nil {
		return err
	}
	if err := c.LoadPluginEnvVars(newCtx); err != nil {
		return err
	}
	if err := c.UnmarshalPluginConfig(newCtx); err != nil {
		return err
	}

	if err := c.LoadGlobalConfigFile(newCtx); err != nil {
		return err
	}
	if err := c.ValidateGlobalConfig(newCtx); err != nil {
		return err
	}
	if err := c.LoadGlobalEnvVars(newCtx); err != nil {
		return err
	}
	if err := c.UnmarshalGlobalConfig(newCtx); err != nil {
		return err
	}

	return nil
}

// LoadDefaults loads the default configuration before loading the config files.
func (c *Config) LoadDefaults(ctx context.Context) *gerr.GatewayDError {
	_, span := otel.Tracer(TracerName).Start(ctx, "Load defaults")

	defaultLogger := Logger{
		Output:            []string{DefaultLogOutput},
		Level:             DefaultLogLevel,
		NoColor:           DefaultNoColor,
		TimeFormat:        DefaultTimeFormat,
		ConsoleTimeFormat: DefaultConsoleTimeFormat,
		FileName:          DefaultLogFileName,
		MaxSize:           DefaultMaxSize,
		MaxBackups:        DefaultMaxBackups,
		MaxAge:            DefaultMaxAge,
		Compress:          DefaultCompress,
		LocalTime:         DefaultLocalTime,
		RSyslogNetwork:    DefaultRSyslogNetwork,
		RSyslogAddress:    DefaultRSyslogAddress,
		SyslogPriority:    DefaultSyslogPriority,
	}

	defaultMetric := Metrics{
		Enabled:           true,
		Address:           DefaultMetricsAddress,
		Path:              DefaultMetricsPath,
		ReadHeaderTimeout: DefaultReadHeaderTimeout,
	}

	defaultClient := Client{
		Network:            DefaultNetwork,
		Address:            DefaultAddress,
		TCPKeepAlive:       DefaultTCPKeepAlive,
		TCPKeepAlivePeriod: DefaultTCPKeepAlivePeriod,
		ReceiveChunkSize:   DefaultChunkSize,
		ReceiveDeadline:    DefaultReceiveDeadline,
		ReceiveTimeout:     DefaultReceiveTimeout,
		SendDeadline:       DefaultSendDeadline,
		DialTimeout:        DefaultDialTimeout,
		Retries:            DefaultRetries,
		Backoff:            DefaultBackoff,
		BackoffMultiplier:  DefaultBackoffMultiplier,
		DisableBackoffCaps: DefaultDisableBackoffCaps,
	}

	defaultPool := Pool{
		Size: DefaultPoolSize,
	}

	defaultProxy := Proxy{
		HealthCheckPeriod: DefaultHealthCheckPeriod,
	}

	defaultServer := Server{
		Network:          DefaultListenNetwork,
		Address:          DefaultListenAddress,
		EnableTicker:     false,
		TickInterval:     DefaultTickInterval,
		EnableTLS:        false,
		CertFile:         "",
		KeyFile:          "",
		HandshakeTimeout: DefaultHandshakeTimeout,
	}

	c.globalDefaults = GlobalConfig{
		Loggers: map[string]*Logger{Default: &defaultLogger},
		Metrics: map[string]*Metrics{Default: &defaultMetric},
		Clients: map[string]*Client{Default: &defaultClient},
		Pools:   map[string]*Pool{Default: &defaultPool},
		Proxies: map[string]*Proxy{Default: &defaultProxy},
		Servers: map[string]*Server{Default: &defaultServer},
		API: API{
			Enabled:     true,
			HTTPAddress: DefaultHTTPAPIAddress,
			GRPCNetwork: DefaultGRPCAPINetwork,
			GRPCAddress: DefaultGRPCAPIAddress,
		},
	}

	//nolint:nestif
	if contents, err := os.ReadFile(c.GlobalConfigFile); err == nil {
		gconf, err := yaml.Parser().Unmarshal(contents)
		if err != nil {
			span.RecordError(err)
			span.End()
			return gerr.ErrConfigParseError.Wrap(
				fmt.Errorf("failed to unmarshal global configuration: %w", err))
		}

		for configObject, configMap := range gconf {
			if configGroup, ok := configMap.(map[string]interface{}); ok {
				for configGroupKey := range configGroup {
					if configGroupKey == Default {
						continue
					}

					switch configObject {
					case "loggers":
						c.globalDefaults.Loggers[configGroupKey] = &defaultLogger
					case "metrics":
						c.globalDefaults.Metrics[configGroupKey] = &defaultMetric
					case "clients":
						c.globalDefaults.Clients[configGroupKey] = &defaultClient
					case "pools":
						c.globalDefaults.Pools[configGroupKey] = &defaultPool
					case "proxies":
						c.globalDefaults.Proxies[configGroupKey] = &defaultProxy
					case "servers":
						c.globalDefaults.Servers[configGroupKey] = &defaultServer
					case "api":
						// TODO: Add support for multiple API config groups.
					default:
						err := fmt.Errorf("unknown config object: %s", configObject)
						span.RecordError(err)
						span.End()
						return gerr.ErrConfigParseError.Wrap(err)
					}
				}
			}
		}
	} else if !os.IsNotExist(err) {
		span.RecordError(err)
		span.End()
		return gerr.ErrConfigParseError.Wrap(
			fmt.Errorf("failed to read global configuration file: %w", err))
	}

	c.pluginDefaults = PluginConfig{
		CompatibilityPolicy: string(Strict),
		EnableMetricsMerger: true,
		MetricsMergerPeriod: DefaultMetricsMergerPeriod,
		HealthCheckPeriod:   DefaultPluginHealthCheckPeriod,
		ReloadOnCrash:       true,
		Timeout:             DefaultPluginTimeout,
		StartTimeout:        DefaultPluginStartTimeout,
		DefaultPolicy:       DefaultPolicy,
		PolicyTimeout:       DefaultPolicyTimeout,
		ActionTimeout:       DefaultActionTimeout,
		Policies:            []Policy{},
		ActionRedis:         ActionRedisConfig{},
	}

	if c.GlobalKoanf != nil {
		if err := c.GlobalKoanf.Load(structs.Provider(c.globalDefaults, "json"), nil); err != nil {
			span.RecordError(err)
			span.End()
			return gerr.ErrConfigParseError.Wrap(
				fmt.Errorf("failed to load default global configuration: %w", err))
		}
	}

	if c.PluginKoanf != nil {
		if err := c.PluginKoanf.Load(structs.Provider(c.pluginDefaults, "json"), nil); err != nil {
			span.RecordError(err)
			span.End()
			return gerr.ErrConfigParseError.Wrap(
				fmt.Errorf("failed to load default plugin configuration: %w", err))
		}
	}

	span.End()

	return nil
}

// LoadGlobalEnvVars loads the environment variables into the global configuration with the
// given prefix, "GATEWAYD_".
func (c *Config) LoadGlobalEnvVars(ctx context.Context) *gerr.GatewayDError {
	_, span := otel.Tracer(TracerName).Start(ctx, "Load global environment variables")

	if err := c.GlobalKoanf.Load(loadEnvVars(), nil); err != nil {
		span.RecordError(err)
		span.End()
		return gerr.ErrConfigParseError.Wrap(
			fmt.Errorf("failed to load environment variables: %w", err))
	}

	span.End()

	return nil
}

// LoadPluginEnvVars loads the environment variables into the plugins configuration with the
// given prefix, "GATEWAYD_".
func (c *Config) LoadPluginEnvVars(ctx context.Context) *gerr.GatewayDError {
	_, span := otel.Tracer(TracerName).Start(ctx, "Load plugin environment variables")

	if err := c.PluginKoanf.Load(loadEnvVars(), nil); err != nil {
		span.RecordError(err)
		span.End()
		return gerr.ErrConfigParseError.Wrap(
			fmt.Errorf("failed to load environment variables: %w", err))
	}

	span.End()

	return nil
}

func loadEnvVars() *env.Env {
	return env.Provider(EnvPrefix, ".", func(env string) string {
		return strings.ReplaceAll(strings.ToLower(strings.TrimPrefix(env, EnvPrefix)), "_", ".")
	})
}

// LoadGlobalConfigFile loads the plugin configuration file.
func (c *Config) LoadGlobalConfigFile(ctx context.Context) *gerr.GatewayDError {
	_, span := otel.Tracer(TracerName).Start(ctx, "Load global config file")

	if err := c.GlobalKoanf.Load(file.Provider(c.GlobalConfigFile), yaml.Parser()); err != nil {
		span.RecordError(err)
		span.End()
		return gerr.ErrConfigParseError.Wrap(
			fmt.Errorf("failed to load global configuration: %w", err))
	}

	span.End()

	return nil
}

// LoadPluginConfigFile loads the plugin configuration file.
func (c *Config) LoadPluginConfigFile(ctx context.Context) *gerr.GatewayDError {
	_, span := otel.Tracer(TracerName).Start(ctx, "Load plugin config file")

	if err := c.PluginKoanf.Load(file.Provider(c.PluginConfigFile), yaml.Parser()); err != nil {
		span.RecordError(err)
		span.End()
		return gerr.ErrConfigParseError.Wrap(
			fmt.Errorf("failed to load plugin configuration: %w", err))
	}

	span.End()

	return nil
}

// UnmarshalGlobalConfig unmarshals the global configuration for easier access.
func (c *Config) UnmarshalGlobalConfig(ctx context.Context) *gerr.GatewayDError {
	_, span := otel.Tracer(TracerName).Start(ctx, "Unmarshal global config")

	if err := c.GlobalKoanf.UnmarshalWithConf("", &c.Global, koanf.UnmarshalConf{
		Tag: "json",
	}); err != nil {
		span.RecordError(err)
		span.End()
		return gerr.ErrConfigParseError.Wrap(
			fmt.Errorf("failed to unmarshal global configuration: %w", err))
	}

	span.End()

	return nil
}

// UnmarshalPluginConfig unmarshals the plugin configuration for easier access.
func (c *Config) UnmarshalPluginConfig(ctx context.Context) *gerr.GatewayDError {
	_, span := otel.Tracer(TracerName).Start(ctx, "Unmarshal plugin config")

	if err := c.PluginKoanf.UnmarshalWithConf("", &c.Plugin, koanf.UnmarshalConf{
		Tag: "json",
	}); err != nil {
		span.RecordError(err)
		span.End()
		return gerr.ErrConfigParseError.Wrap(
			fmt.Errorf("failed to unmarshal plugin configuration: %w", err))
	}

	span.End()

	return nil
}

func (c *Config) MergeGlobalConfig(
	ctx context.Context, updatedGlobalConfig map[string]interface{},
) *gerr.GatewayDError {
	_, span := otel.Tracer(TracerName).Start(ctx, "Merge global config from plugins")

	if err := c.GlobalKoanf.Load(confmap.Provider(updatedGlobalConfig, "."), nil); err != nil {
		span.RecordError(err)
		span.End()
		return gerr.ErrConfigParseError.Wrap(
			fmt.Errorf("failed to merge global configuration: %w", err))
	}

	if err := c.GlobalKoanf.UnmarshalWithConf("", &c.Global, koanf.UnmarshalConf{
		Tag: "json",
	}); err != nil {
		span.RecordError(err)
		span.End()
		return gerr.ErrConfigParseError.Wrap(
			fmt.Errorf("failed to unmarshal global configuration: %w", err))
	}

	span.End()

	return nil
}

func (c *Config) ValidateGlobalConfig(ctx context.Context) *gerr.GatewayDError {
	_, span := otel.Tracer(TracerName).Start(ctx, "Validate global config")

	var globalConfig GlobalConfig
	if err := c.GlobalKoanf.Unmarshal("", &globalConfig); err != nil {
		span.RecordError(err)
		span.End()
		return gerr.ErrValidationFailed.Wrap(
			fmt.Errorf("failed to unmarshal global configuration: %w", err))
	}

	var errors []*gerr.GatewayDError
	configObjects := []string{"loggers", "metrics", "clients", "pools", "proxies", "servers"}
	sort.Strings(configObjects)
	var seenConfigObjects []string

	for configGroup := range globalConfig.Loggers {
		if globalConfig.Loggers[configGroup] == nil {
			err := fmt.Errorf("\"logger.%s\" is nil or empty", configGroup)
			span.RecordError(err)
			errors = append(errors, gerr.ErrValidationFailed.Wrap(err))
		}
	}

	if len(globalConfig.Loggers) > 1 {
		seenConfigObjects = append(seenConfigObjects, "loggers")
	}

	for configGroup := range globalConfig.Metrics {
		if globalConfig.Metrics[configGroup] == nil {
			err := fmt.Errorf("\"metrics.%s\" is nil or empty", configGroup)
			span.RecordError(err)
			errors = append(errors, gerr.ErrValidationFailed.Wrap(err))
		}
	}

	if len(globalConfig.Metrics) > 1 {
		seenConfigObjects = append(seenConfigObjects, "metrics")
	}

	for configGroup := range globalConfig.Clients {
		if globalConfig.Clients[configGroup] == nil {
			err := fmt.Errorf("\"clients.%s\" is nil or empty", configGroup)
			span.RecordError(err)
			errors = append(errors, gerr.ErrValidationFailed.Wrap(err))
		}
	}

	if len(globalConfig.Clients) > 1 {
		seenConfigObjects = append(seenConfigObjects, "clients")
	}

	for configGroup := range globalConfig.Pools {
		if globalConfig.Pools[configGroup] == nil {
			err := fmt.Errorf("\"pools.%s\" is nil or empty", configGroup)
			span.RecordError(err)
			errors = append(errors, gerr.ErrValidationFailed.Wrap(err))
		}
	}

	if len(globalConfig.Pools) > 1 {
		seenConfigObjects = append(seenConfigObjects, "pools")
	}

	for configGroup := range globalConfig.Proxies {
		if globalConfig.Proxies[configGroup] == nil {
			err := fmt.Errorf("\"proxies.%s\" is nil or empty", configGroup)
			span.RecordError(err)
			errors = append(errors, gerr.ErrValidationFailed.Wrap(err))
		}
	}

	if len(globalConfig.Proxies) > 1 {
		seenConfigObjects = append(seenConfigObjects, "proxies")
	}

	for configGroup := range globalConfig.Servers {
		if globalConfig.Servers[configGroup] == nil {
			err := fmt.Errorf("\"servers.%s\" is nil or empty", configGroup)
			span.RecordError(err)
			errors = append(errors, gerr.ErrValidationFailed.Wrap(err))
		}
	}

	if len(globalConfig.Servers) > 1 {
		seenConfigObjects = append(seenConfigObjects, "servers")
	}

	sort.Strings(seenConfigObjects)

	if len(seenConfigObjects) > 0 && !reflect.DeepEqual(configObjects, seenConfigObjects) {
		// Find all strings in configObjects that are not in seenConfigObjects
		var missingConfigObjects []string
		for _, configObject := range configObjects {
			if !slices.Contains(seenConfigObjects, configObject) {
				missingConfigObjects = append(missingConfigObjects, configObject)
			}
		}

		// TODO: Add the actual missing config objects to the error message.
		for _, missingConfigObject := range missingConfigObjects {
			err := fmt.Errorf(
				"global config is missing one or more config group(s) for \"%s\"",
				missingConfigObject)
			span.RecordError(err)
			errors = append(errors, gerr.ErrValidationFailed.Wrap(err))
		}
	}

	if len(errors) > 0 {
		for _, err := range errors {
			log.Println(err)
		}
		err := goerrors.New("failed to validate global configuration")
		span.RecordError(err)
		span.End()
		return gerr.ErrValidationFailed.Wrap(err)
	}

	span.End()

	return nil
}
