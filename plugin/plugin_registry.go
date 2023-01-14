package plugin

import (
	"context"
	"sort"

	semver "github.com/Masterminds/semver/v3"
	"github.com/gatewayd-io/gatewayd/config"
	gerr "github.com/gatewayd-io/gatewayd/errors"
	"github.com/gatewayd-io/gatewayd/logging"
	pluginV1 "github.com/gatewayd-io/gatewayd/plugin/v1"
	"github.com/gatewayd-io/gatewayd/pool"
	goplugin "github.com/hashicorp/go-plugin"
	"github.com/mitchellh/mapstructure"
	"github.com/rs/zerolog"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/structpb"
)

//nolint:interfacebloat
type IRegistry interface {
	// Plugin management
	Add(plugin *Plugin) bool
	Get(id Identifier) *Plugin
	List() []Identifier
	Exists(name, version, remoteURL string) bool
	Remove(id Identifier)
	Shutdown()
	LoadPlugins(plugins []config.Plugin)
	RegisterHooks(id Identifier)

	// Hook management
	AddHook(hookName string, priority Priority, hookMethod Method)
	Hooks() map[string]map[Priority]Method
	Run(
		ctx context.Context,
		args map[string]interface{},
		hookName string,
		verification config.VerificationPolicy,
		opts ...grpc.CallOption,
	) (map[string]interface{}, *gerr.GatewayDError)
}

type Registry struct {
	plugins pool.IPool
	hooks   map[string]map[Priority]Method

	Logger        zerolog.Logger
	Compatibility config.CompatibilityPolicy
	Verification  config.VerificationPolicy
	Acceptance    config.AcceptancePolicy
}

var _ IRegistry = &Registry{}

// NewRegistry creates a new plugin registry.
func NewRegistry(
	compatibility config.CompatibilityPolicy,
	verification config.VerificationPolicy,
	acceptance config.AcceptancePolicy,
	logger zerolog.Logger,
) *Registry {
	return &Registry{
		plugins:       pool.NewPool(config.EmptyPoolCapacity),
		hooks:         map[string]map[Priority]Method{},
		Logger:        logger,
		Compatibility: compatibility,
		Verification:  verification,
		Acceptance:    acceptance,
	}
}

// Add adds a plugin to the registry.
func (reg *Registry) Add(plugin *Plugin) bool {
	_, loaded, err := reg.plugins.GetOrPut(plugin.ID, plugin)
	if err != nil {
		reg.Logger.Error().Err(err).Msg("Failed to add plugin to registry")
		return false
	}
	return loaded
}

// Get returns a plugin from the registry.
func (reg *Registry) Get(id Identifier) *Plugin {
	if plugin, ok := reg.plugins.Get(id).(*Plugin); ok {
		return plugin
	}

	return nil
}

