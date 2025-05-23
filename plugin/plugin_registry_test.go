package plugin

import (
	"context"
	"testing"
	"time"

	sdkPlugin "github.com/gatewayd-io/gatewayd-plugin-sdk/plugin"
	v1 "github.com/gatewayd-io/gatewayd-plugin-sdk/plugin/v1"
	"github.com/gatewayd-io/gatewayd/act"
	"github.com/gatewayd-io/gatewayd/config"
	"github.com/gatewayd-io/gatewayd/logging"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc"
)

func NewPluginRegistry(t *testing.T) *Registry {
	t.Helper()

	cfg := logging.LoggerConfig{
		Output:            []config.LogOutput{config.Console},
		TimeFormat:        zerolog.TimeFormatUnix,
		ConsoleTimeFormat: time.RFC3339,
		Level:             zerolog.InfoLevel,
		NoColor:           true,
	}
	logger := logging.NewLogger(t.Context(), cfg)
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
	reg := NewRegistry(
		t.Context(),
		Registry{
			ActRegistry: actRegistry,
			Logger:      logger,
		})
	return reg
}

// TestPluginRegistry tests the PluginRegistry.
func TestPluginRegistry(t *testing.T) {
	reg := NewPluginRegistry(t)
	assert.NotNil(t, reg)
	assert.NotNil(t, reg.plugins)
	assert.NotNil(t, reg.hooks)
	assert.Empty(t, reg.List())

	ident := sdkPlugin.Identifier{
		Name:      "test",
		Version:   "1.0.0",
		RemoteURL: "github.com/remote/test",
	}
	impl := &Plugin{
		ID: ident,
	}
	reg.Add(impl)
	assert.Len(t, reg.List(), 1)

	instance := reg.Get(ident)
	assert.Equal(t, instance, impl)

	assert.Equal(t, reg.Size(), 1)
	assert.True(t, reg.Exists(ident.Name, ident.Version, ident.RemoteURL))

	reg.ForEach(func(i sdkPlugin.Identifier, p *Plugin) {
		assert.Equal(t, i, ident)
		assert.Equal(t, p, impl)
	})

	reg.Remove(ident)
	assert.Empty(t, reg.List())

	reg.Shutdown()
}

// Test_HookRegistry_Add tests the Add function.
func Test_PluginRegistry_AddHook(t *testing.T) {
	testFunc := func(
		_ context.Context,
		args *v1.Struct,
		_ ...grpc.CallOption,
	) (*v1.Struct, error) {
		return args, nil
	}

	reg := NewPluginRegistry(t)
	reg.AddHook(v1.HookName_HOOK_NAME_ON_NEW_LOGGER, 0, testFunc)
	assert.NotNil(t, reg.Hooks()[v1.HookName_HOOK_NAME_ON_NEW_LOGGER][0])
	assert.ObjectsAreEqual(testFunc, reg.Hooks()[v1.HookName_HOOK_NAME_ON_NEW_LOGGER][0])
}

// Test_HookRegistry_Add_Multiple_Hooks tests the Add function with multiple hooks.
func Test_PluginRegistry_AddHook_Multiple(t *testing.T) {
	reg := NewPluginRegistry(t)
	reg.AddHook(v1.HookName_HOOK_NAME_ON_NEW_LOGGER, 0, func(
		_ context.Context,
		args *v1.Struct,
		_ ...grpc.CallOption,
	) (*v1.Struct, error) {
		return args, nil
	})
	reg.AddHook(v1.HookName_HOOK_NAME_ON_NEW_LOGGER, 1, func(
		_ context.Context,
		args *v1.Struct,
		_ ...grpc.CallOption,
	) (*v1.Struct, error) {
		return args, nil
	})
	assert.NotNil(t, reg.Hooks()[v1.HookName_HOOK_NAME_ON_NEW_LOGGER][0])
	assert.NotNil(t, reg.Hooks()[v1.HookName_HOOK_NAME_ON_NEW_LOGGER][1])
}

// Test_HookRegistry_Run tests the Run function.
func Test_PluginRegistry_Run(t *testing.T) {
	reg := NewPluginRegistry(t)
	reg.AddHook(v1.HookName_HOOK_NAME_ON_NEW_LOGGER, 0, func(
		_ context.Context,
		args *v1.Struct,
		_ ...grpc.CallOption,
	) (*v1.Struct, error) {
		return args, nil
	})
	result, err := reg.Run(t.Context(), map[string]any{}, v1.HookName_HOOK_NAME_ON_NEW_LOGGER)
	assert.NotNil(t, result)
	assert.Nil(t, err)
}

func BenchmarkHookRun(b *testing.B) {
	cfg := logging.LoggerConfig{
		Output:            []config.LogOutput{config.Console},
		TimeFormat:        zerolog.TimeFormatUnix,
		ConsoleTimeFormat: time.RFC3339,
		Level:             zerolog.DebugLevel,
		NoColor:           true,
	}
	logger := logging.NewLogger(b.Context(), cfg)
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

	reg := NewRegistry(
		b.Context(),
		Registry{
			ActRegistry: actRegistry,
			Logger:      logger,
		})
	hookFunction := func(
		_ context.Context, args *v1.Struct, _ ...grpc.CallOption,
	) (*v1.Struct, error) {
		args.Fields["test"] = v1.NewStringValue("test1")
		return args, nil
	}
	for priority := range uint(1000) {
		reg.AddHook(v1.HookName_HOOK_NAME_ON_NEW_LOGGER,
			sdkPlugin.Priority(priority),
			hookFunction,
		)
	}
	for range b.N {
		//nolint:errcheck
		reg.Run(
			b.Context(),
			map[string]any{
				"test": "test",
			},
			v1.HookName_HOOK_NAME_ON_NEW_LOGGER,
		)
	}
}
