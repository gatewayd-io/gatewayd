package plugin

import (
	"context"
	"testing"
	"time"

	sdkPlugin "github.com/gatewayd-io/gatewayd-plugin-sdk/plugin"
	v1 "github.com/gatewayd-io/gatewayd-plugin-sdk/plugin/v1"
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
		Level:             zerolog.DebugLevel,
		NoColor:           true,
	}
	logger := logging.NewLogger(context.Background(), cfg)
	reg := NewRegistry(
		context.Background(),
		config.Loose,
		config.PassDown,
		config.Accept,
		config.Stop,
		logger,
		false,
	)
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
	reg.Verification = config.Ignore
	reg.AddHook(v1.HookName_HOOK_NAME_ON_NEW_LOGGER, 0, func(
		_ context.Context,
		args *v1.Struct,
		_ ...grpc.CallOption,
	) (*v1.Struct, error) {
		return args, nil
	})
	result, err := reg.Run(context.Background(), map[string]interface{}{}, v1.HookName_HOOK_NAME_ON_NEW_LOGGER)
	assert.NotNil(t, result)
	assert.Nil(t, err)
}

// Test_HookRegistry_Run_PassDown tests the Run function with the PassDown option.
func Test_PluginRegistry_Run_PassDown(t *testing.T) {
	reg := NewPluginRegistry(t)
	reg.Verification = config.PassDown
	// The result of the hook will be nil and will be passed down to the next
	reg.AddHook(v1.HookName_HOOK_NAME_ON_NEW_LOGGER, 0, func(
		_ context.Context,
		args *v1.Struct,
		_ ...grpc.CallOption,
	) (*v1.Struct, error) {
		return args, nil
	})
	// The consolidated result should be {"test": "test"}.
	reg.AddHook(v1.HookName_HOOK_NAME_ON_NEW_LOGGER, 1, func(
		context.Context,
		*v1.Struct,
		...grpc.CallOption,
	) (*v1.Struct, error) {
		output, err := v1.NewStruct(map[string]interface{}{
			"test": "test",
		})
		assert.Nil(t, err)
		return output, nil
	})

	// Although the first hook returns nil, and its signature doesn't match the params,
	// so its result (nil) is passed down to the next hook in chain (priority 2).
	// Then the second hook runs and returns a signature with a "test" key and value.
	result, err := reg.Run(
		context.Background(),
		map[string]interface{}{"test": "test"},
		v1.HookName_HOOK_NAME_ON_NEW_LOGGER)
	assert.Nil(t, err)
	assert.NotNil(t, result)
}

// Test_HookRegistry_Run_PassDown_2 tests the Run function with the PassDown option.
func Test_HookRegistry_Run_PassDown_2(t *testing.T) {
	reg := NewPluginRegistry(t)
	reg.Verification = config.PassDown
	// The result of the hook will be nil and will be passed down to the next
	reg.AddHook(v1.HookName_HOOK_NAME_ON_NEW_LOGGER, 0, func(
		_ context.Context,
		args *v1.Struct,
		_ ...grpc.CallOption,
	) (*v1.Struct, error) {
		args.Fields["test1"] = &v1.Value{
			Kind: &v1.Value_StringValue{
				StringValue: "test1",
			},
		}
		return args, nil
	})
	// The consolidated result should be {"test1": "test1", "test2": "test2"}.
	reg.AddHook(v1.HookName_HOOK_NAME_ON_NEW_LOGGER, 1, func(
		_ context.Context,
		args *v1.Struct,
		_ ...grpc.CallOption,
	) (*v1.Struct, error) {
		args.Fields["test2"] = &v1.Value{
			Kind: &v1.Value_StringValue{
				StringValue: "test2",
			},
		}
		return args, nil
	})
	// Although the first hook returns nil, and its signature doesn't match the params,
	// so its result (nil) is passed down to the next hook in chain (priority 2).
	// Then the second hook runs and returns a signature with a "test1" and "test2" key and value.
	result, err := reg.Run(
		context.Background(),
		map[string]interface{}{"test": "test"},
		v1.HookName_HOOK_NAME_ON_NEW_LOGGER)
	assert.Nil(t, err)
	assert.NotNil(t, result)
}

