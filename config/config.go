package config

import (
	"context"
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
	InitConfig(context.Context)
	LoadDefaults(context.Context)
	LoadPluginEnvVars(context.Context)
	LoadGlobalEnvVars(context.Context)
	LoadGlobalConfigFile(context.Context)
	LoadPluginConfigFile(context.Context)
	MergeGlobalConfig(context.Context, map[string]interface{})
}

type Config struct {
	globalDefaults GlobalConfig
	pluginDefaults PluginConfig

	globalConfigFile string
	pluginConfigFile string

	GlobalKoanf *koanf.Koanf
	PluginKoanf *koanf.Koanf

	Global GlobalConfig
	Plugin PluginConfig
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
		globalDefaults:   GlobalConfig{},
		pluginDefaults:   PluginConfig{},
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
	c.ValidateGlobalConfig(newCtx)
	c.LoadGlobalEnvVars(newCtx)
	c.UnmarshalGlobalConfig(newCtx)
}

// LoadDefaults loads the default configuration before loading the config files.
func (c *Config) LoadDefaults(ctx context.Context) {
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
		Enabled: true,
		Address: DefaultMetricsAddress,
		Path:    DefaultMetricsPath,
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
	}

	defaultPool := Pool{
		Size: DefaultPoolSize,
	}

	defaultProxy := Proxy{
		Elastic:             false,
		ReuseElasticClients: false,
		HealthCheckPeriod:   DefaultHealthCheckPeriod,
	}

	defaultServer := Server{
		Network:          DefaultListenNetwork,
		Address:          DefaultListenAddress,
		EnableTicker:     false,
		TickInterval:     DefaultTickInterval,
		MultiCore:        true,
		LockOSThread:     false,
		ReuseAddress:     true,
		ReusePort:        true,
		LoadBalancer:     DefaultLoadBalancer,
		ReadBufferCap:    DefaultBufferSize,
		WriteBufferCap:   DefaultBufferSize,
		SocketRecvBuffer: DefaultBufferSize,
		SocketSendBuffer: DefaultBufferSize,
		TCPKeepAlive:     DefaultTCPKeepAliveDuration,
		TCPNoDelay:       DefaultTCPNoDelay,
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
	if contents, err := os.ReadFile(c.globalConfigFile); err == nil {
		gconf, err := yaml.Parser().Unmarshal(contents)
		if err != nil {
			span.RecordError(err)
			span.End()
			log.Panic(fmt.Errorf("failed to unmarshal global configuration: %w", err))
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
						log.Panic(err)
					}
				}
			}
		}
	} else if !os.IsNotExist(err) {
		span.RecordError(err)
		span.End()
		log.Panic(fmt.Errorf("failed to read global configuration file: %w", err))
	}

	c.pluginDefaults = PluginConfig{
		VerificationPolicy:  string(PassDown),
		CompatibilityPolicy: string(Strict),
		AcceptancePolicy:    string(Accept),
		TerminationPolicy:   string(Stop),
		EnableMetricsMerger: true,
		MetricsMergerPeriod: DefaultMetricsMergerPeriod,
		HealthCheckPeriod:   DefaultPluginHealthCheckPeriod,
		ReloadOnCrash:       true,
		Timeout:             DefaultPluginTimeout,
	}

	if c.GlobalKoanf != nil {
		if err := c.GlobalKoanf.Load(structs.Provider(c.globalDefaults, "json"), nil); err != nil {
			span.RecordError(err)
			span.End()
			log.Panic(fmt.Errorf("failed to load default global configuration: %w", err))
		}
	}

	if c.PluginKoanf != nil {
		if err := c.PluginKoanf.Load(structs.Provider(c.pluginDefaults, "json"), nil); err != nil {
			span.RecordError(err)
			span.End()
			log.Panic(fmt.Errorf("failed to load default plugin configuration: %w", err))
		}
	}

	span.End()
}

// LoadGlobalEnvVars loads the environment variables into the global configuration with the
// given prefix, "GATEWAYD_".
func (c *Config) LoadGlobalEnvVars(ctx context.Context) {
	_, span := otel.Tracer(TracerName).Start(ctx, "Load global environment variables")

	if err := c.GlobalKoanf.Load(loadEnvVars(), nil); err != nil {
		span.RecordError(err)
		span.End()
		log.Panic(fmt.Errorf("failed to load environment variables: %w", err))
	}

	span.End()
}

