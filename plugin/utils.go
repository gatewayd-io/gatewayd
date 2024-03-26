package plugin

import (
	"os/exec"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"google.golang.org/protobuf/types/known/structpb"
)

// Verify compares two structs and returns true if they are equal.
func Verify(params, returnVal *structpb.Struct) bool {
	return cmp.Equal(params.AsMap(), returnVal.AsMap(), cmp.Options{
		cmpopts.SortMaps(func(a, b string) bool {
			return a < b
		}),
		cmpopts.EquateEmpty(),
	})
}

// NewCommand returns a command with the given arguments and environment variables.
func NewCommand(cmd exec.Cmd) *exec.Cmd {
	command := exec.Command(cmd.Path, cmd.Args...)
	if cmd.Env != nil {
		command.Env = append(command.Env, cmd.Env...)
	}
	return command
}

// CastToPrimitiveTypes casts the values of a map to its primitive type
// (e.g. time.Duration to float64) to prevent structpb invalid type(s) errors.
func CastToPrimitiveTypes(args map[string]interface{}) map[string]interface{} {
	for key, value := range args {
		switch value := value.(type) {
		case time.Duration:
			args[key] = value.String()
		case map[string]interface{}:
			// Recursively cast nested maps.
			args[key] = CastToPrimitiveTypes(value)
		// TODO: Add more types here as needed.
		default:
			args[key] = value
		}
	}
	return args
}