// Test_HookRegistry_Run_Ignore tests the Run function with the Ignore option.
func Test_HookRegistry_Run_Ignore(t *testing.T) {
	reg := NewPluginRegistry(t)
	reg.Verification = config.Ignore
	// This should not run, because the return value is not the same as the params
	reg.AddHook(v1.HookName_HOOK_NAME_ON_NEW_LOGGER, 0, func(
		_ context.Context,
		args *v1.Struct,
		_ ...grpc.CallOption,
	) (*v1.Struct, error) {
		return args, nil
	})
	// This should run, because the return value is the same as the params
	reg.AddHook(v1.HookName_HOOK_NAME_ON_NEW_LOGGER, 1, func(
		_ context.Context,
		args *v1.Struct,
		_ ...grpc.CallOption,
	) (*v1.Struct, error) {
		args.Fields["test"] = &v1.Value{
			Kind: &v1.Value_StringValue{
				StringValue: "test",
			},
		}
		return args, nil
	})
	// The first hook returns nil, and its signature doesn't match the params,
	// so its result is ignored.
	// Then the second hook runs and returns a signature with a "test" key and value.
	result, err := reg.Run(
		context.Background(),
		map[string]interface{}{"test": "test"},
		v1.HookName_HOOK_NAME_ON_NEW_LOGGER)
	assert.Nil(t, err)
	assert.NotNil(t, result)
}

// Test_HookRegistry_Run_Abort tests the Run function with the Abort option.
func Test_HookRegistry_Run_Abort(t *testing.T) {
	reg := NewPluginRegistry(t)
	reg.Verification = config.Abort
	// This should not run, because the return value is not the same as the params
	reg.AddHook(v1.HookName_HOOK_NAME_ON_NEW_LOGGER, 0, func(
		_ context.Context,
		args *v1.Struct,
		_ ...grpc.CallOption,
	) (*v1.Struct, error) {
		return args, nil
	})
	// This should not run, because the first hook returns nil, and its result is ignored.
	reg.AddHook(v1.HookName_HOOK_NAME_ON_NEW_LOGGER, 1, func(
		context.Context,
		*v1.Struct,
		...grpc.CallOption,
	) (*v1.Struct, error) {
		output, err := v1.NewStruct(map[string]interface{}{
			"test": "test",
		})
		assert.Nil(t, err)
		return output, nil
	})
	// The first hook returns nil, and it aborts the execution of the rest of the
	result, err := reg.Run(context.Background(), map[string]interface{}{}, v1.HookName_HOOK_NAME_ON_NEW_LOGGER)
	assert.Nil(t, err)
	assert.Equal(t, map[string]interface{}{}, result)
}

// Test_HookRegistry_Run_Remove tests the Run function with the Remove option.
func Test_HookRegistry_Run_Remove(t *testing.T) {
	reg := NewPluginRegistry(t)
	reg.Verification = config.Remove
	// This should not run, because the return value is not the same as the params
	reg.AddHook(v1.HookName_HOOK_NAME_ON_NEW_LOGGER, 0, func(
		_ context.Context,
		args *v1.Struct,
		_ ...grpc.CallOption,
	) (*v1.Struct, error) {
		return args, nil
	})
	// This should not run, because the first hook returns nil, and its result is ignored.
	reg.AddHook(v1.HookName_HOOK_NAME_ON_NEW_LOGGER, 1, func(
		context.Context,
		*v1.Struct,
		...grpc.CallOption,
	) (*v1.Struct, error) {
		output, err := v1.NewStruct(map[string]interface{}{
			"test": "test",
		})
		assert.Nil(t, err)
		return output, nil
	})
	// The first hook returns nil, and its signature doesn't match the params,
	// so its result is ignored. The failing hook is removed from the list and
	// the execution continues with the next hook in the list.
	result, err := reg.Run(context.Background(), map[string]interface{}{}, v1.HookName_HOOK_NAME_ON_NEW_LOGGER)
	assert.Nil(t, err)
	assert.Equal(t, map[string]interface{}{}, result)
	assert.Len(t, reg.Hooks()[v1.HookName_HOOK_NAME_ON_NEW_LOGGER], 1)
}

func BenchmarkHookRun(b *testing.B) {
	cfg := logging.LoggerConfig{
		Output:            []config.LogOutput{config.Console},
		TimeFormat:        zerolog.TimeFormatUnix,
		ConsoleTimeFormat: time.RFC3339,
		Level:             zerolog.DebugLevel,
		NoColor:           true,
	}
	logger := logging.NewLogger(context.Background(), cfg)
	reg := NewRegistry(
		context.Background(),
		config.Loose,
		config.PassDown,
		config.Accept,
		config.Stop,
		logger,
		false,
	)
	reg.Verification = config.PassDown
	hookFunction := func(
		_ context.Context, args *v1.Struct, _ ...grpc.CallOption,
	) (*v1.Struct, error) {
		args.Fields["test"] = v1.NewStringValue("test1")
		return args, nil
	}
	for priority := 0; priority < 1000; priority++ {
		reg.AddHook(v1.HookName_HOOK_NAME_ON_NEW_LOGGER,
			sdkPlugin.Priority(priority),
			hookFunction,
		)
	}
	for i := 0; i < b.N; i++ {
		//nolint:errcheck
		reg.Run(
			context.Background(),
			map[string]interface{}{
				"test": "test",
			},
			v1.HookName_HOOK_NAME_ON_NEW_LOGGER,
		)
	}
}
