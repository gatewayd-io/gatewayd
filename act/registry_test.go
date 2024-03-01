package act

import (
	"bytes"
	"testing"
	"time"

	sdkAct "github.com/gatewayd-io/gatewayd-plugin-sdk/act"
	"github.com/gatewayd-io/gatewayd/config"
	gerr "github.com/gatewayd-io/gatewayd/errors"
	"github.com/rs/zerolog"
	"github.com/spf13/cast"
	"github.com/stretchr/testify/assert"
)

// Test_NewRegistry tests the NewRegistry function.
func Test_NewRegistry(t *testing.T) {
	logger := zerolog.Logger{}

	actRegistry := NewActRegistry(
		BuiltinSignals(), BuiltinPolicies(), BuiltinActions(),
		config.DefaultPolicy, config.DefaultPolicyTimeout, logger)
	assert.NotNil(t, actRegistry)
	assert.NotNil(t, actRegistry.Signals)
	assert.NotNil(t, actRegistry.Policies)
	assert.NotNil(t, actRegistry.Actions)
	assert.Equal(t, config.DefaultPolicy, actRegistry.DefaultPolicy.Name)
	assert.Equal(t, config.DefaultPolicy, actRegistry.DefaultSignal.Name)
}

// Test_NewRegistry_NilSignals tests the NewRegistry function with nil signals,
// actions, and policies. It should return a nil registry.
func Test_NewRegistry_NilBuiltins(t *testing.T) {
	buf := bytes.Buffer{}
	logger := zerolog.New(&buf)
	actRegistry := NewActRegistry(
		nil, nil, nil, config.DefaultPolicy, config.DefaultPolicyTimeout, logger)
	assert.Nil(t, actRegistry)
	assert.Contains(t, buf.String(), "Builtin signals, policies, or actions are nil, not adding")
}

// Test_NewRegistry_NilPolicy tests the NewRegistry function with a nil signal.
// It should return a nil registry.
func Test_NewRegistry_NilSignal(t *testing.T) {
	buf := bytes.Buffer{}
	logger := zerolog.New(&buf)
	actRegistry := NewActRegistry(
		map[string]*sdkAct.Signal{
			"bad": nil,
		},
		BuiltinPolicies(), BuiltinActions(),
		config.DefaultPolicy, config.DefaultPolicyTimeout, logger)
	assert.Nil(t, actRegistry)
	assert.Contains(t, buf.String(), "Signal is nil, not adding")
}

// Test_NewRegistry_NilPolicy tests the NewRegistry function with a nil policy.
// It should return a nil registry.
func Test_NewRegistry_NilPolicy(t *testing.T) {
	buf := bytes.Buffer{}
	logger := zerolog.New(&buf)
	actRegistry := NewActRegistry(
		BuiltinSignals(),
		map[string]*sdkAct.Policy{
			"bad": nil,
		},
		BuiltinActions(),
		config.DefaultPolicy, config.DefaultPolicyTimeout, logger)
	assert.Nil(t, actRegistry)
	assert.Contains(t, buf.String(), "Policy is nil, not adding")
}

// Test_NewRegistry_NilAction tests the NewRegistry function with a nil action.
// It should return a nil registry.
func Test_NewRegistry_NilAction(t *testing.T) {
	buf := bytes.Buffer{}
	logger := zerolog.New(&buf)
	actRegistry := NewActRegistry(
		BuiltinSignals(), BuiltinPolicies(),
		map[string]*sdkAct.Action{
			"bad": nil,
		},
		config.DefaultPolicy, config.DefaultPolicyTimeout, logger)
	assert.Nil(t, actRegistry)
	assert.Contains(t, buf.String(), "Action is nil, not adding")
}

