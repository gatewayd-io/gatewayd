package network

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_NewHookConfig(t *testing.T) {
	hc := NewHookConfig()
	assert.NotNil(t, hc)
}

func Test_HookConfig_Add(t *testing.T) {
	hooks := NewHookConfig()
	testFunc := func(s Signature) Signature {
		return s
	}
	hooks.Add(OnNewLogger, 0, testFunc)
	assert.NotNil(t, hooks.hooks[OnNewLogger][0])
	assert.ObjectsAreEqual(testFunc, hooks.hooks[OnNewLogger][0])
}

func Test_HookConfig_Add_Multiple_Hooks(t *testing.T) {
	hooks := NewHookConfig()
	hooks.Add(OnNewLogger, 0, func(s Signature) Signature {
		return s
	})
	hooks.Add(OnNewLogger, 1, func(s Signature) Signature {
		return s
	})
	assert.NotNil(t, hooks.hooks[OnNewLogger][0])
	assert.NotNil(t, hooks.hooks[OnNewLogger][1])
}

func Test_HookConfig_Get(t *testing.T) {
	hooks := NewHookConfig()
	testFunc := func(s Signature) Signature {
		return s
	}
	prio := Prio(0)
	hooks.Add(OnNewLogger, prio, testFunc)
	assert.NotNil(t, hooks.Get(OnNewLogger))
	assert.ObjectsAreEqual(testFunc, hooks.Get(OnNewLogger)[prio])
}

func Test_HookConfig_Run(t *testing.T) {
	hooks := NewHookConfig()
	hooks.Add(OnNewLogger, 0, func(s Signature) Signature {
		return s
	})
	assert.NotNil(t, hooks.Run(OnNewLogger, Signature{}, Ignore))
}

func Test_verify(t *testing.T) {
	params := Signature{
		"test": "test",
	}
	returnVal := Signature{
		"test": "test",
	}
	assert.True(t, verify(params, returnVal))
}

func Test_verify_fail(t *testing.T) {
	data := [][]Signature{
		{
			{
				"test": "test",
			},
			{
				"test":  "test",
				"test2": "test2",
			},
		},
		{
			{
				"test":  "test",
				"test2": "test2",
			},
			{
				"test": "test",
			},
		},
		{
			{
				"test":  "test",
				"test2": "test2",
			},
			{
				"test":  "test",
				"test3": "test3",
			},
		},
	}

	for _, d := range data {
		assert.False(t, verify(d[0], d[1]))
	}
}

func Test_HookConfig_Run_Ignore(t *testing.T) {
	hooks := NewHookConfig()
	// This should not run, because the return value is not the same as the params
	hooks.Add(OnNewLogger, 0, func(s Signature) Signature {
		return nil
	})
	// This should run, because the return value is the same as the params
	hooks.Add(OnNewLogger, 1, func(s Signature) Signature {
		return Signature{
			"test": "test",
		}
	})
	// The first hook returns nil, and its Signature doesn't match the params,
	// so its result is ignored.
	// Then the second hook runs and returns a Signature with a "test" key and value.
	assert.NotNil(t, hooks.Run(OnNewLogger, Signature{"test": "test"}, Ignore))
}

func Test_HookConfig_Run_Abort(t *testing.T) {
	hooks := NewHookConfig()
	// This should not run, because the return value is not the same as the params
	hooks.Add(OnNewLogger, 0, func(s Signature) Signature {
		return nil
	})
	// This should not run, because the first hook returns nil, and its result is ignored.
	hooks.Add(OnNewLogger, 1, func(s Signature) Signature {
		return Signature{
			"test": "test",
		}
	})
	// The first hook returns nil, and it aborts the execution of the rest of the hook.
	assert.Nil(t, hooks.Run(OnNewLogger, nil, Abort))
}

func Test_HookConfig_Run_Remove(t *testing.T) {
	hooks := NewHookConfig()
	// This should not run, because the return value is not the same as the params
	hooks.Add(OnNewLogger, 0, func(s Signature) Signature {
		return nil
	})
	// This should not run, because the first hook returns nil, and its result is ignored.
	hooks.Add(OnNewLogger, 1, func(s Signature) Signature {
		return Signature{
			"test": "test",
		}
	})
	// The first hook returns nil, and its Signature doesn't match the params,
	// so its result is ignored. The failing hook is removed from the list and
	// the execution continues with the next hook in the list.
	assert.Nil(t, hooks.Run(OnNewLogger, nil, Remove))
	assert.Equal(t, 1, len(hooks.hooks[OnNewLogger]))
}
