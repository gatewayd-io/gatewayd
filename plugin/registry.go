package plugin

import (
	"context"

	semver "github.com/Masterminds/semver/v3"
	gerr "github.com/gatewayd-io/gatewayd/errors"
	"github.com/gatewayd-io/gatewayd/logging"
	pluginV1 "github.com/gatewayd-io/gatewayd/plugin/v1"
	"github.com/gatewayd-io/gatewayd/pool"
	goplugin "github.com/hashicorp/go-plugin"
	"github.com/knadh/koanf"
	"github.com/mitchellh/mapstructure"
	"google.golang.org/protobuf/types/known/structpb"
)

type PluginCompatPolicy uint

const (
	DefaultMinPort      uint   = 50000
	DefaultMaxPort      uint   = 60000
	PluginPriorityStart uint   = 1000
	EmptyPoolCapacity   int    = 0
	LoggerName          string = "plugin"
)

const (
	Strict PluginCompatPolicy = iota
	Loose
)

type Registry interface {
	Add(plugin *Impl) bool
	Get(id Identifier) *Impl
	List() []Identifier
	Exists(name, version, remoteUrl string) bool
	Remove(id Identifier)
	Shutdown()
	LoadPlugins(pluginConfig *koanf.Koanf)
	RegisterHooks(id Identifier)
}

type RegistryImpl struct {
	plugins      pool.Pool
	hooksConfig  *HookConfig
	CompatPolicy PluginCompatPolicy
}

var _ Registry = &RegistryImpl{}

// NewRegistry creates a new plugin registry.
func NewRegistry(hooksConfig *HookConfig) *RegistryImpl {
	return &RegistryImpl{plugins: pool.NewPool(EmptyPoolCapacity), hooksConfig: hooksConfig}
}

// Add adds a plugin to the registry.
func (reg *RegistryImpl) Add(plugin *Impl) bool {
	_, loaded, err := reg.plugins.GetOrPut(plugin.ID, plugin)
	if err != nil {
		reg.hooksConfig.Logger.Error().Err(err).Msg("Failed to add plugin to registry")
		return false
	}
	return loaded
}

// Get returns a plugin from the registry.
func (reg *RegistryImpl) Get(id Identifier) *Impl {
	if plugin, ok := reg.plugins.Get(id).(*Impl); ok {
		return plugin
	}

	return nil
}

// List returns a list of all plugins in the registry.
func (reg *RegistryImpl) List() []Identifier {
	var plugins []Identifier
	reg.plugins.ForEach(func(key, _ interface{}) bool {
		if id, ok := key.(Identifier); ok {
			plugins = append(plugins, id)
		}
		return true
	})
	return plugins
}

// Exists checks if a plugin exists in the registry.
func (reg *RegistryImpl) Exists(name, version, remoteUrl string) bool {
	for _, plugin := range reg.List() {
		if plugin.Name == name && plugin.RemoteURL == remoteUrl {
			// Parse the supplied version and the version in the registry.
			suppliedVer, err := semver.NewVersion(version)
			if err != nil {
				reg.hooksConfig.Logger.Error().Err(err).Msg(
					"Failed to parse supplied plugin version")
				return false
			}

			registryVer, err := semver.NewVersion(plugin.Version)
			if err != nil {
				reg.hooksConfig.Logger.Error().Err(err).Msg(
					"Failed to parse plugin version in registry")
				return false
			}

			// Check if the version of the plugin is less than or equal to
			// the version in the registry.
			if suppliedVer.LessThan(registryVer) || suppliedVer.Equal(registryVer) {
				return true
			} else {
				reg.hooksConfig.Logger.Debug().Str("name", name).Str("version", version).Msg(
					"Supplied plugin version is greater than the version in registry")
				return false
			}
		}
	}

	return false
}

// Remove removes a plugin from the registry.
func (reg *RegistryImpl) Remove(id Identifier) {
	reg.plugins.Remove(id)
}

// Shutdown shuts down all plugins in the registry.
func (reg *RegistryImpl) Shutdown() {
	reg.plugins.ForEach(func(key, value interface{}) bool {
		if id, ok := key.(Identifier); ok {
			if plugin, ok := value.(*Impl); ok {
				plugin.Stop()
				reg.Remove(id)
			}
		}
		return true
	})
	goplugin.CleanupClients()
}

