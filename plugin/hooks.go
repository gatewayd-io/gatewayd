package plugin

import (
	"context"
	"fmt"
	"sort"

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

func NewHookConfig() *HookConfig {
	return &HookConfig{
		hooks: map[HookType]map[Priority]HookDef{},
	}
}

func (h *HookConfig) Hooks() map[HookType]map[Priority]HookDef {
	return h.hooks
}

func (h *HookConfig) Add(hookType HookType, prio Priority, hook HookDef) {
	if len(h.hooks[hookType]) == 0 {
		h.hooks[hookType] = map[Priority]HookDef{prio: hook}
	} else {
		if _, ok := h.hooks[hookType][prio]; ok {
			h.Logger.Warn().Msgf("Hook %s replaced with priority %d.", hookType, prio)
		}
		h.hooks[hookType][prio] = hook
	}
}

func (h *HookConfig) Get(hookType HookType) map[Priority]HookDef {
	return h.hooks[hookType]
}

//nolint:funlen,contextcheck
func (h *HookConfig) Run(
	ctx context.Context,
	args *structpb.Struct,
	hookType HookType,
	verification Policy,
	opts ...grpc.CallOption,
) (*structpb.Struct, error) {
	if ctx == nil {
		ctx = context.Background()
	}

	// Sort hooks by priority
	priorities := make([]Priority, 0, len(h.hooks[hookType]))
	for prio := range h.hooks[hookType] {
		priorities = append(priorities, prio)
	}
	sort.SliceStable(priorities, func(i, j int) bool {
		return priorities[i] < priorities[j]
	})

	// Run hooks, passing the result of the previous hook to the next one
	returnVal := &structpb.Struct{}
	var removeList []Priority
	// The signature of parameters and args MUST be the same for this to work
	for idx, prio := range priorities {
		var result *structpb.Struct
		var err error
		if idx == 0 {
			// TODO: Run hooks from the registry
			result, err = h.hooks[hookType][prio](ctx, args, opts...)
		} else {
			// TODO: Run hooks from the registry
			result, err = h.hooks[hookType][prio](ctx, returnVal, opts...)
		}

		// This is done to ensure that the return value of the hook is always valid,
		// and that the hook does not return any unexpected values.
		// If the verification mode is non-strict (permissive), let the plugin pass
		// extra keys/values to the next plugin in chain.
		if Verify(args, result) || verification == PassDown {
			// Update the last return value with the current result
			returnVal = result
			continue
		}

		// At this point, the hook returned an invalid value, so we need to handle it.
		// The result of the current hook will be ignored, regardless of the policy.
		switch verification {
		// Ignore the result of this plugin, log an error and execute the next hook.
		case Ignore:
			errMsg := fmt.Sprintf(
				"Hook %s (Prio %d) returned invalid value, ignoring", hookType, prio)
			// Logger is not available when loading configuration, so we can't log anything
			if hookType != OnConfigLoaded {
				h.Logger.Error().Msgf(errMsg)
			} else {
				panic(errMsg)
			}
			if idx == 0 {
				returnVal = args
			}
		// Abort execution of the plugins, log the error and return the result of the last hook.
		case Abort:
			errMsg := fmt.Sprintf(
				"Hook %s (Prio %d) returned invalid value, aborting", hookType, prio)
			if hookType != OnConfigLoaded {
				h.Logger.Error().Msgf(errMsg)
			} else {
				panic(errMsg)
			}
			if idx == 0 {
				return args, err
			}
			return returnVal, err
		// Remove the hook from the registry, log the error and execute the next hook.
		case Remove:
			errMsg := fmt.Sprintf(
				"Hook %s (Prio %d) returned invalid value, removing", hookType, prio)
			if hookType != OnConfigLoaded {
				h.Logger.Error().Msgf(errMsg)
			} else {
				panic(errMsg)
			}
			removeList = append(removeList, prio)
			if idx == 0 {
				returnVal = args
			}
		case PassDown:
		default:
			returnVal = result
		}
	}

	// Remove hooks that failed verification
	for _, prio := range removeList {
		delete(h.hooks[hookType], prio)
	}

	return returnVal, nil
}
