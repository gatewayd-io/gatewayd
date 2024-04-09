package act

import (
	"context"
	"errors"
	"slices"
	"time"

	sdkAct "github.com/gatewayd-io/gatewayd-plugin-sdk/act"
	"github.com/gatewayd-io/gatewayd/config"
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
	// Default timeout for running actions
	defaultActionTimeout time.Duration

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
	defaultActionTimeout time.Duration,
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
		logger.Warn().Str("name", defaultPolicy).Msgf(
			"The specified default policy does not exist, using %s", config.DefaultPolicy)
		defaultPolicy = config.DefaultPolicy
	}

	logger.Debug().Str("name", defaultPolicy).Msg("Using default policy")

	return &Registry{
		logger:               logger,
		policyTimeout:        policyTimeout,
		defaultActionTimeout: defaultActionTimeout,
		Signals:              builtinSignals,
		Policies:             builtinsPolicies,
		Actions:              builtinActions,
		DefaultPolicy:        builtinsPolicies[defaultPolicy],
		DefaultSignal:        builtinSignals[defaultPolicy],
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

	// Separate terminal and non-terminal signals to find contradictions.
	var terminal []string
	var nonTerminal []string
	for _, signal := range signals {
		action, exists := r.Actions[signal.Name]
		if exists && action.Sync && action.Terminal {
			terminal = append(terminal, signal.Name)
		} else if exists && action.Sync && !action.Terminal {
			nonTerminal = append(nonTerminal, signal.Name)
		}
	}

	var outputs []*sdkAct.Output
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

		// Apply the signal and append the output to the list of outputs.
		output, err := r.apply(signal)
		if err != nil {
			r.logger.Error().Err(err).Str("name", signal.Name).Msg("Error applying signal")
			// If there is an error evaluating the policy, continue to the next signal.
			// This also prevents stack overflows from infinite loops of the external
			// if condition below.
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

	// Create a context with a timeout for policy evaluation.
	ctx, cancel := context.WithTimeout(context.Background(), r.policyTimeout)
	defer cancel()

	// Evaluate the policy.
	// TODO: Policy should be able to receive other parameters like server and client IPs, etc.
	verdict, err := policy.Eval(
		ctx, sdkAct.Input{
			Name:   signal.Name,
			Policy: policy.Metadata,
			Signal: signal.Metadata,
			// Action dictates the sync mode, not the signal.
			Sync: action.Sync,
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

// Run runs the function associated with the output.MatchedPolicy and
// returns its result. If the action is synchronous, the result is returned
// immediately. If the action is asynchronous, the result is nil and the
// error is ErrAsyncAction, which is a sentinel error to indicate that the
// action is running asynchronously.
func (r *Registry) Run(
	output *sdkAct.Output, params ...sdkAct.Parameter,
) (any, *gerr.GatewayDError) {
	// In certain cases, the output may be nil, for example, if the policy
	// evaluation fails. In this case, the run is aborted.
	if output == nil {
		// This should never happen, since the output is always set by the registry
		// to be the default policy if no signals are provided.
		r.logger.Debug().Msg("Output is nil, run aborted")
		return nil, gerr.ErrNilPointer
	}

	action, ok := r.Actions[output.MatchedPolicy]
	if !ok {
		r.logger.Warn().Str("matchedPolicy", output.MatchedPolicy).Msg(
			"Action does not exist, run aborted")
		return nil, gerr.ErrActionNotExist
	}

	// Prepend the logger to the parameters.
	params = append([]sdkAct.Parameter{WithLogger(r.logger)}, params...)

	timeout := r.defaultActionTimeout
	if action.Timeout > 0 {
		timeout = time.Duration(action.Timeout) * time.Second
	}
	var ctx context.Context
	var cancel context.CancelFunc
	// if timeout is zero, then the context should not have timeout
	if timeout > 0 {
		ctx, cancel = context.WithTimeout(context.Background(), timeout)
	} else {
		ctx, cancel = context.WithCancel(context.Background())
	}
	// If the action is synchronous, run it and return the result immediately.
	if action.Sync {
		defer cancel()
		return runActionWithTimeout(ctx, action, output, params, r.logger)
	}

	// Run the action asynchronously.
	go func() {
		defer cancel()
		_, _ = runActionWithTimeout(ctx, action, output, params, r.logger)
	}()
	return nil, gerr.ErrAsyncAction
}

func runActionWithTimeout(
	ctx context.Context,
	action *sdkAct.Action,
	output *sdkAct.Output,
	params []sdkAct.Parameter,
	logger zerolog.Logger,
) (any, *gerr.GatewayDError) {
	execMode := "sync"
	if !action.Sync {
		execMode = "async"
	}
	logger.Debug().Fields(map[string]interface{}{
		"executionMode": execMode,
		"action":        action.Name,
	}).Msgf("Running action")
	outputChan := make(chan any)
	errChan := make(chan *gerr.GatewayDError)

	go func() {
		actionOutput, err := action.Run(output.Metadata, params...)
		if err != nil {
			logger.Error().Err(err).Str("action", action.Name).Msg("Error running action")
			errChan <- gerr.ErrRunningAction.Wrap(err)
		}
		outputChan <- actionOutput
	}()
	select {
	case <-ctx.Done():
		logger.Error().Str("action", action.Name).Msg("Action timed out")
		return nil, gerr.ErrRunningActionTimeout
	case actionOutput := <-outputChan:
		return actionOutput, nil
	case err := <-errChan:
		return nil, err
	}
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