// LoadPlugins loads plugins from the config file.
//
//nolint:funlen
func (reg *RegistryImpl) LoadPlugins(pluginConfig *koanf.Koanf) {
	// Get top-level list of plugins.
	plugins := pluginConfig.MapKeys("")

	// TODO: Append built-in plugins to the list of plugins
	// Built-in plugins are plugins that are compiled and shipped with the gatewayd binary.

	// Add each plugin to the registry.
	for priority, name := range plugins {
		// Skip the top-level "plugins" key.
		if name == "plugins" {
			continue
		}

		reg.hooksConfig.Logger.Debug().Str("name", name).Msg("Loading plugin")
		plugin := &Impl{
			ID: Identifier{
				Name: name,
			},
		}

		// Is the plugin enabled?
		plugin.Enabled = pluginConfig.Bool(name + ".enabled")
		if !plugin.Enabled {
			reg.hooksConfig.Logger.Debug().Str("name", name).Msg("Plugin is disabled")
			continue
		}

		// File path of the plugin on disk.
		plugin.LocalPath = pluginConfig.String(name + ".localPath")
		if plugin.LocalPath == "" {
			reg.hooksConfig.Logger.Debug().Str("name", name).Msg(
				"Local file of the plugin doesn't exist or is not set")
			continue
		}

		// Checksum of the plugin.
		plugin.ID.Checksum = pluginConfig.String(name + ".checksum")
		if plugin.ID.Checksum == "" {
			reg.hooksConfig.Logger.Debug().Str("name", name).Msg(
				"Checksum of plugin doesn't exist or is not set")
			continue
		}

		// Verify the checksum.
		// TODO: Load the plugin from a remote location if the checksum didn't match?
		if sum, err := sha256sum(plugin.LocalPath); err != nil {
			reg.hooksConfig.Logger.Debug().Err(err).Msg("Failed to calculate checksum")
			continue
		} else if sum != plugin.ID.Checksum {
			reg.hooksConfig.Logger.Debug().Fields(
				map[string]interface{}{
					"calculated": sum,
					"expected":   plugin.ID.Checksum,
				},
			).Msg("Checksum mismatch")
			continue
		}

		// Commandline arguments to pass to the plugin.
		if args := pluginConfig.Strings(name + ".args"); len(args) > 0 {
			plugin.Args = args
		}

		// Custom environment variables to pass to the plugin.
		if env := pluginConfig.Strings(name + ".env"); len(env) > 0 {
			plugin.Env = append(plugin.Env, env...)
		}

		// Plugin priority is determined by the order in which the plugin is listed
		// in the config file. Built-in plugins are loaded first, followed by user-defined
		// plugins. Built-in plugins have a priority of 0 to 999, and user-defined plugins
		// have a priority of 1000 or greater.
		plugin.Priority = Priority(PluginPriorityStart + uint(priority))

		logAdapter := logging.NewHcLogAdapter(&reg.hooksConfig.Logger, LoggerName)

		plugin.client = goplugin.NewClient(
			&goplugin.ClientConfig{
				HandshakeConfig: pluginV1.Handshake,
				Plugins:         pluginV1.GetPluginMap(plugin.ID.Name),
				Cmd:             NewCommand(plugin.LocalPath, plugin.Args, plugin.Env),
				AllowedProtocols: []goplugin.Protocol{
					goplugin.ProtocolGRPC,
				},
				// SecureConfig: nil,
				Logger:  logAdapter,
				Managed: true,
				MinPort: DefaultMinPort,
				MaxPort: DefaultMaxPort,
				// TODO: Enable GRPC DialOptions
				// GRPCDialOptions: []grpc.DialOption{
				// 	grpc.WithInsecure(),
				// },
				AutoMTLS: true,
			},
		)

		reg.hooksConfig.Logger.Debug().Str("name", plugin.ID.Name).Msg("Plugin loaded")
		if _, err := plugin.Start(); err != nil {
			reg.hooksConfig.Logger.Debug().Err(err).Msg("Failed to start plugin")
		}

		// Load metadata from the plugin.
		var metadata *structpb.Struct
		if pluginV1, err := plugin.Dispense(); err != nil {
			reg.hooksConfig.Logger.Debug().Err(err).Msg("Failed to dispense plugin")
			continue
		} else {
			if md, origErr := pluginV1.GetPluginConfig(
				context.Background(), &structpb.Struct{}); err != nil {
				reg.hooksConfig.Logger.Debug().Err(origErr).Msg("Failed to get plugin metadata")
				continue
			} else {
				metadata = md
			}
		}

		// Retrieve plugin requirements.
		if err := mapstructure.Decode(metadata.Fields["requires"].GetListValue().AsSlice(),
			&plugin.Requires); err != nil {
			reg.hooksConfig.Logger.Debug().Err(err).Msg("Failed to decode plugin requirements")
		}

		// Too many requirements or not enough plugins loaded.
		if len(plugin.Requires) > reg.plugins.Size() {
			reg.hooksConfig.Logger.Debug().Msg(
				"The plugin has too many requirements, " +
					"and not enough of them exist in the registry, so it won't work properly")
		}

		// Check if the plugin requirements are met.
		for _, req := range plugin.Requires {
			if !reg.Exists(req.Name, req.Version, req.RemoteURL) {
				reg.hooksConfig.Logger.Debug().Str("name", plugin.ID.Name).Msg(
					"The plugin requirement is not met, so it won't work properly")
				if reg.CompatPolicy == Strict {
					reg.hooksConfig.Logger.Debug().Str("name", plugin.ID.Name).Msg(
						"Registry is in strict compatibility mode, so the plugin won't be loaded")
					plugin.Stop() // Stop the plugin.
					continue
				} else {
					reg.hooksConfig.Logger.Debug().Str("name", plugin.ID.Name).Msg(
						"Registry is in loose compatibility mode, " +
							"so the plugin will be loaded anyway")
				}
			}
		}

		plugin.ID.RemoteURL = metadata.Fields["id"].GetStructValue().Fields["remoteUrl"].GetStringValue()
		plugin.ID.Version = metadata.Fields["id"].GetStructValue().Fields["version"].GetStringValue()
		plugin.Description = metadata.Fields["description"].GetStringValue()
		plugin.License = metadata.Fields["license"].GetStringValue()
		plugin.ProjectURL = metadata.Fields["projectUrl"].GetStringValue()
		// Retrieve authors.
		if err := mapstructure.Decode(metadata.Fields["authors"].GetListValue().AsSlice(),
			&plugin.Authors); err != nil {
			reg.hooksConfig.Logger.Debug().Err(err).Msg("Failed to decode plugin authors")
		}
		// Retrieve hooks.
		if err := mapstructure.Decode(metadata.Fields["hooks"].GetListValue().AsSlice(),
			&plugin.Hooks); err != nil {
			reg.hooksConfig.Logger.Debug().Err(err).Msg("Failed to decode plugin hooks")
		}

		// Retrieve plugin config.
		plugin.Config = make(map[string]string)
		for key, value := range metadata.Fields["config"].GetStructValue().AsMap() {
			if val, ok := value.(string); ok {
				plugin.Config[key] = val
			} else {
				reg.hooksConfig.Logger.Debug().Str("key", key).Msg(
					"Failed to decode plugin config")
			}
		}

		reg.hooksConfig.Logger.Trace().Msgf("Plugin metadata: %+v", plugin)

		reg.Add(plugin)
		reg.hooksConfig.Logger.Debug().Str("name", plugin.ID.Name).Msg("Plugin metadata loaded")

		reg.RegisterHooks(plugin.ID)
		reg.hooksConfig.Logger.Debug().Str("name", plugin.ID.Name).Msg("Plugin hooks registered")
	}
}

