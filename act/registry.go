package act

import (
	"context"
	"errors"
	"slices"
	"time"

	sdkAct "github.com/gatewayd-io/gatewayd-plugin-sdk/act"
	gerr "github.com/gatewayd-io/gatewayd/errors"
	"github.com/rs/zerolog"
)

type IRegistry interface {
	Add(policy *sdkAct.Policy)
	Apply(signals []sdkAct.Signal) []*sdkAct.Output
	Run(output *sdkAct.Output, params ...sdkAct.Parameter) (any, *gerr.GatewayDError)
}

// Registry keeps track of all policies and actions.
type Registry struct {
	logger zerolog.Logger
	// Timeout for policy evaluation.
	policyTimeout time.Duration

	Signals       map[string]*sdkAct.Signal
	Policies      map[string]*sdkAct.Policy
	Actions       map[string]*sdkAct.Action
	DefaultPolicy *sdkAct.Policy
	DefaultSignal *sdkAct.Signal
}

var _ IRegistry = (*Registry)(nil)

// NewActRegistry creates a new act registry with the specified default policy and timeout
// and the builtin signals, policies, and actions.
func NewActRegistry(
	builtinSignals map[string]*sdkAct.Signal,
	builtinsPolicies map[string]*sdkAct.Policy,
	builtinActions map[string]*sdkAct.Action,
	defaultPolicy string,
	policyTimeout time.Duration,
	logger zerolog.Logger,
) *Registry {
	if builtinSignals == nil || builtinsPolicies == nil || builtinActions == nil {
		logger.Warn().Msg("Builtin signals, policies, or actions are nil, not adding")
		return nil
	}

	for _, signal := range builtinSignals {
		if signal == nil {
			logger.Warn().Msg("Signal is nil, not adding")
			return nil
		}
		logger.Debug().Str("name", signal.Name).Msg("Registered builtin signal")
	}

	for _, policy := range builtinsPolicies {
		if policy == nil {
			logger.Warn().Msg("Policy is nil, not adding")
			return nil
		}
		logger.Debug().Str("name", policy.Name).Msg("Registered builtin policy")
	}

	for _, action := range builtinActions {
		if action == nil {
			logger.Warn().Msg("Action is nil, not adding")
			return nil
		}
		logger.Debug().Str("name", action.Name).Msg("Registered builtin action")
	}

	// The default policy must exist, otherwise use passthrough.
	if _, exists := builtinsPolicies[defaultPolicy]; !exists || defaultPolicy == "" {
		logger.Warn().Str("name", defaultPolicy).Msg(
			"The specified default policy does not exist, using passthrough")
		defaultPolicy = "passthrough"
	}

	logger.Debug().Str("name", defaultPolicy).Msg("Using default policy")

	return &Registry{
		logger:        logger,
		policyTimeout: policyTimeout,
		Signals:       builtinSignals,
		Policies:      builtinsPolicies,
		Actions:       builtinActions,
		DefaultPolicy: builtinsPolicies[defaultPolicy],
		DefaultSignal: builtinSignals[defaultPolicy],
	}
}

// Add adds a policy to the registry.
func (r *Registry) Add(policy *sdkAct.Policy) {
	if policy == nil {
		r.logger.Warn().Msg("Policy is nil, not adding")
		return
	}

	if _, exists := r.Policies[policy.Name]; exists {
		r.logger.Warn().Str("name", policy.Name).Msg("Policy already exists, overwriting")
	}

	// Builtin policies are can be overwritten by user-defined policies.
	r.Policies[policy.Name] = policy
}

// Apply applies the signals to the registry and returns the outputs.
func (r *Registry) Apply(signals []sdkAct.Signal) []*sdkAct.Output {
	// If there are no signals, apply the default policy.
	if len(signals) == 0 {
		r.logger.Debug().Msg("No signals provided, applying default signal")
		return r.Apply([]sdkAct.Signal{*r.DefaultSignal})
	}

	terminal := []string{}
	nonTerminal := []string{}
	for _, signal := range signals {
		action, exists := r.Actions[signal.Name]
		if exists && action.Sync && action.Terminal {
			terminal = append(terminal, signal.Name)
		} else if exists && action.Sync && !action.Terminal {
			nonTerminal = append(nonTerminal, signal.Name)
		}
	}

	outputs := []*sdkAct.Output{}
	evalErr := false
	for _, signal := range signals {
		// Ignore contradictory actions (forward vs. terminate) if one of the signals is terminal.
		// If the signal is terminal, all non-terminal signals are ignored. Also, it only
		// makes sense to have a terminal signal if the action is synchronous and terminal.
		if len(terminal) > 0 && slices.Contains(nonTerminal, signal.Name) {
			r.logger.Warn().Str("name", signal.Name).Msg(
				"Terminal signal takes precedence, ignoring non-terminal signals")
			continue
		}

		output, err := r.apply(signal)
		if err != nil {
			r.logger.Error().Err(err).Str("name", signal.Name).Msg("Error applying signal")
			if errors.Is(err, gerr.ErrEvalError) {
				evalErr = true
			}
			continue
		}

		outputs = append(outputs, output)
	}

	if len(outputs) == 0 && !evalErr {
		return r.Apply([]sdkAct.Signal{*r.DefaultSignal})
	}

	return outputs
}