// Test_Add tests the Add function of the act registry.
func Test_Add(t *testing.T) {
	actRegistry := NewActRegistry(
		BuiltinSignals(), BuiltinPolicies(), BuiltinActions(),
		config.DefaultPolicy, config.DefaultPolicyTimeout, zerolog.Logger{})
	assert.NotNil(t, actRegistry)

	assert.Len(t, actRegistry.Policies, len(BuiltinPolicies()))
	actRegistry.Add(&sdkAct.Policy{Name: "test-policy", Policy: "true"})
	assert.NotNil(t, actRegistry.Policies["test-policy"])
	assert.Equal(t, "test-policy", actRegistry.Policies["test-policy"].Name)
	assert.Equal(t, "true", actRegistry.Policies["test-policy"].Policy)
	assert.Nil(t, actRegistry.Policies["test-policy"].Metadata)
	assert.Len(t, actRegistry.Policies, len(BuiltinPolicies())+1)
}

// Test_Add_NilPolicy tests the Add function of the act registry with a nil policy.
func Test_Add_NilPolicy(t *testing.T) {
	buf := bytes.Buffer{}
	logger := zerolog.New(&buf)
	actRegistry := NewActRegistry(
		BuiltinSignals(), map[string]*sdkAct.Policy{}, BuiltinActions(),
		config.DefaultPolicy, config.DefaultPolicyTimeout, logger)
	assert.NotNil(t, actRegistry)

	assert.Len(t, actRegistry.Policies, 0)
	actRegistry.Add(nil)
	assert.Len(t, actRegistry.Policies, 0)
	assert.Contains(t, buf.String(), "Policy is nil, not adding")
}

// Test_Add_ExistentPolicy tests the Add function of the act registry with an existent policy.
func Test_Add_ExistentPolicy(t *testing.T) {
	buf := bytes.Buffer{}
	logger := zerolog.New(&buf)
	actRegistry := NewActRegistry(
		BuiltinSignals(), BuiltinPolicies(), BuiltinActions(),
		config.DefaultPolicy, config.DefaultPolicyTimeout, logger)
	assert.NotNil(t, actRegistry)

	assert.Len(t, actRegistry.Policies, len(BuiltinPolicies()))
	actRegistry.Add(BuiltinPolicies()["passthrough"])
	assert.Len(t, actRegistry.Policies, len(BuiltinPolicies()))
	assert.Contains(t, buf.String(), "Policy already exists, overwriting")
}

// Test_Apply tests the Apply function of the act registry.
func Test_Apply(t *testing.T) {
	actRegistry := NewActRegistry(
		BuiltinSignals(), BuiltinPolicies(), BuiltinActions(),
		config.DefaultPolicy, config.DefaultPolicyTimeout, zerolog.Logger{})
	assert.NotNil(t, actRegistry)

	outputs := actRegistry.Apply([]sdkAct.Signal{
		*sdkAct.Passthrough(),
	})
	assert.NotNil(t, outputs)
	assert.Len(t, outputs, 1)
	assert.Equal(t, "passthrough", outputs[0].MatchedPolicy)
	assert.Nil(t, outputs[0].Metadata)
	assert.True(t, outputs[0].Sync)
	assert.True(t, cast.ToBool(outputs[0].Verdict))
	assert.False(t, outputs[0].Terminal)
}

// Test_Apply_NoSignals tests the Apply function of the act registry with no signals.
// It should apply the default policy.
func Test_Apply_NoSignals(t *testing.T) {
	buf := bytes.Buffer{}
	logger := zerolog.New(&buf)
	actRegistry := NewActRegistry(
		BuiltinSignals(), BuiltinPolicies(), BuiltinActions(),
		config.DefaultPolicy, config.DefaultPolicyTimeout, logger)
	assert.NotNil(t, actRegistry)

	outputs := actRegistry.Apply([]sdkAct.Signal{})
	assert.NotNil(t, outputs)
	assert.Len(t, outputs, 1)
	assert.Equal(t, "passthrough", outputs[0].MatchedPolicy)
	assert.Nil(t, outputs[0].Metadata)
	assert.True(t, outputs[0].Sync)
	assert.True(t, cast.ToBool(outputs[0].Verdict))
	assert.False(t, outputs[0].Terminal)
	assert.Contains(t, buf.String(), "No signals provided, applying default signal")
}

