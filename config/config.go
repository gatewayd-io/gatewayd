package config

import (
	"context"
	"encoding/json"
	goerrors "errors"
	"fmt"
	"log"
	"math"
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
	if err := c.ConvertKeysToLowercase(newCtx); err != nil {
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
		LoadBalancer:     LoadBalancer{Strategy: DefaultLoadBalancerStrategy},
	}

	c.globalDefaults = GlobalConfig{
		Loggers: map[string]*Logger{Default: &defaultLogger},
		Metrics: map[string]*Metrics{Default: &defaultMetric},
		Clients: map[string]map[string]*Client{Default: {DefaultConfigurationBlock: &defaultClient}},
		Pools:   map[string]map[string]*Pool{Default: {DefaultConfigurationBlock: &defaultPool}},
		Proxies: map[string]map[string]*Proxy{Default: {DefaultConfigurationBlock: &defaultProxy}},
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
			configGroup, ok := configMap.(map[string]any)
			if !ok {
				err := fmt.Errorf("invalid config structure for %s", configObject)
				span.RecordError(err)
				span.End()
				return gerr.ErrConfigParseError.Wrap(err)
			}

			if configObject == "api" {
				// Handle API configuration separately
				// TODO: Add support for multiple API config groups.
				continue
			}

			for configGroupKey, configBlocksInterface := range configGroup {
				if configGroupKey == Default {
					continue
				}

				configBlocks, ok := configBlocksInterface.(map[string]any)
				if !ok {
					err := fmt.Errorf("invalid config blocks structure for %s.%s", configObject, configGroupKey)
					span.RecordError(err)
					span.End()
					return gerr.ErrConfigParseError.Wrap(err)
				}

				for configBlockKey := range configBlocks {
					if configBlockKey == DefaultConfigurationBlock {
						continue
					}
					switch configObject {
					case "loggers":
						c.globalDefaults.Loggers[configGroupKey] = &defaultLogger
					case "metrics":
						c.globalDefaults.Metrics[configGroupKey] = &defaultMetric
					case "clients":
						if c.globalDefaults.Clients[configGroupKey] == nil {
							c.globalDefaults.Clients[configGroupKey] = make(map[string]*Client)
						}
						c.globalDefaults.Clients[configGroupKey][configBlockKey] = &defaultClient
					case "pools":
						if c.globalDefaults.Pools[configGroupKey] == nil {
							c.globalDefaults.Pools[configGroupKey] = make(map[string]*Pool)
						}
						c.globalDefaults.Pools[configGroupKey][configBlockKey] = &defaultPool
					case "proxies":
						if c.globalDefaults.Proxies[configGroupKey] == nil {
							c.globalDefaults.Proxies[configGroupKey] = make(map[string]*Proxy)
						}
						c.globalDefaults.Proxies[configGroupKey][configBlockKey] = &defaultProxy
					case "servers":
						c.globalDefaults.Servers[configGroupKey] = &defaultServer
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
		ActionRedis: ActionRedisConfig{
			Enabled: DefaultActionRedisEnabled,
			Address: DefaultRedisAddress,
			Channel: DefaultRedisChannel,
		},
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
	return env.Provider(EnvPrefix, ".", transformEnvVariable)
}

// transformEnvVariable transforms the environment variable name to a format based on JSON tags.
func transformEnvVariable(envVar string) string {
	structs := []interface{}{
		&API{},
		&Logger{},
		&Pool{},
		&Proxy{},
		&Server{},
		&Metrics{},
		&PluginConfig{},
	}
	tagMapping := make(map[string]string)
	generateTagMapping(structs, tagMapping)

	lowerEnvVar := strings.ToLower(strings.TrimPrefix(envVar, EnvPrefix))
	parts := strings.Split(lowerEnvVar, "_")

	var transformedParts strings.Builder

	for i, part := range parts {
		if i > 0 {
			transformedParts.WriteString(".")
		}
		if mappedValue, exists := tagMapping[part]; exists {
			transformedParts.WriteString(mappedValue)
		} else {
			transformedParts.WriteString(part)
		}
	}

	return transformedParts.String()
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

// convertMapKeysToLowercase converts all keys in a map to lowercase.
//
// Parameters:
//   - m: A map with string keys to be converted.
//
// Returns:
//   - map[string]*T: A new map with lowercase keys.
func convertMapKeysToLowercase[T any](m map[string]*T) map[string]*T {
	newMap := make(map[string]*T)
	for k, v := range m {
		lowercaseKey := strings.ToLower(k)
		newMap[lowercaseKey] = v
	}
	return newMap
}

// convertNestedMapKeysToLowercase converts all keys in a nested map structure to lowercase.
//
// Parameters:
//   - m: A nested map with string keys to be converted.
//
// Returns:
//   - map[string]map[string]*T: A new nested map with lowercase keys.
func convertNestedMapKeysToLowercase[T any](m map[string]map[string]*T) map[string]map[string]*T {
	newMap := make(map[string]map[string]*T)
	for k, v := range m {
		lowercaseKey := strings.ToLower(k)
		newMap[lowercaseKey] = convertMapKeysToLowercase(v)
	}
	return newMap
}

// ConvertKeysToLowercase converts all keys in the global configuration to lowercase.
// It unmarshals the configuration data into a GlobalConfig struct, then recursively converts
// all map keys to lowercase.
//
// Parameters:
//   - ctx (context.Context): The context for tracing and cancellation, used for monitoring
//     and propagating execution state.
//
// Returns:
//   - *gerr.GatewayDError: An error if unmarshalling fails, otherwise nil.
func (c *Config) ConvertKeysToLowercase(ctx context.Context) *gerr.GatewayDError {
	_, span := otel.Tracer(TracerName).Start(ctx, "Validate global config")

	defer span.End()

	var globalConfig GlobalConfig
	if err := c.GlobalKoanf.Unmarshal("", &globalConfig); err != nil {
		span.RecordError(err)
		return gerr.ErrValidationFailed.Wrap(
			fmt.Errorf("failed to unmarshal global configuration: %w", err))
	}

	globalConfig.Loggers = convertMapKeysToLowercase(globalConfig.Loggers)
	globalConfig.Clients = convertNestedMapKeysToLowercase(globalConfig.Clients)
	globalConfig.Pools = convertNestedMapKeysToLowercase(globalConfig.Pools)
	globalConfig.Proxies = convertNestedMapKeysToLowercase(globalConfig.Proxies)
	globalConfig.Servers = convertMapKeysToLowercase(globalConfig.Servers)
	globalConfig.Metrics = convertMapKeysToLowercase(globalConfig.Metrics)

	// Convert the globalConfig back to a map[string]interface{}
	configMap, err := structToMap(globalConfig)
	if err != nil {
		span.RecordError(err)
		return gerr.ErrValidationFailed.Wrap(
			fmt.Errorf("failed to convert global configuration to map: %w", err))
	}

	// Create a new koanf instance and load the updated map
	newKoanf := koanf.New(".")
	if err := newKoanf.Load(confmap.Provider(configMap, "."), nil); err != nil {
		span.RecordError(err)
		return gerr.ErrValidationFailed.Wrap(
			fmt.Errorf("failed to load updated configuration into koanf: %w", err))
	}

	// Update the GlobalKoanf with the new instance
	c.GlobalKoanf = newKoanf
	// Update the Global with the new instance
	c.Global = globalConfig

	return nil
}

// structToMap converts a given struct to a map[string]interface{}.
func structToMap(v interface{}) (map[string]interface{}, error) {
	var result map[string]interface{}
	data, err := json.Marshal(v)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal struct: %w", err)
	}
	err = json.Unmarshal(data, &result)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal data into map: %w", err)
	}
	return result, nil
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
		if configGroup != strings.ToLower(configGroup) {
			err := fmt.Errorf(`"logger.%s" is not lowercase`, configGroup)
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
		if configGroup != strings.ToLower(configGroup) {
			err := fmt.Errorf(`"metrics.%s" is not lowercase`, configGroup)
			span.RecordError(err)
			errors = append(errors, gerr.ErrValidationFailed.Wrap(err))
		}
	}

	if len(globalConfig.Metrics) > 1 {
		seenConfigObjects = append(seenConfigObjects, "metrics")
	}

	clientConfigGroups := make(map[string]map[string]bool)
	for configGroupName, configGroups := range globalConfig.Clients {
		if _, ok := clientConfigGroups[configGroupName]; !ok {
			clientConfigGroups[configGroupName] = make(map[string]bool)
		}
		if configGroupName != strings.ToLower(configGroupName) {
			err := fmt.Errorf(`"clients.%s" is not lowercase`, configGroupName)
			span.RecordError(err)
			errors = append(errors, gerr.ErrValidationFailed.Wrap(err))
		}
		for configBlockName := range configGroups {
			clientConfigGroups[configGroupName][configBlockName] = true
			if globalConfig.Clients[configGroupName][configBlockName] == nil {
				err := fmt.Errorf(`"clients.%s" is nil or empty`, configBlockName)
				span.RecordError(err)
				errors = append(errors, gerr.ErrValidationFailed.Wrap(err))
			}
			if configBlockName != strings.ToLower(configBlockName) {
				err := fmt.Errorf(`"clients.%s.%s" is not lowercase`, configGroupName, configBlockName)
				span.RecordError(err)
				errors = append(errors, gerr.ErrValidationFailed.Wrap(err))
			}
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
		if configGroup != strings.ToLower(configGroup) {
			err := fmt.Errorf(`"pools.%s" is not lowercase`, configGroup)
			span.RecordError(err)
			errors = append(errors, gerr.ErrValidationFailed.Wrap(err))
		}
	}

	if len(globalConfig.Pools) > 1 {
		seenConfigObjects = append(seenConfigObjects, "pools")
	}

	for configGroup := range globalConfig.Proxies {
		if globalConfig.Proxies[configGroup] == nil {
			err := fmt.Errorf(`"proxies.%s" is nil or empty`, configGroup)
			span.RecordError(err)
			errors = append(errors, gerr.ErrValidationFailed.Wrap(err))
		}
		if configGroup != strings.ToLower(configGroup) {
			err := fmt.Errorf(`"proxies.%s" is not lowercase`, configGroup)
			span.RecordError(err)
			errors = append(errors, gerr.ErrValidationFailed.Wrap(err))
		}
	}

	if len(globalConfig.Proxies) > 1 {
		seenConfigObjects = append(seenConfigObjects, "proxies")
	}

	for configGroup := range globalConfig.Servers {
		serverConfig := globalConfig.Servers[configGroup]

		if serverConfig == nil {
			err := fmt.Errorf("\"servers.%s\" is nil or empty", configGroup)
			span.RecordError(err)
			errors = append(errors, gerr.ErrValidationFailed.Wrap(err))
			continue
		}
		if configGroup != strings.ToLower(configGroup) {
			err := fmt.Errorf(`"servers.%s" is not lowercase`, configGroup)
			span.RecordError(err)
			errors = append(errors, gerr.ErrValidationFailed.Wrap(err))
		}

		// Validate Load Balancing Rules
		validatelBRulesErrors := ValidateLoadBalancingRules(serverConfig, configGroup, clientConfigGroups)
		for _, err := range validatelBRulesErrors {
			span.RecordError(err)
			errors = append(errors, gerr.ErrValidationFailed.Wrap(err))
		}
	}

	if len(globalConfig.Servers) > 1 {
		seenConfigObjects = append(seenConfigObjects, "servers")
	}

	// ValidateClientsPoolsProxies checks if all configGroups in globalConfig.Pools and globalConfig.Proxies
	// are referenced in globalConfig.Clients.
	if len(globalConfig.Clients) != len(globalConfig.Pools) || len(globalConfig.Clients) != len(globalConfig.Proxies) {
		err := goerrors.New("clients, pools, and proxies do not have the same number of objects")
		span.RecordError(err)
		errors = append(errors, gerr.ErrValidationFailed.Wrap(err))
	}

	// Check if all proxies are referenced in client configuration
	for configGroupName, configGroups := range globalConfig.Proxies {
		for configBlockName := range configGroups {
			if !clientConfigGroups[configGroupName][configBlockName] {
				err := fmt.Errorf(`"proxies.%s.%s" not referenced in client configuration`, configGroupName, configBlockName)
				span.RecordError(err)
				errors = append(errors, gerr.ErrValidationFailed.Wrap(err))
			}
			if configBlockName != strings.ToLower(configBlockName) {
				err := fmt.Errorf(`"proxies.%s.%s" is not lowercase`, configGroupName, configBlockName)
				span.RecordError(err)
				errors = append(errors, gerr.ErrValidationFailed.Wrap(err))
			}
		}
	}

	// Check if all pools are referenced in client configuration
	for configGroupName, configGroups := range globalConfig.Pools {
		for configBlockName := range configGroups {
			if !clientConfigGroups[configGroupName][configBlockName] {
				err := fmt.Errorf(`"pools.%s.%s" not referenced in client configuration`, configGroupName, configBlockName)
				span.RecordError(err)
				errors = append(errors, gerr.ErrValidationFailed.Wrap(err))
			}
			if configBlockName != strings.ToLower(configBlockName) {
				err := fmt.Errorf(`"pools.%s.%s" is not lowercase`, configGroupName, configBlockName)
				span.RecordError(err)
				errors = append(errors, gerr.ErrValidationFailed.Wrap(err))
			}
		}
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

// generateTagMapping generates a map of JSON tags to lower case json tags.
func generateTagMapping(structs []interface{}, tagMapping map[string]string) {
	for _, s := range structs {
		structValue := reflect.ValueOf(s).Elem()
		structType := structValue.Type()

		for i := range structValue.NumField() {
			field := structType.Field(i)
			fieldValue := structValue.Field(i)

			// Handle nested structs
			if field.Type.Kind() == reflect.Struct {
				generateTagMapping([]interface{}{fieldValue.Addr().Interface()}, tagMapping)
			}

			jsonTag := field.Tag.Get("json")
			if jsonTag != "" {
				tagMapping[strings.ToLower(jsonTag)] = jsonTag
			} else {
				tagMapping[strings.ToLower(field.Name)] = strings.ToLower(field.Name)
			}
		}
	}
}

// ValidateLoadBalancingRules validates the load balancing rules in the server configuration.
func ValidateLoadBalancingRules(
	serverConfig *Server,
	configGroup string,
	clientConfigGroups map[string]map[string]bool,
) []error {
	var errors []error

	// Return early if there are no load balancing rules
	if serverConfig.LoadBalancer.LoadBalancingRules == nil {
		return errors
	}

	// Validate each load balancing rule
	for _, rule := range serverConfig.LoadBalancer.LoadBalancingRules {
		// Validate the condition of the rule
		if err := validateRuleCondition(rule.Condition, configGroup); err != nil {
			errors = append(errors, err)
		}

		// Validate the distribution of the rule
		if err := validateDistribution(
			rule.Distribution,
			configGroup,
			rule.Condition,
			clientConfigGroups,
		); err != nil {
			errors = append(errors, err)
		}
	}

	return errors
}

// validateRuleCondition checks if the rule condition is empty for LoadBalancingRules.
func validateRuleCondition(condition string, configGroup string) error {
	if condition == "" {
		err := fmt.Errorf(`"servers.%s.loadBalancer.loadBalancingRules.condition" is nil or empty`, configGroup)
		return err
	}
	return nil
}

// validateDistribution checks if the distribution list is valid for LoadBalancingRules.
func validateDistribution(
	distributionList []Distribution,
	configGroup string,
	condition string,
	clientConfigGroups map[string]map[string]bool,
) error {
	// Check if the distribution list is empty
	if len(distributionList) == 0 {
		return fmt.Errorf(
			`"servers.%s.loadBalancer.loadBalancingRules.distribution" is empty`,
			configGroup,
		)
	}

	var totalWeight int
	for _, distribution := range distributionList {
		// Validate each distribution entry
		if err := validateDistributionEntry(distribution, configGroup, condition, clientConfigGroups); err != nil {
			return err
		}

		// Check if adding the weight would exceed the maximum integer value
		if totalWeight > math.MaxInt-distribution.Weight {
			return fmt.Errorf(
				`"servers.%s.loadBalancer.loadBalancingRules.%s" total weight exceeds maximum int value`,
				configGroup,
				condition,
			)
		}

		totalWeight += distribution.Weight
	}

	return nil
}

// validateDistributionEntry validates a single distribution entry for LoadBalancingRules.
func validateDistributionEntry(
	distribution Distribution,
	configGroup string,
	condition string,
	clientConfigGroups map[string]map[string]bool,
) error {
	// Check if the distribution.ProxyName is referenced in the proxy configuration
	if !clientConfigGroups[configGroup][distribution.ProxyName] {
		return fmt.Errorf(
			`"servers.%s.loadBalancer.loadBalancingRules.%s.%s" not referenced in proxy configuration`,
			configGroup,
			condition,
			distribution.ProxyName,
		)
	}

	// Ensure that the distribution weight is positive
	if distribution.Weight <= 0 {
		return fmt.Errorf(
			`"servers.%s.loadBalancer.loadBalancingRules.%s.%s.weight" must be positive`,
			configGroup,
			condition,
			distribution.ProxyName,
		)
	}

	return nil
}
