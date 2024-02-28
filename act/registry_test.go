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

	actRegistry := NewRegistry(
		BuiltinSignals(), BuiltinPolicies(), BuiltinActions(),
		config.DefaultPolicy, config.DefaultPolicyTimeout, logger)
	assert.NotNil(t, actRegistry)
	assert.NotNil(t, actRegistry.Signals)
	assert.NotNil(t, actRegistry.Policies)
	assert.NotNil(t, actRegistry.Actions)
	assert.Equal(t, config.DefaultPolicy, actRegistry.DefaultPolicy.Name)
	assert.Equal(t, config.DefaultPolicy, actRegistry.DefaultSignal.Name)
}

// Test_Add tests the Add function of the policy registry.
func Test_Add(t *testing.T) {
	actRegistry := NewRegistry(
		BuiltinSignals(), BuiltinPolicies(), BuiltinActions(),
		config.DefaultPolicy, config.DefaultPolicyTimeout, zerolog.Logger{})
	assert.NotNil(t, actRegistry)

	actRegistry.Add(&sdkAct.Policy{Name: "test-policy", Policy: "true"})
	assert.NotNil(t, actRegistry.Policies["test-policy"])
	assert.Equal(t, "test-policy", actRegistry.Policies["test-policy"].Name)
	assert.Equal(t, "true", actRegistry.Policies["test-policy"].Policy)
	assert.Nil(t, actRegistry.Policies["test-policy"].Metadata)
}

// Test_Apply tests the Apply function of the policy registry.
func Test_Apply(t *testing.T) {
	actRegistry := NewRegistry(
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
	assert.True(t, outputs[0].Verdict)
	assert.False(t, outputs[0].Terminal)
}

// Test_Run tests the Run function of the policy registry with a non-terminal action.
func Test_Run(t *testing.T) {
	logger := zerolog.Logger{}
	actRegistry := NewRegistry(
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

// Test_Run_Terminate tests the Run function of the policy registry with a terminal action.
func Test_Run_Terminate(t *testing.T) {
	logger := zerolog.Logger{}
	actRegistry := NewRegistry(
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
	assert.True(t, outputs[0].Verdict)
	assert.True(t, outputs[0].Terminal)

	result, err := actRegistry.Run(outputs[0], WithResult(map[string]any{}))
	assert.Nil(t, err)
	resultMap := cast.ToStringMap(result)
	assert.Contains(t, resultMap, "response")
	assert.NotEmpty(t, resultMap["response"])
}

// Test_Run_Async tests the Run function of the policy registry with an asynchronous action.
func Test_Run_Async(t *testing.T) {
	out := bytes.Buffer{}
	logger := zerolog.New(&out)
	actRegistry := NewRegistry(
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
	assert.True(t, outputs[0].Verdict)
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
