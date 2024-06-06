package plugin

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"sort"
	"time"

	"github.com/Masterminds/semver/v3"
	sdkAct "github.com/gatewayd-io/gatewayd-plugin-sdk/act"
	sdkPlugin "github.com/gatewayd-io/gatewayd-plugin-sdk/plugin"
	v1 "github.com/gatewayd-io/gatewayd-plugin-sdk/plugin/v1"
	"github.com/gatewayd-io/gatewayd/act"
	"github.com/gatewayd-io/gatewayd/config"
	gerr "github.com/gatewayd-io/gatewayd/errors"
	"github.com/gatewayd-io/gatewayd/logging"
	"github.com/gatewayd-io/gatewayd/metrics"
	"github.com/gatewayd-io/gatewayd/pool"
	goplugin "github.com/hashicorp/go-plugin"
	"github.com/mitchellh/mapstructure"
	"github.com/rs/zerolog"
	"github.com/spf13/cast"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"google.golang.org/grpc"
)

type IHook interface {
	AddHook(hookName v1.HookName, priority sdkPlugin.Priority, hookMethod sdkPlugin.Method)
	Hooks() map[v1.HookName]map[sdkPlugin.Priority]sdkPlugin.Method
	Run(
		ctx context.Context,
		args map[string]any,
		hookName v1.HookName,
		opts ...grpc.CallOption,
	) (map[string]any, *gerr.GatewayDError)
}

//nolint:interfacebloat
type IRegistry interface {
	// Plugin management
	Add(plugin *Plugin) bool
	Get(pluginID sdkPlugin.Identifier) *Plugin
	List() []sdkPlugin.Identifier
	Size() int
	Exists(name, version, remoteURL string) bool
	ForEach(f func(sdkPlugin.Identifier, *Plugin))
	Remove(pluginID sdkPlugin.Identifier)
	Shutdown()
	LoadPlugins(ctx context.Context, plugins []config.Plugin, startTimeout time.Duration)
	RegisterHooks(ctx context.Context, pluginID sdkPlugin.Identifier)
	Apply(hook sdkAct.Hook) ([]*sdkAct.Output, bool)

	// Hook management
	IHook
}

type Registry struct {
	plugins     pool.IPool
	ActRegistry *act.Registry
	hooks       map[v1.HookName]map[sdkPlugin.Priority]sdkPlugin.Method
	ctx         context.Context //nolint:containedctx
	DevMode     bool

	Logger        zerolog.Logger
	Compatibility config.CompatibilityPolicy
	StartTimeout  time.Duration
}

var _ IRegistry = (*Registry)(nil)

// NewRegistry creates a new plugin registry.
func NewRegistry(
	ctx context.Context,
	registry Registry,
) *Registry {
	regCtx, span := otel.Tracer(config.TracerName).Start(ctx, "Create new registry")
	defer span.End()

	return &Registry{
		plugins:       pool.NewPool(regCtx, config.EmptyPoolCapacity),
		hooks:         map[v1.HookName]map[sdkPlugin.Priority]sdkPlugin.Method{},
		ActRegistry:   registry.ActRegistry,
		ctx:           regCtx,
		DevMode:       registry.DevMode,
		Logger:        registry.Logger,
		Compatibility: registry.Compatibility,
	}
}

// Add adds a plugin to the registry.
func (reg *Registry) Add(plugin *Plugin) bool {
	_, span := otel.Tracer(config.TracerName).Start(reg.ctx, "Add")
	defer span.End()

	_, loaded, err := reg.plugins.GetOrPut(plugin.ID, plugin)
	if err != nil {
		reg.Logger.Error().Err(err).Msg("Failed to add plugin to registry")
		span.RecordError(err)
		return false
	}
	return loaded
}

// Get returns a plugin from the registry.
func (reg *Registry) Get(pluginID sdkPlugin.Identifier) *Plugin {
	_, span := otel.Tracer(config.TracerName).Start(reg.ctx, "Get")
	defer span.End()

	if plugin, ok := reg.plugins.Get(pluginID).(*Plugin); ok {
		return plugin
	}

	return nil
}

// List returns a list of all plugins in the registry.
func (reg *Registry) List() []sdkPlugin.Identifier {
	_, span := otel.Tracer(config.TracerName).Start(reg.ctx, "List")
	defer span.End()

	var plugins []sdkPlugin.Identifier
	reg.plugins.ForEach(func(key, _ interface{}) bool {
		if id, ok := key.(sdkPlugin.Identifier); ok {
			plugins = append(plugins, id)
		}
		return true
	})
	return plugins
}

