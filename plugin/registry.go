package plugin

import (
	"context"
	"os/exec"

	// testplugin "github.com/gatewayd-io/gatewayd-plugin-test/plugin"

	plugin_v1 "github.com/gatewayd-io/gatewayd/plugin/v1"
	"github.com/gatewayd-io/gatewayd/pool"
	goplugin "github.com/hashicorp/go-plugin"
	"github.com/knadh/koanf"
	"github.com/mitchellh/mapstructure"
	"google.golang.org/protobuf/types/known/structpb"
)

const (
	DefaultMinPort      uint = 50000
	DefaultMaxPort      uint = 60000
	PluginPriorityStart uint = 1000
)

var handshakeConfig = goplugin.HandshakeConfig{
	ProtocolVersion:  1,
	MagicCookieKey:   "GATEWAYD_PLUGIN",
	MagicCookieValue: "5712b87aa5d7e9f9e9ab643e6603181c5b796015cb1c09d6f5ada882bf2a1872",
}

type Registry interface {
	Add(plugin *PluginImpl) bool
	Get(id Identifier) *PluginImpl
	List() []Identifier
	Remove(id Identifier)
	Shutdown()
	LoadPlugins(pluginConfig *koanf.Koanf)
	RegisterHooks(id Identifier)
}

type RegistryImpl struct {
	plugins     pool.Pool
	hooksConfig *HookConfig
}

var _ Registry = &RegistryImpl{}

func NewRegistry(hooksConfig *HookConfig) *RegistryImpl {
	return &RegistryImpl{plugins: pool.NewPool(), hooksConfig: hooksConfig}
}

func (reg *RegistryImpl) Add(plugin *PluginImpl) bool {
	_, loaded := reg.plugins.GetOrPut(plugin.ID, plugin)
	return loaded
}

func (reg *RegistryImpl) Get(id Identifier) *PluginImpl {
	if plugin, ok := reg.plugins.Get(id).(*PluginImpl); ok {
		return plugin
	}

	return nil
}

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

func (reg *RegistryImpl) Remove(id Identifier) {
	reg.plugins.Remove(id)
}

func (reg *RegistryImpl) Shutdown() {
	reg.plugins.ForEach(func(key, value interface{}) bool {
		if id, ok := key.(Identifier); ok {
			if plugin, ok := value.(*PluginImpl); ok {
				plugin.Stop()
				reg.Remove(id)
			}
		}
		return true
	})
}

