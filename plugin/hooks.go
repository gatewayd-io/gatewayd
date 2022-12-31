package plugin

import (
	"context"
	"sort"

	gerr "github.com/gatewayd-io/gatewayd/errors"
	"github.com/rs/zerolog"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/structpb"
)

type (
	// Priority is the priority of a hook.
	// Smaller values are executed first (higher priority).
	Priority uint
	HookType string
	HookDef  func(
		context.Context, *structpb.Struct, ...grpc.CallOption) (*structpb.Struct, error)
	Policy int
)

const (
	// Non-strict (permissive) mode.
	PassDown Policy = iota // Pass down the extra keys/values in result to the next plugins
	// Strict mode.
	Ignore // Ignore errors and continue
	Abort  // Abort on first error and return results
	Remove // Remove the hook from the list on error and continue
)

const (
	// Run command hooks (cmd/run.go).
	OnConfigLoaded HookType = "onConfigLoaded"
	OnNewLogger    HookType = "onNewLogger"
	OnNewPool      HookType = "onNewPool"
	OnNewProxy     HookType = "onNewProxy"
	OnNewServer    HookType = "onNewServer"
	OnSignal       HookType = "onSignal"
	// Server hooks (network/server.go).
	OnRun            HookType = "onRun"
	OnBooting        HookType = "onBooting"
	OnBooted         HookType = "onBooted"
	OnOpening        HookType = "onOpening"
	OnOpened         HookType = "onOpened"
	OnClosing        HookType = "onClosing"
	OnClosed         HookType = "onClosed"
	OnTraffic        HookType = "onTraffic"
	OnIngressTraffic HookType = "onIngressTraffic"
	OnEgressTraffic  HookType = "onEgressTraffic"
	OnShutdown       HookType = "onShutdown"
	OnTick           HookType = "onTick"
	// Pool hooks (network/pool.go).
	OnNewClient HookType = "onNewClient"
)

type HookConfig struct {
	hooks        map[HookType]map[Priority]HookDef
	Logger       zerolog.Logger
	Verification Policy
}

// NewHookConfig returns a new HookConfig.
func NewHookConfig() *HookConfig {
	return &HookConfig{
		hooks: map[HookType]map[Priority]HookDef{},
	}
}

// Hooks returns the hooks.
func (h *HookConfig) Hooks() map[HookType]map[Priority]HookDef {
	return h.hooks
}

// Add adds a hook with a priority to the hooks map.
func (h *HookConfig) Add(hookType HookType, prio Priority, hook HookDef) {
	if len(h.hooks[hookType]) == 0 {
		h.hooks[hookType] = map[Priority]HookDef{prio: hook}
	} else {
		if _, ok := h.hooks[hookType][prio]; ok {
			h.Logger.Warn().Fields(
				map[string]interface{}{
					"hookType": hookType,
					"priority": prio,
				},
			).Msg("Hook is replaced")
		}
		h.hooks[hookType][prio] = hook
	}
}

// Get returns the hooks of a specific type.
func (h *HookConfig) Get(hookType HookType) map[Priority]HookDef {
	return h.hooks[hookType]
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
// are passed down to the next hook. The verification mode is set to PassDown by default.
// The opts are passed to the hooks as well to allow them to use the grpc.CallOption.
//
//nolint:funlen,contextcheck
func (h *HookConfig) Run(
	ctx context.Context,
	args map[string]interface{},
	hookType HookType,
	verification Policy,
	opts ...grpc.CallOption,
) (map[string]interface{}, *gerr.GatewayDError) {
	if ctx == nil {
		ctx = context.Background()
	}

	var params *structpb.Struct
	if args == nil {
		params = &structpb.Struct{}
	} else if casted, err := structpb.NewStruct(args); err == nil {
		params = casted
	} else {
		return nil, gerr.ErrCastFailed.Wrap(err)
	}

	// Sort hooks by priority.
	priorities := make([]Priority, 0, len(h.hooks[hookType]))
	for prio := range h.hooks[hookType] {
		priorities = append(priorities, prio)
	}
	sort.SliceStable(priorities, func(i, j int) bool {
		return priorities[i] < priorities[j]
	})

	// Run hooks, passing the result of the previous hook to the next one.
	returnVal := &structpb.Struct{}
	var removeList []Priority
	// The signature of parameters and args MUST be the same for this to work.
	for idx, prio := range priorities {
		var result *structpb.Struct
		var err error
		if idx == 0 {
			// TODO: Run hooks from the registry
			result, err = h.hooks[hookType][prio](ctx, params, opts...)
		} else {
			// TODO: Run hooks from the registry
			result, err = h.hooks[hookType][prio](ctx, returnVal, opts...)
		}

		// This is done to ensure that the return value of the hook is always valid,
		// and that the hook does not return any unexpected values.
		// If the verification mode is non-strict (permissive), let the plugin pass
		// extra keys/values to the next plugin in chain.
		if Verify(params, result) || verification == PassDown {
			// Update the last return value with the current result
			returnVal = result
			continue
		}

		// At this point, the hook returned an invalid value, so we need to handle it.
		// The result of the current hook will be ignored, regardless of the policy.
		switch verification {
		// Ignore the result of this plugin, log an error and execute the next hook.
		case Ignore:
			h.Logger.Error().Err(err).Fields(
				map[string]interface{}{
					"hookType": hookType,
					"priority": prio,
				},
			).Msg("Hook returned invalid value, ignoring")
			if idx == 0 {
				returnVal = params
			}
		// Abort execution of the plugins, log the error and return the result of the last hook.
		case Abort:
			h.Logger.Error().Err(err).Fields(
				map[string]interface{}{
					"hookType": hookType,
					"priority": prio,
				},
			).Msg("Hook returned invalid value, aborting")
			if idx == 0 {
				return args, nil
			}
			return returnVal.AsMap(), nil
		// Remove the hook from the registry, log the error and execute the next hook.
		case Remove:
			h.Logger.Error().Err(err).Fields(
				map[string]interface{}{
					"hookType": hookType,
					"priority": prio,
				},
			).Msg("Hook returned invalid value, removing")
			removeList = append(removeList, prio)
			if idx == 0 {
				returnVal = params
			}
		case PassDown:
		default:
			returnVal = result
		}
	}

	// Remove hooks that failed verification.
	for _, prio := range removeList {
		delete(h.hooks[hookType], prio)
	}

	return returnVal.AsMap(), nil
}