// Test_Apply_ContradictorySignals tests the Apply function of the act registry
// with contradictory signals. The terminate signal should take precedence over
// the passthrough signal because it is a terminal action. The passthrough
// signal should be ignored.
func Test_Apply_ContradictorySignals(t *testing.T) {
	// The following signals are contradictory because they have different actions.
	// The terminate signal should take precedence over the passthrough signal.
	// The order of the signals is NOT important.
	signals := [][]sdkAct.Signal{
		{
			*sdkAct.Terminate(),
			*sdkAct.Passthrough(),
			*sdkAct.Log("info", "test", map[string]any{"async": true}),
		},
		{
			*sdkAct.Passthrough(),
			*sdkAct.Terminate(),
			*sdkAct.Log("info", "test", map[string]any{"async": true}),
		},
	}

	buf := bytes.Buffer{}
	logger := zerolog.New(&buf)
	actRegistry := NewActRegistry(
		BuiltinSignals(), BuiltinPolicies(), BuiltinActions(),
		config.DefaultPolicy, config.DefaultPolicyTimeout, logger)
	assert.NotNil(t, actRegistry)

	for _, s := range signals {
		outputs := actRegistry.Apply(s)
		assert.NotNil(t, outputs)
		assert.Len(t, outputs, 2)
		assert.Equal(t, "terminate", outputs[0].MatchedPolicy)
		assert.Equal(t, outputs[0].Metadata, map[string]any{"terminate": true})
		assert.True(t, outputs[0].Sync)
		assert.True(t, cast.ToBool(outputs[0].Verdict))
		assert.True(t, outputs[0].Terminal)
		assert.Contains(
			t, buf.String(), "Terminal signal takes precedence, ignoring non-terminal signals")
		assert.Equal(t, "log", outputs[1].MatchedPolicy)
		assert.Equal(t,
			map[string]interface{}{
				"async":   true,
				"level":   "info",
				"log":     true,
				"message": "test",
			},
			outputs[1].Metadata,
		)
		assert.False(t, outputs[1].Sync)
		assert.True(t, cast.ToBool(outputs[1].Verdict))
		assert.False(t, outputs[1].Terminal)
	}
}

// Test_Apply_ActionNotMatched tests the Apply function of the act registry
// with a signal that does not match any action. The signal should be ignored.
// The default policy should be applied instead.
func Test_Apply_ActionNotMatched(t *testing.T) {
	buf := bytes.Buffer{}
	logger := zerolog.New(&buf)
	actRegistry := NewActRegistry(
		BuiltinSignals(), BuiltinPolicies(), BuiltinActions(),
		config.DefaultPolicy, config.DefaultPolicyTimeout, logger)
	assert.NotNil(t, actRegistry)

	outputs := actRegistry.Apply([]sdkAct.Signal{
		{Name: "non-existent"},
	})
	assert.NotNil(t, outputs)
	assert.Len(t, outputs, 1)
	assert.Equal(t, "passthrough", outputs[0].MatchedPolicy)
	assert.Nil(t, outputs[0].Metadata)
	assert.True(t, outputs[0].Sync)
	assert.True(t, cast.ToBool(outputs[0].Verdict))
	assert.False(t, outputs[0].Terminal)
	assert.Contains(t, buf.String(), "{\"level\":\"error\",\"error\":\"no matching action\",\"name\":\"non-existent\",\"message\":\"Error applying signal\"}") //nolint:lll
}