func (reg *RegistryImpl) LoadPlugins(pluginConfig *koanf.Koanf) {
	// Get top-level list of plugins
	plugins := pluginConfig.MapKeys("")

	// TODO: Append built-in plugins to the list of plugins
	// Built-in plugins are plugins that are compiled and shipped with the gatewayd binary

	// Add each plugin to the registry
	for priority, name := range plugins {
		reg.hooksConfig.Logger.Debug().Msgf("Loading plugin: %s", name)
		plugin := &PluginImpl{
			ID: Identifier{
				Name: name,
			},
		}

		if enabled, ok := pluginConfig.Get(name + ".enabled").(bool); !ok || !enabled {
			reg.hooksConfig.Logger.Debug().Msgf("Plugin is disabled or is not set: %s", name)
			continue
		} else {
			plugin.Enabled = enabled
		}

		if localPath, ok := pluginConfig.Get(
			name + ".localPath").(string); !ok || localPath == "" {
			reg.hooksConfig.Logger.Debug().Msgf("Local file of plugin doesn't exist or is not set: %s", name)
			continue
		} else {
			plugin.LocalPath = localPath
		}

		if checksum, ok := pluginConfig.Get(name + ".checksum").(string); !ok || checksum == "" {
			reg.hooksConfig.Logger.Debug().Msgf("Checksum of plugin doesn't exist or is not set: %s", name)
			continue
		} else {
			plugin.ID.Checksum = checksum
		}

		// Verify the checksum
		// TODO: Load the plugin from a remote location if the checksum doesn't match
		if sum, err := sha256sum(plugin.LocalPath); err != nil {
			reg.hooksConfig.Logger.Debug().Err(err).Msg("Failed to calculate checksum")
			continue
		} else {
			if sum != plugin.ID.Checksum {
				reg.hooksConfig.Logger.Debug().Msgf(
					"Checksum mismatch: %s != %s", sum, plugin.ID.Checksum)
				continue
			}
		}

		// Plugin priority is determined by the order in which it is listed in the config file
		// Built-in plugins are loaded first, followed by user-defined plugins. Built-in plugins
		// have a priority of 0 to 999, and user-defined plugins have a priority of 1000 or greater.
		plugin.Priority = Priority(PluginPriorityStart + uint(priority))

		plugin.client = goplugin.NewClient(
			&goplugin.ClientConfig{
				HandshakeConfig: handshakeConfig,
				Plugins:         plugin_v1.GetPluginMap(plugin.ID.Name),
				Cmd:             exec.Command(plugin.LocalPath),
				AllowedProtocols: []goplugin.Protocol{
					goplugin.ProtocolGRPC,
				},
				// SecureConfig: nil,
				Managed: true,
				MinPort: DefaultMinPort,
				MaxPort: DefaultMaxPort,
				// GRPCDialOptions: []grpc.DialOption{
				// 	grpc.WithInsecure(),
				// },
				AutoMTLS: true,
			},
		)

		reg.hooksConfig.Logger.Debug().Msgf("Plugin loaded: %s", plugin.ID.Name)
		if _, err := plugin.Start(); err != nil {
			reg.hooksConfig.Logger.Debug().Err(err).Msg("Failed to start plugin")
		}

		// Load metadata from the plugin
		var metadata *structpb.Struct
		if plugin_v1, err := plugin.Dispense(); err != nil {
			reg.hooksConfig.Logger.Debug().Err(err).Msg("Failed to dispense plugin")
			continue
		} else {
			if metadata, err = plugin_v1.GetPluginConfig(context.Background(), &structpb.Struct{}); err != nil {
				reg.hooksConfig.Logger.Debug().Err(err).Msg("Failed to get plugin metadata")
				continue
			}
		}

		plugin.ID.RemoteURL = metadata.Fields["id"].GetStructValue().Fields["remoteUrl"].GetStringValue()
		plugin.ID.Version = metadata.Fields["id"].GetStructValue().Fields["version"].GetStringValue()
		plugin.Description = metadata.Fields["description"].GetStringValue()
		plugin.License = metadata.Fields["license"].GetStringValue()
		plugin.ProjectURL = metadata.Fields["projectUrl"].GetStringValue()
		mapstructure.Decode(metadata.Fields["authors"].GetListValue().AsSlice(), &plugin.Authors)
		mapstructure.Decode(metadata.Fields["hooks"].GetListValue().AsSlice(), &plugin.Hooks)
		mapstructure.Decode(metadata.Fields["config"].GetListValue().AsSlice(), &plugin.Config)

		reg.Add(plugin)

		reg.RegisterHooks(plugin.ID)
		reg.hooksConfig.Logger.Debug().Msgf("Plugin metadata loaded: %s", plugin.ID.Name)
	}
}

func (reg *RegistryImpl) RegisterHooks(id Identifier) {
	pluginImpl := reg.Get(id)
	reg.hooksConfig.Logger.Debug().Msgf("Registering hooks for plugin: %s", pluginImpl.ID.Name)
	if plugin_v1, err := pluginImpl.Dispense(); err != nil {
		reg.hooksConfig.Logger.Debug().Err(err).Msg("Failed to dispense plugin")
		return
	} else {
		for _, hook := range pluginImpl.Hooks {
			var hookFunc HookDef
			switch hook {
			case OnConfigLoaded:
				hookFunc = plugin_v1.OnConfigLoaded
			case OnNewLogger:
				hookFunc = plugin_v1.OnNewLogger
			case OnNewPool:
				hookFunc = plugin_v1.OnNewPool
			case OnNewProxy:
				hookFunc = plugin_v1.OnNewProxy
			case OnNewServer:
				hookFunc = plugin_v1.OnNewServer
			case OnSignal:
				hookFunc = plugin_v1.OnSignal
			case OnRun:
				hookFunc = plugin_v1.OnRun
			case OnBooting:
				hookFunc = plugin_v1.OnBooting
			case OnBooted:
				hookFunc = plugin_v1.OnBooted
			case OnOpening:
				hookFunc = plugin_v1.OnOpening
			case OnOpened:
				hookFunc = plugin_v1.OnOpened
			case OnClosing:
				hookFunc = plugin_v1.OnClosing
			case OnClosed:
				hookFunc = plugin_v1.OnClosed
			case OnTraffic:
				hookFunc = plugin_v1.OnTraffic
			case OnIngressTraffic:
				hookFunc = plugin_v1.OnIngressTraffic
			case OnEgressTraffic:
				hookFunc = plugin_v1.OnEgressTraffic
			case OnShutdown:
				hookFunc = plugin_v1.OnShutdown
			case OnTick:
				hookFunc = plugin_v1.OnTick
			case OnNewClient:
				hookFunc = plugin_v1.OnNewClient
			default:
				reg.hooksConfig.Logger.Warn().Msgf("Unknown hook type: %s", hook)
				continue
			}
			reg.hooksConfig.Logger.Debug().Msgf("Registering hook: %s", hook)
			reg.hooksConfig.Add(HookType(hook), pluginImpl.Priority, hookFunc)
		}
	}
}