// Size returns the number of plugins in the registry.
func (reg *Registry) Size() int {
	_, span := otel.Tracer(config.TracerName).Start(reg.ctx, "Size")
	defer span.End()
	return reg.plugins.Size()
}

// Exists checks if a plugin exists in the registry.
func (reg *Registry) Exists(name, version, remoteURL string) bool {
	_, span := otel.Tracer(config.TracerName).Start(reg.ctx, "Exists")
	defer span.End()

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
			// TODO: Should we check the major version only, or as well?
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

// ForEach iterates over all plugins in the registry.
func (reg *Registry) ForEach(function func(sdkPlugin.Identifier, *Plugin)) {
	_, span := otel.Tracer(config.TracerName).Start(reg.ctx, "ForEach")
	defer span.End()

	reg.plugins.ForEach(func(key, value interface{}) bool {
		if id, ok := key.(sdkPlugin.Identifier); ok {
			if plugin, ok := value.(*Plugin); ok {
				function(id, plugin)
			}
		}
		return true
	})
}

// Remove removes plugin hooks and then removes the plugin from the registry.
func (reg *Registry) Remove(pluginID sdkPlugin.Identifier) {
	_, span := otel.Tracer(config.TracerName).Start(reg.ctx, "Remove")
	defer span.End()

	plugin := reg.Get(pluginID)
	for _, hooks := range reg.hooks {
		delete(hooks, plugin.Priority)
	}
	reg.plugins.Remove(pluginID)
}

// Shutdown shuts down all plugins in the registry.
func (reg *Registry) Shutdown() {
	_, span := otel.Tracer(config.TracerName).Start(reg.ctx, "Shutdown")
	defer span.End()

	reg.plugins.ForEach(func(key, value interface{}) bool {
		if id, ok := key.(sdkPlugin.Identifier); ok {
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
func (reg *Registry) Hooks() map[v1.HookName]map[sdkPlugin.Priority]sdkPlugin.Method {
	_, span := otel.Tracer(config.TracerName).Start(reg.ctx, "Hooks")
	defer span.End()

	return reg.hooks
}

// AddHook adds a hook with a priority to the hooks map.
func (reg *Registry) AddHook(hookName v1.HookName, priority sdkPlugin.Priority, hookMethod sdkPlugin.Method) {
	_, span := otel.Tracer(config.TracerName).Start(reg.ctx, "AddHook")
	defer span.End()

	if len(reg.hooks[hookName]) == 0 {
		reg.hooks[hookName] = map[sdkPlugin.Priority]sdkPlugin.Method{priority: hookMethod}
	} else {
		if _, ok := reg.hooks[hookName][priority]; ok {
			reg.Logger.Warn().Fields(
				map[string]any{
					"hookName": hookName.String(),
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
// hook, and so on. The result of the last hook is eventually returned.
// The opts are passed to the hooks as well to allow them to use the grpc.CallOption.
func (reg *Registry) Run(
	ctx context.Context,
	args map[string]any,
	hookName v1.HookName,
	opts ...grpc.CallOption,
) (map[string]any, *gerr.GatewayDError) {
	_, span := otel.Tracer(config.TracerName).Start(reg.ctx, "Run")
	defer span.End()

	metrics.PluginHooksExecuted.Inc()

	if ctx == nil {
		return nil, gerr.ErrNilContext
	}

	// Inherit context.
	inheritedCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	// Cast custom fields to their primitive types, like time.Duration to float64.
	args = castToPrimitiveTypes(args)

	// Create v1.Struct from args.
	var params *v1.Struct
	if len(args) == 0 {
		params = &v1.Struct{}
	} else if casted, err := v1.NewStruct(args); err == nil {
		params = casted
	} else {
		span.RecordError(err)
		return nil, gerr.ErrCastFailed.Wrap(err)
	}

	// Sort hooks by priority.
	priorities := make([]sdkPlugin.Priority, 0, len(reg.hooks[hookName]))
	for priority := range reg.hooks[hookName] {
		priorities = append(priorities, priority)
	}
	sort.SliceStable(priorities, func(i, j int) bool {
		return priorities[i] < priorities[j]
	})

	// Run hooks, passing the result of the previous hook to the next one.
	returnVal := &v1.Struct{}
	var outputs []*sdkAct.Output
	// The signature of parameters and args MUST be the same for this to work.
	for idx, priority := range priorities {
		var result *v1.Struct
		var err error
		if idx == 0 {
			result, err = reg.hooks[hookName][priority](inheritedCtx, params, opts...)
		} else {
			result, err = reg.hooks[hookName][priority](inheritedCtx, returnVal, opts...)
		}

		if err != nil {
			reg.Logger.Error().Err(err).Fields(
				map[string]any{
					"hookName": hookName.String(),
					"priority": priority,
				},
			).Msg("Hook returned an error")
			span.RecordError(err)
		}

		if result == nil {
			// Remove the hook from the registry, log the error and execute the next hook.
			reg.Logger.Error().Fields(
				map[string]any{
					"hookName": hookName.String(),
					"priority": priority,
				},
			).Msg("Hook returned nil result, so it won't work properly")
			delete(reg.hooks[hookName], priority)
			continue
		}

		out, terminal := reg.Apply(
			sdkAct.Hook{
				Name:     hookName.String(),
				Priority: uint(priority),
				Params:   params.AsMap(),
				Result:   result.AsMap(),
			},
		)
		outputs = append(outputs, out...)

		if terminal {
			// Any signal matching a policy with a terminal action
			// will terminate the execution of the rest of the hooks.
			reg.Logger.Debug().Msg("Terminal signal received")
			span.AddEvent("Terminal signal received")
			resultMap := result.AsMap()
			resultMap[sdkAct.Outputs] = outputs
			resultMap[sdkAct.Terminal] = true
			return resultMap, nil
		}

		returnVal = result
	}

	returnMap := returnVal.AsMap()
	returnMap[sdkAct.Outputs] = outputs
	return returnMap, nil
}

// Apply applies policies to the result.
func (reg *Registry) Apply(hook sdkAct.Hook) ([]*sdkAct.Output, bool) {
	_, span := otel.Tracer(config.TracerName).Start(reg.ctx, "Apply")
	defer span.End()

	// Get signals from the result.
	signals := getSignals(hook.Result)
	// Apply policies to the signals.
	// The outputs contain the verdicts of the policies and their metadata.
	// And using this list, the caller can take further actions.
	outputs := applyPolicies(hook, signals, reg.Logger, reg.ActRegistry)

	// If no policies are found, return a default output.
	// Note: this should never happen, as the default policy is always loaded.
	if len(outputs) == 0 {
		reg.Logger.Debug().Msg("No policies found for the given signals")
		return nil, false
	}

	// Check if any of the policies have a terminal action.
	var terminal bool
	for _, output := range outputs {
		if output.Verdict != nil && cast.ToBool(output.Verdict) && output.Terminal {
			terminal = true
			break
		}
	}

	return outputs, terminal
}

// LoadPlugins loads plugins from the config file.
func (reg *Registry) LoadPlugins(
	ctx context.Context, plugins []config.Plugin, startTimeout time.Duration,
) {
	// TODO: Append built-in plugins to the list of plugins
	// Built-in plugins are plugins that are compiled and shipped with the gatewayd binary.
	ctx, span := otel.Tracer("").Start(ctx, "Load plugins")
	defer span.End()

	// Add each plugin to the registry.
	for priority, pCfg := range plugins {
		pluginCtx, span := otel.Tracer("").Start(ctx, "Load plugin")
		span.SetAttributes(attribute.Int("priority", priority))
		span.SetAttributes(attribute.String("name", pCfg.Name))
		span.SetAttributes(attribute.String("url", pCfg.URL))
		span.SetAttributes(attribute.Bool("enabled", pCfg.Enabled))
		span.SetAttributes(attribute.String("checksum", pCfg.Checksum))
		span.SetAttributes(attribute.String("local_path", pCfg.LocalPath))
		span.SetAttributes(attribute.StringSlice("args", pCfg.Args))
		span.SetAttributes(attribute.StringSlice("env", pCfg.Env))
		defer span.End()

		reg.Logger.Debug().Str("name", pCfg.Name).Msg("Loading plugin")
		plugin := &Plugin{
			ID: sdkPlugin.Identifier{
				Name:     pCfg.Name,
				Checksum: pCfg.Checksum,
			},
			Enabled:   pCfg.Enabled,
			LocalPath: pCfg.LocalPath,
			Args:      pCfg.Args,
			Env:       pCfg.Env,
		}

		span.AddEvent("Created plugin object")

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

		var secureConfig *goplugin.SecureConfig
		if !reg.DevMode {
			// Checksum of the plugin.
			if plugin.ID.Checksum == "" {
				reg.Logger.Debug().Str("name", plugin.ID.Name).Msg(
					"Checksum of plugin doesn't exist or is not set")
				continue
			}

			// Verify the checksum.
			// TODO: Load the plugin from a remote location if the checksum didn't match?
			checksum, err := hex.DecodeString(plugin.ID.Checksum)
			if err != nil {
				reg.Logger.Debug().Str("name", plugin.ID.Name).Err(err).Msg(
					"Failed to decode checksum")
				continue
			}

			if len(checksum) != sha256.Size {
				reg.Logger.Debug().Str("name", plugin.ID.Name).Msg("Invalid checksum length")
				continue
			}

			secureConfig = &goplugin.SecureConfig{
				Checksum: checksum,
				Hash:     sha256.New(),
			}

			span.AddEvent("Created secure config for validating plugin checksum")
		} else {
			span.AddEvent("Skipping plugin checksum verification (dev mode)")
		}

		// Plugin priority is determined by the order in which the plugin is listed
		// in the config file. Built-in plugins are loaded first, followed by user-defined
		// plugins. Built-in plugins have a priority of 0 to 999, and user-defined plugins
		// have a priority of 1000 or greater.
		plugin.Priority = sdkPlugin.Priority(config.PluginPriorityStart + uint(priority))

		logAdapter := logging.NewHcLogAdapter(&reg.Logger, pCfg.Name)

		plugin.Client = goplugin.NewClient(
			&goplugin.ClientConfig{
				HandshakeConfig: v1.Handshake,
				Plugins:         v1.GetPluginMap(plugin.ID.Name),
				Cmd:             NewCommand(plugin.LocalPath, plugin.Args, plugin.Env),
				AllowedProtocols: []goplugin.Protocol{
					goplugin.ProtocolGRPC,
				},
				SecureConfig: secureConfig,
				Logger:       logAdapter,
				Managed:      true,
				MinPort:      config.DefaultMinPort,
				MaxPort:      config.DefaultMaxPort,
				AutoMTLS:     true,
				StartTimeout: startTimeout,
			},
		)

		span.AddEvent("Created plugin client")

		reg.Logger.Debug().Str("name", plugin.ID.Name).Msg("Plugin loaded")
		if _, err := plugin.Start(); err != nil {
			reg.Logger.Error().Str("name", plugin.ID.Name).Err(err).Msg(
				"Failed to start plugin")
			plugin.Client.Kill()
			continue
		}

		span.AddEvent("Started plugin")

		// Load metadata from the plugin.
		pluginV1, err := plugin.Dispense()
		if err != nil {
			reg.Logger.Debug().Str("name", plugin.ID.Name).Err(err).Msg(
				"Failed to dispense plugin")
			plugin.Client.Kill()
			continue
		}

		metadata, origErr := pluginV1.GetPluginConfig( //nolint:contextcheck
			context.Background(), &v1.Struct{})
		if origErr != nil || metadata == nil {
			reg.Logger.Debug().Str("name", plugin.ID.Name).Err(origErr).Msg(
				"Failed to get plugin metadata")
			continue
		}

		span.AddEvent("Fetched plugin metadata")

		// Retrieve plugin requirements.
		if requires, ok := metadata.GetFields()["requires"]; ok && requires != nil && requires.GetListValue() != nil {
			if err := mapstructure.Decode(
				requires.GetListValue().AsSlice(), &plugin.Requires); err != nil {
				reg.Logger.Debug().Err(err).Msg("Failed to decode plugin requirements")
			}
		} else {
			reg.Logger.Debug().Str("name", plugin.ID.Name).Msg(
				"Plugin doesn't have any requirements")
		}

		// Too many requirements or not enough plugins loaded.
		// Note: Plugin requirements won't cause the required plugins to be loaded.
		if len(plugin.Requires) > reg.plugins.Size() {
			reg.Logger.Debug().Msg(
				"The plugin has too many requirements, " +
					"and not enough of them exist in the registry, so it won't work properly")
		}

		// Check if the plugin requirements are met.
		for _, req := range plugin.Requires {
			if !reg.Exists(req.Name, req.Version, req.RemoteURL) {
				reg.Logger.Debug().Fields(
					map[string]any{
						"name":        plugin.ID.Name,
						"requirement": req.Name,
					},
				).Msg("The plugin requirement is not met, so it won't work properly")
				if reg.Compatibility == config.Strict {
					reg.Logger.Debug().Str("name", plugin.ID.Name).Msg(
						"Registry is in strict compatibility mode, so the plugin won't be loaded")
					plugin.Stop() // Stop the plugin.
					continue
				}
				reg.Logger.Debug().Fields(
					map[string]any{
						"name":        plugin.ID.Name,
						"requirement": req.Name,
					},
				).Msg("Registry is in loose compatibility mode, " +
					"so the plugin will be loaded anyway")
			}
		}

		span.AddEvent("Verified plugin requirements")

		plugin.ID.RemoteURL = metadata.GetFields()["id"].GetStructValue().GetFields()["remoteUrl"].GetStringValue()
		plugin.ID.Version = metadata.GetFields()["id"].GetStructValue().GetFields()["version"].GetStringValue()
		plugin.Description = metadata.GetFields()["description"].GetStringValue()
		plugin.License = metadata.GetFields()["license"].GetStringValue()
		plugin.ProjectURL = metadata.GetFields()["projectUrl"].GetStringValue()
		// Retrieve authors.
		if metadata.GetFields()["authors"] != nil && metadata.GetFields()["authors"].GetListValue() != nil {
			if err := mapstructure.Decode(metadata.GetFields()["authors"].GetListValue().AsSlice(),
				&plugin.Authors); err != nil {
				reg.Logger.Debug().Err(err).Msg("Failed to decode plugin authors")
			}
		} else {
			reg.Logger.Debug().Str("name", plugin.ID.Name).Msg(
				"Plugin doesn't have any authors")
		}

		// Retrieve hooks.
		if metadata.GetFields()["hooks"] != nil && metadata.GetFields()["hooks"].GetListValue() != nil {
			if err := mapstructure.Decode(metadata.GetFields()["hooks"].GetListValue().AsSlice(),
				&plugin.Hooks); err != nil {
				reg.Logger.Debug().Err(err).Msg("Failed to decode plugin hooks")
			}
		} else {
			reg.Logger.Debug().Str("name", plugin.ID.Name).Msg(
				"Plugin doesn't attach to any hooks")
		}

		// Retrieve plugin config.
		plugin.Config = make(map[string]string)
		if metadata.GetFields()["config"] != nil && metadata.GetFields()["config"].GetStructValue() != nil {
			for key, value := range metadata.GetFields()["config"].GetStructValue().AsMap() {
				if val, ok := value.(string); ok {
					plugin.Config[key] = val
				} else {
					reg.Logger.Debug().Str("key", key).Msg(
						"Failed to decode plugin config")
				}
			}
		} else {
			reg.Logger.Debug().Str("name", plugin.ID.Name).Msg(
				"Plugin doesn't have any config")
		}

		span.AddEvent("Decoded plugin metadata")

		reg.Logger.Trace().Msgf("Plugin metadata: %+v", plugin)

		reg.Add(plugin)
		reg.Logger.Debug().Str("name", plugin.ID.Name).Msg("Plugin metadata loaded")

		span.AddEvent("Plugin metadata loaded")

		reg.RegisterHooks(pluginCtx, plugin.ID)
		reg.Logger.Debug().Str("name", plugin.ID.Name).Msg("Plugin hooks registered")

		span.AddEvent("Registered plugin hooks")

		metrics.PluginsLoaded.Inc()
		reg.Logger.Info().Str("name", plugin.ID.Name).Msg("Plugin is ready")
	}
}

// RegisterHooks registers the hooks for the given plugin.
func (reg *Registry) RegisterHooks(ctx context.Context, pluginID sdkPlugin.Identifier) {
	_, span := otel.Tracer("gatewayd").Start(ctx, "Register plugin hooks")
	defer span.End()

	pluginImpl := reg.Get(pluginID)
	reg.Logger.Debug().Str("name", pluginImpl.ID.Name).Msg(
		"Registering hooks for plugin")
	var pluginV1 v1.GatewayDPluginServiceClient
	var err *gerr.GatewayDError
	if pluginV1, err = pluginImpl.Dispense(); err != nil {
		reg.Logger.Debug().Str("name", pluginImpl.ID.Name).Err(err).Msg(
			"Failed to dispense plugin")
		span.RecordError(err)
		return
	}

	reg.Logger.Info().Str("name", pluginImpl.ID.Name).Msg("Registering plugin hooks")
	hooks := make([]string, 0)
	for _, hook := range pluginImpl.Hooks {
		hooks = append(hooks, hook.String())
	}
	span.SetAttributes(attribute.StringSlice("hooks", hooks))
	reg.Logger.Debug().Str("name", pluginImpl.ID.Name).Msgf(
		"Plugin hooks: %+v", pluginImpl.Hooks)

	for _, hookName := range pluginImpl.Hooks {
		var hookMethod sdkPlugin.Method
		switch hookName {
		case v1.HookName_HOOK_NAME_UNSPECIFIED:
			reg.Logger.Debug().Str("name", pluginImpl.ID.Name).Msg(
				"Plugin hook is unspecified or invalid, so it won't work properly")
			reg.Logger.Debug().Str("name", pluginImpl.ID.Name).Msg(
				"Consider casting the enum value to an int32")
		case v1.HookName_HOOK_NAME_ON_CONFIG_LOADED:
			hookMethod = pluginV1.OnConfigLoaded
		case v1.HookName_HOOK_NAME_ON_NEW_LOGGER:
			hookMethod = pluginV1.OnNewLogger
		case v1.HookName_HOOK_NAME_ON_NEW_POOL:
			hookMethod = pluginV1.OnNewPool
		case v1.HookName_HOOK_NAME_ON_NEW_CLIENT:
			hookMethod = pluginV1.OnNewClient
		case v1.HookName_HOOK_NAME_ON_NEW_PROXY:
			hookMethod = pluginV1.OnNewProxy
		case v1.HookName_HOOK_NAME_ON_NEW_SERVER:
			hookMethod = pluginV1.OnNewServer
		case v1.HookName_HOOK_NAME_ON_SIGNAL:
			hookMethod = pluginV1.OnSignal
		case v1.HookName_HOOK_NAME_ON_RUN:
			hookMethod = pluginV1.OnRun
		case v1.HookName_HOOK_NAME_ON_BOOTING:
			hookMethod = pluginV1.OnBooting
		case v1.HookName_HOOK_NAME_ON_BOOTED:
			hookMethod = pluginV1.OnBooted
		case v1.HookName_HOOK_NAME_ON_OPENING:
			hookMethod = pluginV1.OnOpening
		case v1.HookName_HOOK_NAME_ON_OPENED:
			hookMethod = pluginV1.OnOpened
		case v1.HookName_HOOK_NAME_ON_CLOSING:
			hookMethod = pluginV1.OnClosing
		case v1.HookName_HOOK_NAME_ON_CLOSED:
			hookMethod = pluginV1.OnClosed
		case v1.HookName_HOOK_NAME_ON_TRAFFIC:
			hookMethod = pluginV1.OnTraffic
		case v1.HookName_HOOK_NAME_ON_TRAFFIC_FROM_CLIENT:
			hookMethod = pluginV1.OnTrafficFromClient
		case v1.HookName_HOOK_NAME_ON_TRAFFIC_TO_SERVER:
			hookMethod = pluginV1.OnTrafficToServer
		case v1.HookName_HOOK_NAME_ON_TRAFFIC_FROM_SERVER:
			hookMethod = pluginV1.OnTrafficFromServer
		case v1.HookName_HOOK_NAME_ON_TRAFFIC_TO_CLIENT:
			hookMethod = pluginV1.OnTrafficToClient
		case v1.HookName_HOOK_NAME_ON_SHUTDOWN:
			hookMethod = pluginV1.OnShutdown
		case v1.HookName_HOOK_NAME_ON_TICK:
			hookMethod = pluginV1.OnTick
		case v1.HookName_HOOK_NAME_ON_HOOK:
			hookMethod = pluginV1.OnHook
		default:
			continue
		}

		reg.Logger.Debug().Fields(map[string]any{
			"hook":     hookName.String(),
			"priority": pluginImpl.Priority,
			"name":     pluginImpl.ID.Name,
		}).Msg("Registering hook")
		metrics.PluginHooksRegistered.Inc()
		reg.AddHook(hookName, pluginImpl.Priority, hookMethod)
	}
}
