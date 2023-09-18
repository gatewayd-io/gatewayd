package plugin

import (
	"os/exec"
	"time"

	v1 "github.com/gatewayd-io/gatewayd-plugin-sdk/plugin/v1"
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
)

// Verify compares two structs and returns true if they are equal.
func Verify(params, returnVal *v1.Struct) bool {
	return cmp.Equal(params.AsMap(), returnVal.AsMap(), cmp.Options{
		cmpopts.SortMaps(func(a, b string) bool {
			return a < b
		}),
		cmpopts.EquateEmpty(),
	})
}

// NewCommand returns a command with the given arguments and environment variables.
func NewCommand(cmd string, args []string, env []string) *exec.Cmd {
	command := exec.Command(cmd, args...)
	if env != nil {
		command.Env = append(command.Env, env...)
	}
	return command
}

// CastToPrimitiveTypes casts the values of a map to its primitive type
// (e.g. time.Duration to float64) to prevent structpb invalid type(s) errors.
func CastToPrimitiveTypes(args map[string]interface{}) map[string]interface{} {
	for key, value := range args {
		switch value := value.(type) {
		case time.Duration:
			// Cast time.Duration to string.
			args[key] = value.String()
		case map[string]interface{}:
			// Recursively cast nested maps.
			args[key] = CastToPrimitiveTypes(value)
		case []interface{}:
			// Recursively cast nested arrays.
			array := make([]interface{}, len(value))
			for idx, v := range value {
				result := v
				if v, ok := v.(time.Duration); ok {
					// Cast time.Duration to string.
					array[idx] = v.String()
				} else {
					array[idx] = result
				}
			}
			args[key] = array
		// TODO: Add more types here as needed.
		default:
			args[key] = value
		}
	}
	return args
}
