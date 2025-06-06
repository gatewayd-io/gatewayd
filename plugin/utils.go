package plugin

import (
	"fmt"
	"os/exec"
	"strings"
	"time"

	sdkAct "github.com/gatewayd-io/gatewayd-plugin-sdk/act"
	"github.com/gatewayd-io/gatewayd/act"
	"github.com/gatewayd-io/gatewayd/config"
	"github.com/rs/zerolog"
	"github.com/spf13/cast"
)

// NewCommand returns a command with the given arguments and environment variables.
func NewCommand(cmd string, args []string, env []string) *exec.Cmd {
	command := exec.Command(cmd, args...)
	if env != nil {
		command.Env = append(command.Env, env...)
	}
	return command
}

// castToPrimitiveTypes casts the values of a map to its primitive type
// (e.g. time.Duration to float64) to prevent structpb invalid type(s) errors.
func castToPrimitiveTypes(args map[string]any) map[string]any {
	for key, value := range args {
		switch value := value.(type) {
		case time.Duration:
			// Cast time.Duration to string.
			args[key] = value.String()
		case map[string]any:
			// Recursively cast nested maps.
			args[key] = castToPrimitiveTypes(value)
		case []any:
			// Recursively cast nested arrays.
			array := make([]any, len(value))
			for idx, v := range value {
				if durVal, ok := v.(time.Duration); ok {
					// Cast time.Duration to string.
					array[idx] = durVal.String()
				} else {
					array[idx] = v
				}
			}
			args[key] = array
		case map[string]map[string]any:
			for _, valuemap := range value {
				// Recursively cast nested maps.
				args[key] = castToPrimitiveTypes(valuemap)
			}
		case []config.RaftPeer:
			// Cast raft peers to comma-separated string using strings.Join
			peers := make([]string, len(value))
			for i, peer := range value {
				peers[i] = fmt.Sprintf("%s:%s:%s", peer.ID, peer.Address, peer.GRPCAddress)
			}
			args[key] = strings.Join(peers, ",")
		// TODO: Add more types here as needed.
		default:
			args[key] = value
		}
	}
	return args
}

// getSignals decodes the signals from the result map and returns them as a list of Signal objects.
func getSignals(result map[string]any) []sdkAct.Signal {
	var decodedSignals []sdkAct.Signal

	if signals, ok := result[sdkAct.Signals]; ok {
		signals := cast.ToSlice(signals)
		for _, signal := range signals {
			signalMap := cast.ToStringMap(signal)
			name := cast.ToString(signalMap[sdkAct.Name])
			metadata := cast.ToStringMap(signalMap[sdkAct.Metadata])

			if name != "" {
				// Add the signal to the list of signals.
				decodedSignals = append(decodedSignals, sdkAct.Signal{
					Name:     name,
					Metadata: metadata,
				})
			}
		}
	}

	return decodedSignals
}

// applyPolicies applies the policies to the signals and returns the outputs.
func applyPolicies(
	hook sdkAct.Hook,
	signals []sdkAct.Signal,
	logger zerolog.Logger,
	reg act.IRegistry,
) []*sdkAct.Output {
	signalNames := []string{}
	for _, signal := range signals {
		signalNames = append(signalNames, signal.Name)
	}

	logger.Debug().Fields(
		map[string]any{
			"hook":    hook.Name,
			"signals": signalNames,
		},
	).Msg("Detected signals from the plugin hook")

	outputs := reg.Apply(signals, hook)
	logger.Debug().Fields(
		map[string]any{
			"hook":    hook.Name,
			"outputs": outputs,
		},
	).Msg("Applied policies to signals")

	return outputs
}