// List returns a list of all plugins in the registry.
func (reg *Registry) List() []Identifier {
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
func (reg *Registry) Exists(name, version, remoteURL string) bool {
	for _, plugin := range reg.List() {
		if plugin.Name == name && plugin.RemoteURL == remoteURL {
			// Parse the supplied version and the version in the registry.
			suppliedVer, err := semver.NewVersion(version)
			if err != nil {
				reg.Logger.Error().Err(err).Msg(
					"Failed to parse supplied plugin version")
				return false
			}

			registryVer, err := semver.NewVersion(plugin.Version)
			if err != nil {
				reg.Logger.Error().Err(err).Msg(
					"Failed to parse plugin version in registry")
				return false
			}

			// Check if the version of the plugin is less than or equal to
			// the version in the registry.
			if suppliedVer.LessThan(registryVer) || suppliedVer.Equal(registryVer) {
				return true
			}

			reg.Logger.Debug().Str("name", name).Str("version", version).Msg(
				"Supplied plugin version is greater than the version in registry")
			return false
		}
	}

	return false
}

// Remove removes a plugin from the registry.
func (reg *Registry) Remove(id Identifier) {
	reg.plugins.Remove(id)
}

// Shutdown shuts down all plugins in the registry.
func (reg *Registry) Shutdown() {
	reg.plugins.ForEach(func(key, value interface{}) bool {
		if id, ok := key.(Identifier); ok {
			if plugin, ok := value.(*Plugin); ok {
				plugin.Stop()
				reg.Remove(id)
			}
		}
		return true
	})
	goplugin.CleanupClients()
}

// Hooks returns the hooks map.
func (reg *Registry) Hooks() map[string]map[Priority]Method {
	return reg.hooks
}

// Add adds a hook with a priority to the hooks map.
func (reg *Registry) AddHook(hookName string, priority Priority, hookMethod Method) {
	if len(reg.hooks[hookName]) == 0 {
		reg.hooks[hookName] = map[Priority]Method{priority: hookMethod}
	} else {
		if _, ok := reg.hooks[hookName][priority]; ok {
			reg.Logger.Warn().Fields(
				map[string]interface{}{
					"hookName": hookName,
					"priority": priority,
				},
			).Msg("Hook is replaced")
		}
		reg.hooks[hookName][priority] = hookMethod
	}
}

// Run runs the hooks of a specific type. The result of the previous hook is passed
// to the next hook as the argument, aka. chained. The context is passed to the
// hooks as well to allow them to cancel the execution. The args are passed to the
// first hook as the argument. The result of the first hook is passed to the second
// hook, and so on. The result of the last hook is eventually returned. The verification
// mode is used to determine how to handle errors. If the verification mode is set to
// Abort, the execution is aborted on the first error. If the verification mode is set
// to Remove, the hook is removed from the list of hooks on the first error. If the
// verification mode is set to Ignore, the error is ignored and the execution continues.
// If the verification mode is set to PassDown, the extra keys/values in the result
// are passed down to the next  The verification mode is set to PassDown by default.
// The opts are passed to the hooks as well to allow them to use the grpc.CallOption.
//
//nolint:funlen
func (reg *Registry) Run(
	ctx context.Context,
	args map[string]interface{},
	hookName string,
	verification config.VerificationPolicy,
	opts ...grpc.CallOption,
) (map[string]interface{}, *gerr.GatewayDError) {
	if ctx == nil {
		return nil, gerr.ErrNilContext
	}

	// Inherit context.
	inheritedCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	// Cast custom fields to their primitive types, like time.Duration to float64.
	args = CastToPrimitiveTypes(args)

	// Create structpb.Struct from args.
	var params *structpb.Struct
	if len(args) == 0 {
		params = &structpb.Struct{}
	} else if casted, err := structpb.NewStruct(args); err == nil {
		params = casted
	} else {
		return nil, gerr.ErrCastFailed.Wrap(err)
	}

	// Sort hooks by priority.
	priorities := make([]Priority, 0, len(reg.hooks[hookName]))
	for priority := range reg.hooks[hookName] {
		priorities = append(priorities, priority)
	}
	sort.SliceStable(priorities, func(i, j int) bool {
		return priorities[i] < priorities[j]
	})

	// Run hooks, passing the result of the previous hook to the next one.
	returnVal := &structpb.Struct{}
	var removeList []Priority
	// The signature of parameters and args MUST be the same for this to work.
	for idx, priority := range priorities {
		var result *structpb.Struct
		var err error
		if idx == 0 {
			result, err = reg.hooks[hookName][priority](inheritedCtx, params, opts...)
		} else {
			result, err = reg.hooks[hookName][priority](inheritedCtx, returnVal, opts...)
		}

		// This is done to ensure that the return value of the hook is always valid,
		// and that the hook does not return any unexpected values.
		// If the verification mode is non-strict (permissive), let the plugin pass
		// extra keys/values to the next plugin in chain.
		if Verify(params, result) || verification == config.PassDown {
			// Update the last return value with the current result
			returnVal = result
			continue
		}

		// At this point, the hook returned an invalid value, so we need to handle it.
		// The result of the current hook will be ignored, regardless of the policy.
		switch verification {
		// Ignore the result of this plugin, log an error and execute the next
		case config.Ignore:
			reg.Logger.Error().Err(err).Fields(
				map[string]interface{}{
					"hookName": hookName,
					"priority": priority,
				},
			).Msg("Hook returned invalid value, ignoring")
			if idx == 0 {
				returnVal = params
			}
		// Abort execution of the plugins, log the error and return the result of the last
		case config.Abort:
			reg.Logger.Error().Err(err).Fields(
				map[string]interface{}{
					"hookName": hookName,
					"priority": priority,
				},
			).Msg("Hook returned invalid value, aborting")
			if idx == 0 {
				return args, nil
			}
			return returnVal.AsMap(), nil
		// Remove the hook from the registry, log the error and execute the next
		case config.Remove:
			reg.Logger.Error().Err(err).Fields(
				map[string]interface{}{
					"hookName": hookName,
					"priority": priority,
				},
			).Msg("Hook returned invalid value, removing")
			removeList = append(removeList, priority)
			if idx == 0 {
				returnVal = params
			}
		case config.PassDown:
		default:
			returnVal = result
		}
	}

	// Remove hooks that failed verification.
	for _, priority := range removeList {
		delete(reg.hooks[hookName], priority)
	}

	return returnVal.AsMap(), nil
}

// LoadPlugins loads plugins from the config file.
//
//nolint:funlen
func (reg *Registry) LoadPlugins(plugins []config.Plugin) {
	// TODO: Append built-in plugins to the list of plugins
	// Built-in plugins are plugins that are compiled and shipped with the gatewayd binary.

	// Add each plugin to the registry.
	for priority, pCfg := range plugins {
		// Skip the top-level "plugins" key.
		if pCfg.Name == "plugins" {
			continue
		}

		reg.Logger.Debug().Str("name", pCfg.Name).Msg("Loading plugin")
		plugin := &Plugin{
			ID: Identifier{
				Name:     pCfg.Name,
				Checksum: pCfg.Checksum,
			},
			Enabled:   pCfg.Enabled,
			LocalPath: pCfg.LocalPath,
			Args:      pCfg.Args,
			Env:       pCfg.Env,
		}

		// Is the plugin enabled?
		plugin.Enabled = pCfg.Enabled
		if !plugin.Enabled {
			reg.Logger.Debug().Str("name", plugin.ID.Name).Msg("Plugin is disabled")
			continue
		}

		// File path of the plugin on disk.
		if plugin.LocalPath == "" {
			reg.Logger.Debug().Str("name", plugin.ID.Name).Msg(
				"Local file of the plugin doesn't exist or is not set")
			continue
		}

		// Checksum of the plugin.
		if plugin.ID.Checksum == "" {
			reg.Logger.Debug().Str("name", plugin.ID.Name).Msg(
				"Checksum of plugin doesn't exist or is not set")
			continue
		}

		// Verify the checksum.
		// TODO: Load the plugin from a remote location if the checksum didn't match?
		if sum, err := SHA256SUM(plugin.LocalPath); err != nil {
			reg.Logger.Debug().Str("name", plugin.ID.Name).Err(err).Msg(
				"Failed to calculate checksum")
			continue
		} else if sum != plugin.ID.Checksum {
			reg.Logger.Debug().Fields(
				map[string]interface{}{
					"calculated": sum,
					"expected":   plugin.ID.Checksum,
					"name":       plugin.ID.Name,
				},
			).Msg("Checksum mismatch")
			continue
		}

		// Plugin priority is determined by the order in which the plugin is listed
		// in the config file. Built-in plugins are loaded first, followed by user-defined
		// plugins. Built-in plugins have a priority of 0 to 999, and user-defined plugins
		// have a priority of 1000 or greater.
		plugin.Priority = Priority(config.PluginPriorityStart + uint(priority))

		logAdapter := logging.NewHcLogAdapter(&reg.Logger, config.LoggerName)

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
				MinPort: config.DefaultMinPort,
				MaxPort: config.DefaultMaxPort,
				// TODO: Enable GRPC DialOptions
				// GRPCDialOptions: []grpc.DialOption{
				// 	grpc.WithInsecure(),
				// },
				AutoMTLS: true,
			},
		)

		reg.Logger.Debug().Str("name", plugin.ID.Name).Msg("Plugin loaded")
		if _, err := plugin.Start(); err != nil {
			reg.Logger.Debug().Str("name", plugin.ID.Name).Err(err).Msg(
				"Failed to start plugin")
		}

		// Load metadata from the plugin.
		var metadata *structpb.Struct
		if pluginV1, err := plugin.Dispense(); err != nil {
			reg.Logger.Debug().Str("name", plugin.ID.Name).Err(err).Msg(
				"Failed to dispense plugin")
			continue
		} else {
			if meta, origErr := pluginV1.GetPluginConfig(
				context.Background(), &structpb.Struct{}); err != nil {
				reg.Logger.Debug().Str("name", plugin.ID.Name).Err(origErr).Msg(
					"Failed to get plugin metadata")
				continue
			} else {
				metadata = meta
			}
		}

		// Retrieve plugin requirements.
		if err := mapstructure.Decode(metadata.Fields["requires"].GetListValue().AsSlice(),
			&plugin.Requires); err != nil {
			reg.Logger.Debug().Err(err).Msg("Failed to decode plugin requirements")
		}

		// Too many requirements or not enough plugins loaded.
		if len(plugin.Requires) > reg.plugins.Size() {
			reg.Logger.Debug().Msg(
				"The plugin has too many requirements, " +
					"and not enough of them exist in the registry, so it won't work properly")
		}

		// Check if the plugin requirements are met.
		for _, req := range plugin.Requires {
			if !reg.Exists(req.Name, req.Version, req.RemoteURL) {
				reg.Logger.Debug().Fields(
					map[string]interface{}{
						"name":        plugin.ID.Name,
						"requirement": req.Name,
					},
				).Msg("The plugin requirement is not met, so it won't work properly")
				if reg.Compatibility == config.Strict {
					reg.Logger.Debug().Str("name", plugin.ID.Name).Msg(
						"Registry is in strict compatibility mode, so the plugin won't be loaded")
					plugin.Stop() // Stop the plugin.
					continue
				} else {
					reg.Logger.Debug().Fields(
						map[string]interface{}{
							"name":        plugin.ID.Name,
							"requirement": req.Name,
						},
					).Msg("Registry is in loose compatibility mode, " +
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
			reg.Logger.Debug().Err(err).Msg("Failed to decode plugin authors")
		}
		// Retrieve hooks.
		if err := mapstructure.Decode(metadata.Fields["hooks"].GetListValue().AsSlice(),
			&plugin.Hooks); err != nil {
			reg.Logger.Debug().Err(err).Msg("Failed to decode plugin hooks")
		}

		// Retrieve plugin config.
		plugin.Config = make(map[string]string)
		for key, value := range metadata.Fields["config"].GetStructValue().AsMap() {
			if val, ok := value.(string); ok {
				plugin.Config[key] = val
			} else {
				reg.Logger.Debug().Str("key", key).Msg(
					"Failed to decode plugin config")
			}
		}

		reg.Logger.Trace().Msgf("Plugin metadata: %+v", plugin)

		reg.Add(plugin)
		reg.Logger.Debug().Str("name", plugin.ID.Name).Msg("Plugin metadata loaded")

		reg.RegisterHooks(plugin.ID)
		reg.Logger.Debug().Str("name", plugin.ID.Name).Msg("Plugin hooks registered")
	}
}

// RegisterHooks registers the hooks for the given plugin.
//
//nolint:funlen
func (reg *Registry) RegisterHooks(id Identifier) {
	pluginImpl := reg.Get(id)
	reg.Logger.Debug().Str("name", pluginImpl.ID.Name).Msg(
		"Registering hooks for plugin")
	var pluginV1 pluginV1.GatewayDPluginServiceClient
	var err *gerr.GatewayDError
	if pluginV1, err = pluginImpl.Dispense(); err != nil {
		reg.Logger.Debug().Str("name", pluginImpl.ID.Name).Err(err).Msg(
			"Failed to dispense plugin")
		return
	}

	for _, hookName := range pluginImpl.Hooks {
		var hookMethod Method
		switch hookName {
		case OnConfigLoaded:
			hookMethod = pluginV1.OnConfigLoaded
		case OnNewLogger:
			hookMethod = pluginV1.OnNewLogger
		case OnNewPool:
			hookMethod = pluginV1.OnNewPool
		case OnNewProxy:
			hookMethod = pluginV1.OnNewProxy
		case OnNewServer:
			hookMethod = pluginV1.OnNewServer
		case OnSignal:
			hookMethod = pluginV1.OnSignal
		case OnRun:
			hookMethod = pluginV1.OnRun
		case OnBooting:
			hookMethod = pluginV1.OnBooting
		case OnBooted:
			hookMethod = pluginV1.OnBooted
		case OnOpening:
			hookMethod = pluginV1.OnOpening
		case OnOpened:
			hookMethod = pluginV1.OnOpened
		case OnClosing:
			hookMethod = pluginV1.OnClosing
		case OnClosed:
			hookMethod = pluginV1.OnClosed
		case OnTraffic:
			hookMethod = pluginV1.OnTraffic
		case OnTrafficFromClient:
			hookMethod = pluginV1.OnTrafficFromClient
		case OnTrafficToServer:
			hookMethod = pluginV1.OnTrafficToServer
		case OnTrafficFromServer:
			hookMethod = pluginV1.OnTrafficFromServer
		case OnTrafficToClient:
			hookMethod = pluginV1.OnTrafficToClient
		case OnShutdown:
			hookMethod = pluginV1.OnShutdown
		case OnTick:
			hookMethod = pluginV1.OnTick
		case OnNewClient:
			hookMethod = pluginV1.OnNewClient
		default:
			switch reg.Acceptance {
			case config.Reject:
				reg.Logger.Warn().Fields(map[string]interface{}{
					"hook":     hookName,
					"priority": pluginImpl.Priority,
					"name":     pluginImpl.ID.Name,
				}).Msg("Unknown hook, skipping")
			case config.Accept: // fallthrough
			default:
				// Default is to accept custom hooks.
				reg.Logger.Warn().Fields(map[string]interface{}{
					"hook":     hookName,
					"priority": pluginImpl.Priority,
					"name":     pluginImpl.ID.Name,
				}).Msg("Registering a custom hook")
				reg.AddHook(hookName, pluginImpl.Priority, pluginV1.OnHook)
			}
			continue
		}

		reg.Logger.Debug().Fields(map[string]interface{}{
			"hook":     hookName,
			"priority": pluginImpl.Priority,
			"name":     pluginImpl.ID.Name,
		}).Msg("Registering hook")
		reg.AddHook(hookName, pluginImpl.Priority, hookMethod)
	}
}