// Test_Apply_PolicyNotMatched tests the Apply function of the act registry
// with a signal that does not match any policy. The signal should be ignored.
// The default policy should be applied instead.
func Test_Apply_PolicyNotMatched(t *testing.T) {
	buf := bytes.Buffer{}
	logger := zerolog.New(&buf)
	actRegistry := NewActRegistry(
		BuiltinSignals(),
		map[string]*sdkAct.Policy{
			"passthrough": BuiltinPolicies()["passthrough"],
		},
		BuiltinActions(),
		config.DefaultPolicy, config.DefaultPolicyTimeout, logger)
	assert.NotNil(t, actRegistry)

	outputs := actRegistry.Apply([]sdkAct.Signal{
		*sdkAct.Terminate(),
	})
	assert.NotNil(t, outputs)
	assert.Len(t, outputs, 1)
	assert.Equal(t, "passthrough", outputs[0].MatchedPolicy)
	assert.Nil(t, outputs[0].Metadata)
	assert.True(t, outputs[0].Sync)
	assert.True(t, cast.ToBool(outputs[0].Verdict))
	assert.False(t, outputs[0].Terminal)
	assert.Contains(t, buf.String(), "{\"level\":\"error\",\"error\":\"no matching policy\",\"name\":\"terminate\",\"message\":\"Error applying signal\"}") //nolint:lll
}

// Test_Apply_NonBoolPolicy tests the Apply function of the act registry
// with a non-bool policy.
func Test_Apply_NonBoolPolicy(t *testing.T) {
	badPolicies := []map[string]*sdkAct.Policy{
		{
			"passthrough": sdkAct.MustNewPolicy(
				"passthrough",
				"2/0",
				nil,
			),
		},
		{
			"passthrough": sdkAct.MustNewPolicy(
				"passthrough",
				"2+2",
				nil,
			),
		},
	}

	for _, policies := range badPolicies {
		buf := bytes.Buffer{}
		logger := zerolog.New(&buf)
		actRegistry := NewActRegistry(
			BuiltinSignals(),
			policies,
			BuiltinActions(),
			config.DefaultPolicy, config.DefaultPolicyTimeout, logger)
		assert.NotNil(t, actRegistry)

		outputs := actRegistry.Apply([]sdkAct.Signal{
			*sdkAct.Passthrough(),
		})
		assert.NotNil(t, outputs)
		assert.Len(t, outputs, 1)
		assert.Equal(t, "passthrough", outputs[0].MatchedPolicy)
		assert.Nil(t, outputs[0].Metadata)
		assert.True(t, outputs[0].Sync)
		assert.NotNil(t, outputs[0].Verdict)
		assert.NotEmpty(t, outputs[0].Verdict)
	}
}

// Test_Apply_BadPolicy tests the NewRegistry function with a bad policy,
// which should return a nil registry.
func Test_Apply_BadPolicy(t *testing.T) {
	badPolicies := []map[string]*sdkAct.Policy{
		{
			"passthrough": sdkAct.MustNewPolicy(
				"passthrough",
				"2/0 + 'test'",
				nil,
			),
		},
		{
			"passthrough": sdkAct.MustNewPolicy(
				"passthrough",
				"2+2+true",
				nil,
			),
		},
	}

	for _, policies := range badPolicies {
		buf := bytes.Buffer{}
		logger := zerolog.New(&buf)
		actRegistry := NewActRegistry(
			BuiltinSignals(),
			policies,
			BuiltinActions(),
			config.DefaultPolicy, config.DefaultPolicyTimeout, logger)
		assert.Nil(t, actRegistry)
	}
}

// Test_Run tests the Run function of the act registry with a non-terminal action.
func Test_Run(t *testing.T) {
	logger := zerolog.Logger{}
	actRegistry := NewActRegistry(
		BuiltinSignals(), BuiltinPolicies(), BuiltinActions(),
		config.DefaultPolicy, config.DefaultPolicyTimeout, logger)
	assert.NotNil(t, actRegistry)

	outputs := actRegistry.Apply([]sdkAct.Signal{
		*sdkAct.Passthrough(),
	})
	assert.NotNil(t, outputs)

	result, err := actRegistry.Run(outputs[0], WithLogger(logger))
	assert.Nil(t, err)
	assert.True(t, cast.ToBool(result))
}

