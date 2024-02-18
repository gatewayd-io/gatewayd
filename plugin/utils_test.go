package plugin

import (
	"testing"
	"time"

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

// Test_CastToPrimitiveTypes tests the CastToPrimitiveTypes function.
func Test_CastToPrimitiveTypes(t *testing.T) {
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
		"duration": "123ns", // time.Duration is casted to string.
		"array": []interface{}{
			"test",
			123,
			true,
			map[string]interface{}{
				"test": "test",
			},
			"123ns", // time.Duration is casted to string.
		},
	}

	casted := CastToPrimitiveTypes(actual)
	assert.Equal(t, expected, casted)
}
