package act

import (
	"context"
	"errors"
	"time"

	sdkAct "github.com/gatewayd-io/gatewayd-plugin-sdk/act"
	"github.com/rs/zerolog"
)

type IRegistry interface {
	Add(policy *sdkAct.Policy)
	Apply(signals []sdkAct.Signal) []*sdkAct.Output
	Run(output *sdkAct.Output) (any, error)
}

// Registry keeps track of all policies and actions.
type Registry struct {
	logger  zerolog.Logger
	timeout time.Duration // Timeout for policy evaluation

	Policies      map[string]*sdkAct.Policy
	Actions       map[string]*sdkAct.Action
	DefaultPolicy *sdkAct.Policy
}

// NewRegistry creates a new registry with the specified default policy and timeout.
func NewRegistry(defaultPolicy string, timeout time.Duration, logger zerolog.Logger) *Registry {
	policies := make(map[string]*sdkAct.Policy)
	for _, policy := range builtinsPolicies {
		policies[policy.Name] = policy
		logger.Debug().Str("name", policy.Name).Msg("Registered builtin policy")
	}

	actions := make(map[string]*sdkAct.Action)
	for _, action := range builtinActions {
		actions[action.Name] = action
		logger.Debug().Str("name", action.Name).Msg("Registered builtin action")
	}

	// The default policy must exist, otherwise use passthrough.
	if _, exists := policies[defaultPolicy]; !exists || defaultPolicy == "" {
		logger.Warn().Str("name", defaultPolicy).Msg(
			"The specified default policy does not exist, using passthrough")
		defaultPolicy = "passthrough"
	}

	return &Registry{
		logger:        logger,
		timeout:       timeout,
		Actions:       actions,
		Policies:      policies,
		DefaultPolicy: policies[defaultPolicy],
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
	DefaultOutputs := []*sdkAct.Output{
		DefaultOutput(),
	}

	if len(signals) == 0 {
		return DefaultOutputs
	}

	// TODO: Check for non-contradictory actions (forward vs. drop)

	outputs := []*sdkAct.Output{}
	for _, signal := range signals {
		output, err := r.apply(signal)
		if err != nil {
			r.logger.Error().Err(err).Str("name", signal.Name).Msg("Error applying signal")
			continue
		}
		outputs = append(outputs, output)
	}

	if len(outputs) == 0 {
		return DefaultOutputs
	}

	return outputs
}

// apply applies the signal to the registry and returns the output.
func (r *Registry) apply(signal sdkAct.Signal) (*sdkAct.Output, error) {
	action, ok := r.Actions[signal.Name]
	if !ok {
		return nil, errors.New("No matching action")
	}

	policy, ok := r.Policies[action.Name]
	if !ok {
		return nil, errors.New("No matching policy")
	}

	ctx, cancel := context.WithTimeout(context.Background(), r.timeout)
	defer cancel()

	// Action dictates the sync mode, not the signal.
	verdict, err := policy.Eval(
		ctx, sdkAct.Input{
			Name:   signal.Name,
			Policy: policy.Metadata,
			Signal: signal.Metadata,
			Sync:   action.Sync,
		},
	)
	if err != nil {
		return nil, err
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
func (r *Registry) Run(output *sdkAct.Output) (any, error) {
	if output == nil {
		r.logger.Warn().Msg("Output is nil, not running")
		return nil, errors.New("Output is nil")
	}

	action, ok := r.Actions[output.MatchedPolicy]
	if !ok {
		r.logger.Warn().Str("matched_policy", output.MatchedPolicy).Msg(
			"Action does not exist, not running")
		return nil, errors.New("Action does not exist")
	}

	if action.Sync {
		r.logger.Debug().Fields(map[string]interface{}{
			"execution_mode": "sync",
			"action":         action.Name,
		}).Msgf("Running action")
		return action.Run(output.Metadata, WithLogger(r.logger))
	} else {
		r.logger.Debug().Fields(map[string]interface{}{
			"execution_mode": "async",
			"action":         action.Name,
		}).Msgf("Running action")
		go action.Run(output.Metadata, WithLogger(r.logger))
		return nil, nil
	}
}

func WithLogger(logger zerolog.Logger) sdkAct.Parameter {
	return sdkAct.Parameter{
		Key:   "logger",
		Value: logger,
	}
}