// Test_Run_Terminate tests the Run function of the act registry with a terminal action.
func Test_Run_Terminate(t *testing.T) {
	logger := zerolog.Logger{}
	actRegistry := NewActRegistry(
		BuiltinSignals(), BuiltinPolicies(), BuiltinActions(),
		config.DefaultPolicy, config.DefaultPolicyTimeout, logger)
	assert.NotNil(t, actRegistry)

	outputs := actRegistry.Apply([]sdkAct.Signal{
		*sdkAct.Terminate(),
	})
	assert.NotNil(t, outputs)
	assert.Equal(t, "terminate", outputs[0].MatchedPolicy)
	assert.Equal(t, outputs[0].Metadata, map[string]interface{}{"terminate": true})
	assert.True(t, outputs[0].Sync)
	assert.True(t, cast.ToBool(outputs[0].Verdict))
	assert.True(t, outputs[0].Terminal)

	result, err := actRegistry.Run(outputs[0], WithResult(map[string]any{}))
	assert.Nil(t, err)
	resultMap := cast.ToStringMap(result)
	assert.Contains(t, resultMap, "response")
	assert.NotEmpty(t, resultMap["response"])
}

// Test_Run_Async tests the Run function of the act registry with an asynchronous action.
func Test_Run_Async(t *testing.T) {
	out := bytes.Buffer{}
	logger := zerolog.New(&out)
	actRegistry := NewActRegistry(
		BuiltinSignals(), BuiltinPolicies(), BuiltinActions(),
		config.DefaultPolicy, config.DefaultPolicyTimeout, logger)
	assert.NotNil(t, actRegistry)

	outputs := actRegistry.Apply([]sdkAct.Signal{
		*sdkAct.Log("info", "test", map[string]any{"async": true}),
	})
	assert.NotNil(t, outputs)
	assert.Equal(t, "log", outputs[0].MatchedPolicy)
	assert.Equal(t,
		map[string]interface{}{
			"async":   true,
			"level":   "info",
			"log":     true,
			"message": "test",
		},
		outputs[0].Metadata,
	)
	assert.False(t, outputs[0].Sync)
	assert.True(t, cast.ToBool(outputs[0].Verdict))
	assert.False(t, outputs[0].Terminal)

	result, err := actRegistry.Run(outputs[0], WithResult(map[string]any{}))
	assert.Equal(t, err, gerr.ErrAsyncAction, "expected async action sentinel error")
	assert.Nil(t, result, "expected nil result")

	time.Sleep(time.Millisecond) // wait for async action to complete

	// The following is the expected log output from running the async action.
	assert.Contains(t, out.String(), "{\"level\":\"debug\",\"action\":\"log\",\"execution_mode\":\"async\",\"message\":\"Running action\"}") //nolint:lll
	// The following is the expected log output from the run function of the async action.
	assert.Contains(t, out.String(), "{\"level\":\"info\",\"async\":true,\"message\":\"test\"}")
}

// Test_Run_NilRegistry tests the Run function of the action with a nil output object.
func Test_Run_NilOutput(t *testing.T) {
	buf := bytes.Buffer{}
	logger := zerolog.New(&buf)
	actRegistry := NewActRegistry(
		BuiltinSignals(), BuiltinPolicies(), BuiltinActions(),
		config.DefaultPolicy, config.DefaultPolicyTimeout, logger)
	assert.NotNil(t, actRegistry)

	result, err := actRegistry.Run(nil, WithLogger(logger))
	assert.Nil(t, result)
	assert.Equal(t, err, gerr.ErrNilPointer)
	assert.Contains(t, buf.String(), "Output is nil, run aborted")
}

// Test_Run_ActionNotExist tests the Run function of the action with an empty output object.
func Test_Run_ActionNotExist(t *testing.T) {
	buf := bytes.Buffer{}
	logger := zerolog.New(&buf)
	actRegistry := NewActRegistry(
		BuiltinSignals(), BuiltinPolicies(), BuiltinActions(),
		config.DefaultPolicy, config.DefaultPolicyTimeout, logger)
	assert.NotNil(t, actRegistry)

	result, err := actRegistry.Run(&sdkAct.Output{}, WithLogger(logger))
	assert.Nil(t, result)
	assert.Equal(t, err, gerr.ErrActionNotExist)
	assert.Contains(t, buf.String(), "Action does not exist, run aborted")
}