// apply applies the signal to the registry and returns the output.
func (r *Registry) apply(signal sdkAct.Signal) (*sdkAct.Output, *gerr.GatewayDError) {
	action, exists := r.Actions[signal.Name]
	if !exists {
		return nil, gerr.ErrActionNotMatched
	}

	policy, exists := r.Policies[action.Name]
	if !exists {
		return nil, gerr.ErrPolicyNotMatched
	}

	ctx, cancel := context.WithTimeout(context.Background(), r.policyTimeout)
	defer cancel()

	// Action dictates the sync mode, not the signal.
	// TODO: Policy should be able to receive other parameters like server and client IPs, etc.
	verdict, err := policy.Eval(
		ctx, sdkAct.Input{
			Name:   signal.Name,
			Policy: policy.Metadata,
			Signal: signal.Metadata,
			Sync:   action.Sync,
		},
	)
	if err != nil {
		return nil, gerr.ErrEvalError.Wrap(err)
	}

	return &sdkAct.Output{
		MatchedPolicy: policy.Name,
		Verdict:       verdict,
		Metadata:      signal.Metadata,
		Terminal:      action.Terminal,
		Sync:          action.Sync,
	}, nil
}

// Run runs the output and returns the result.
func (r *Registry) Run(
	output *sdkAct.Output, params ...sdkAct.Parameter,
) (any, *gerr.GatewayDError) {
	if output == nil {
		// This should never happen, since the output is always set by the registry
		// to be the default policy if no signals are provided.
		r.logger.Debug().Msg("Output is nil, run aborted")
		return nil, gerr.ErrNilPointer
	}

	action, ok := r.Actions[output.MatchedPolicy]
	if !ok {
		r.logger.Warn().Str("matched_policy", output.MatchedPolicy).Msg(
			"Action does not exist, run aborted")
		return nil, gerr.ErrActionNotExist
	}

	// Prepend the logger to the parameters.
	params = append([]sdkAct.Parameter{WithLogger(r.logger)}, params...)

	if action.Sync {
		r.logger.Debug().Fields(map[string]interface{}{
			"execution_mode": "sync",
			"action":         action.Name,
		}).Msgf("Running action")
		output, err := action.Run(output.Metadata, params...)
		if err != nil {
			r.logger.Error().Err(err).Str("action", action.Name).Msg("Error running action")
			return nil, gerr.ErrRunningAction.Wrap(err)
		}
		return output, nil
	}

	r.logger.Debug().Fields(map[string]interface{}{
		"execution_mode": "async",
		"action":         action.Name,
	}).Msgf("Running action")

	// Run the action asynchronously.
	// TODO: Add a way to cancel the action.
	go func(
		action *sdkAct.Action,
		output *sdkAct.Output,
		params []sdkAct.Parameter,
		logger zerolog.Logger,
	) {
		_, err := action.Run(output.Metadata, params...)
		if err != nil {
			logger.Error().Err(err).Str("action", action.Name).Msg("Error running action")
		}
	}(action, output, params, r.logger)

	return nil, gerr.ErrAsyncAction
}

// WithLogger returns a parameter with the logger to be used by the action.
// This is automatically prepended to the parameters when running an action.
func WithLogger(logger zerolog.Logger) sdkAct.Parameter {
	return sdkAct.Parameter{
		Key:   LoggerKey,
		Value: logger,
	}
}

// WithResult returns a parameter with the result of the plugin hook
// to be used by the action.
func WithResult(result map[string]any) sdkAct.Parameter {
	return sdkAct.Parameter{
		Key:   ResultKey,
		Value: result,
	}
}
