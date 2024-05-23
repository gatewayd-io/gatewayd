package plugin

import (
	"testing"
	"time"

	sdkAct "github.com/gatewayd-io/gatewayd-plugin-sdk/act"
	"github.com/gatewayd-io/gatewayd/act"
	"github.com/gatewayd-io/gatewayd/config"
	"github.com/rs/zerolog"
	"github.com/spf13/cast"
	"github.com/stretchr/testify/assert"
)

func Test_NewCommand(t *testing.T) {
	cmd := NewCommand("/test", []string{"--test"}, []string{"test=123"})
	assert.NotNil(t, cmd)
	assert.Equal(t, "/test", cmd.Path)
	// Command.Args[0] is always set to the command name itself.
	assert.Equal(t, []string{"/test", "--test"}, cmd.Args)
	assert.Equal(t, []string{"test=123"}, cmd.Env)
}

// Test_castToPrimitiveTypes tests the CastToPrimitiveTypes function.
func Test_castToPrimitiveTypes(t *testing.T) {
	actual := map[string]interface{}{
		"string":   "test",
		"int":      123,
		"bool":     true,
		"map":      map[string]interface{}{"test": "test"},
		"duration": time.Duration(123),
		"array": []interface{}{
			"test",
			123,
			true,
			map[string]interface{}{
				"test": "test",
			},
			time.Duration(123),
		},
	}
	expected := map[string]interface{}{
		"string":   "test",
		"int":      123,
		"bool":     true,
		"map":      map[string]interface{}{"test": "test"},
		"duration": "123ns", // time.Duration is cast to string.
		"array": []interface{}{
			"test",
			123,
			true,
			map[string]interface{}{
				"test": "test",
			},
			"123ns", // time.Duration is cast to string.
		},
	}

	casted := castToPrimitiveTypes(actual)
	assert.Equal(t, expected, casted)
}

// Test_getSignals tests the getSignals function.
func Test_getSignals(t *testing.T) {
	result := map[string]interface{}{
		sdkAct.Signals: []any{
			(&sdkAct.Signal{
				Name:     "test",
				Metadata: map[string]any{"test": "test"},
			}).ToMap(),
			sdkAct.Passthrough().ToMap(),
		},
	}
	signals := getSignals(result)
	assert.Len(t, signals, 2)
	assert.Equal(t, "test", signals[0].Name)
	assert.Equal(t, "test", signals[0].Metadata["test"])
	assert.Equal(t, "passthrough", signals[1].Name)
	assert.Nil(t, signals[1].Metadata)
}

// Test_getSignals_empty tests the getSignals function with an empty result.
func Test_getSignals_empty(t *testing.T) {
	result := map[string]interface{}{}
	signals := getSignals(result)
	assert.Len(t, signals, 0)
}

// Test_applyPolicies tests the applyPolicies function with a passthrough policy.
// It also tests the Run function of the registered passthrough (built-in) action.
func Test_applyPolicies(t *testing.T) {
	logger := zerolog.Logger{}
	actRegistry := act.NewActRegistry(
		act.Registry{
			Signals:              act.BuiltinSignals(),
			Policies:             act.BuiltinPolicies(),
			Actions:              act.BuiltinActions(),
			DefaultPolicyName:    config.DefaultPolicy,
			PolicyTimeout:        config.DefaultPolicyTimeout,
			DefaultActionTimeout: config.DefaultActionTimeout,
			Logger:               logger,
		})

	output := applyPolicies(
		sdkAct.Hook{
			Name:     "HOOK_NAME_ON_TRAFFIC_FROM_CLIENT",
			Priority: 1000,
			Params:   map[string]any{},
			Result:   map[string]any{},
		},
		[]sdkAct.Signal{*sdkAct.Passthrough()},
		logger,
		actRegistry,
	)
	assert.Len(t, output, 1)
	assert.Equal(t, "passthrough", output[0].MatchedPolicy)
	assert.Nil(t, output[0].Metadata)
	assert.True(t, output[0].Sync)
	assert.False(t, output[0].Terminal)
	assert.True(t, cast.ToBool(output[0].Verdict))

	result, gerr := actRegistry.Run(output[0], act.WithLogger(logger))
	assert.Nil(t, gerr)
	assert.True(t, cast.ToBool(result))
}