// LoadPluginEnvVars loads the environment variables into the plugins configuration with the
// given prefix, "GATEWAYD_".
func (c *Config) LoadPluginEnvVars(ctx context.Context) {
	_, span := otel.Tracer(TracerName).Start(ctx, "Load plugin environment variables")

	if err := c.PluginKoanf.Load(loadEnvVars(), nil); err != nil {
		span.RecordError(err)
		span.End()
		log.Panic(fmt.Errorf("failed to load environment variables: %w", err))
	}

	span.End()
}

func loadEnvVars() *env.Env {
	return env.Provider(EnvPrefix, ".", func(env string) string {
		return strings.ReplaceAll(strings.ToLower(strings.TrimPrefix(env, EnvPrefix)), "_", ".")
	})
}

// LoadGlobalConfig loads the plugin configuration file.
func (c *Config) LoadGlobalConfigFile(ctx context.Context) {
	_, span := otel.Tracer(TracerName).Start(ctx, "Load global config file")

	if err := c.GlobalKoanf.Load(file.Provider(c.globalConfigFile), yaml.Parser()); err != nil {
		span.RecordError(err)
		span.End()
		log.Panic(fmt.Errorf("failed to load global configuration: %w", err))
	}

	span.End()
}

// LoadPluginConfig loads the plugin configuration file.
func (c *Config) LoadPluginConfigFile(ctx context.Context) {
	_, span := otel.Tracer(TracerName).Start(ctx, "Load plugin config file")

	if err := c.PluginKoanf.Load(file.Provider(c.pluginConfigFile), yaml.Parser()); err != nil {
		span.RecordError(err)
		span.End()
		log.Panic(fmt.Errorf("failed to load plugin configuration: %w", err))
	}

	span.End()
}

// UnmarshalGlobalConfig unmarshals the global configuration for easier access.
func (c *Config) UnmarshalGlobalConfig(ctx context.Context) {
	_, span := otel.Tracer(TracerName).Start(ctx, "Unmarshal global config")

	if err := c.GlobalKoanf.UnmarshalWithConf("", &c.Global, koanf.UnmarshalConf{
		Tag: "json",
	}); err != nil {
		span.RecordError(err)
		span.End()
		log.Panic(fmt.Errorf("failed to unmarshal global configuration: %w", err))
	}

	span.End()
}

// UnmarshalPluginConfig unmarshals the plugin configuration for easier access.
func (c *Config) UnmarshalPluginConfig(ctx context.Context) {
	_, span := otel.Tracer(TracerName).Start(ctx, "Unmarshal plugin config")

	if err := c.PluginKoanf.UnmarshalWithConf("", &c.Plugin, koanf.UnmarshalConf{
		Tag: "json",
	}); err != nil {
		span.RecordError(err)
		span.End()
		log.Panic(fmt.Errorf("failed to unmarshal plugin configuration: %w", err))
	}

	span.End()
}

func (c *Config) MergeGlobalConfig(
	ctx context.Context, updatedGlobalConfig map[string]interface{},
) {
	_, span := otel.Tracer(TracerName).Start(ctx, "Merge global config from plugins")

	if err := c.GlobalKoanf.Load(confmap.Provider(updatedGlobalConfig, "."), nil); err != nil {
		span.RecordError(err)
		span.End()
		log.Panic(fmt.Errorf("failed to merge global configuration: %w", err))
	}

	if err := c.GlobalKoanf.UnmarshalWithConf("", &c.Global, koanf.UnmarshalConf{
		Tag: "json",
	}); err != nil {
		span.RecordError(err)
		span.End()
		log.Panic(fmt.Errorf("failed to unmarshal global configuration: %w", err))
	}

	span.End()
}

func (c *Config) ValidateGlobalConfig(ctx context.Context) {
	_, span := otel.Tracer(TracerName).Start(ctx, "Validate global config")

	var globalConfig GlobalConfig
	if err := c.GlobalKoanf.Unmarshal("", &globalConfig); err != nil {
		span.RecordError(err)
		span.End()
		log.Panic(
			gerr.ErrValidationFailed.Wrap(
				fmt.Errorf("failed to unmarshal global configuration: %w", err)),
		)
	}

	var errors []*gerr.GatewayDError
	configObjects := []string{"loggers", "metrics", "clients", "pools", "proxies", "servers"}
	sort.Strings(configObjects)
	seenConfigObjects := []string{}

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
		span.RecordError(fmt.Errorf("failed to validate global configuration"))
		span.End()
		log.Panic("failed to validate global configuration")
	}
}