// RegisterHooks registers the hooks for the given plugin.
//
//nolint:funlen
func (reg *RegistryImpl) RegisterHooks(id Identifier) {
	pluginImpl := reg.Get(id)
	reg.hooksConfig.Logger.Debug().Str("name", pluginImpl.ID.Name).Msg(
		"Registering hooks for plugin")
	var pluginV1 pluginV1.GatewayDPluginServiceClient
	var err *gerr.GatewayDError
	if pluginV1, err = pluginImpl.Dispense(); err != nil {
		reg.hooksConfig.Logger.Debug().Err(err).Msg("Failed to dispense plugin")
		return
	}

	for _, hook := range pluginImpl.Hooks {
		var hookFunc HookDef
		switch hook {
		case OnConfigLoaded:
			hookFunc = pluginV1.OnConfigLoaded
		case OnNewLogger:
			hookFunc = pluginV1.OnNewLogger
		case OnNewPool:
			hookFunc = pluginV1.OnNewPool
		case OnNewProxy:
			hookFunc = pluginV1.OnNewProxy
		case OnNewServer:
			hookFunc = pluginV1.OnNewServer
		case OnSignal:
			hookFunc = pluginV1.OnSignal
		case OnRun:
			hookFunc = pluginV1.OnRun
		case OnBooting:
			hookFunc = pluginV1.OnBooting
		case OnBooted:
			hookFunc = pluginV1.OnBooted
		case OnOpening:
			hookFunc = pluginV1.OnOpening
		case OnOpened:
			hookFunc = pluginV1.OnOpened
		case OnClosing:
			hookFunc = pluginV1.OnClosing
		case OnClosed:
			hookFunc = pluginV1.OnClosed
		case OnTraffic:
			hookFunc = pluginV1.OnTraffic
		case OnIngressTraffic:
			hookFunc = pluginV1.OnIngressTraffic
		case OnEgressTraffic:
			hookFunc = pluginV1.OnEgressTraffic
		case OnShutdown:
			hookFunc = pluginV1.OnShutdown
		case OnTick:
			hookFunc = pluginV1.OnTick
		case OnNewClient:
			hookFunc = pluginV1.OnNewClient
		default:
			reg.hooksConfig.Logger.Warn().Str("hook", string(hook)).Msg("Unknown hook type")
			continue
		}
		reg.hooksConfig.Logger.Debug().Str("hook", string(hook)).Msg("Registering hook")
		reg.hooksConfig.Add(hook, pluginImpl.Priority, hookFunc)
	}
}
