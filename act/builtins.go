package act

import (
	sdkAct "github.com/gatewayd-io/gatewayd-plugin-sdk/act"
	"github.com/gatewayd-io/gatewayd-plugin-sdk/databases/postgres"
	"github.com/gatewayd-io/gatewayd-plugin-sdk/logging"
	gerr "github.com/gatewayd-io/gatewayd/errors"
	"github.com/jackc/pgx/v5/pgproto3"
	"github.com/rs/zerolog"
	"github.com/spf13/cast"
)

const (
	// TerminateDefaultParamCount is the default parameter count for the terminate action.
	TerminateDefaultParamCount = 2

	// LogDefaultKeyCount is the default key count in the metadata for the log action.
	LogDefaultKeyCount = 3

	// These are the keys used to pass the logger and the result to the built-in actions.
	LoggerKey = "__logger__"
	ResultKey = "__result__"
)

// BuiltinSignals returns a map of built-in signals.
func BuiltinSignals() map[string]*sdkAct.Signal {
	return map[string]*sdkAct.Signal{
		"passthrough": sdkAct.Passthrough(),
		"terminate":   sdkAct.Terminate(),
		"log":         {Name: "log"},
	}
}

// BuiltinPolicies returns a map of built-in policies.
func BuiltinPolicies() map[string]*sdkAct.Policy {
	return map[string]*sdkAct.Policy{
		"passthrough": sdkAct.MustNewPolicy("passthrough", "true", nil),
		"terminate": sdkAct.MustNewPolicy(
			"terminate",
			`Signal.terminate == true && Policy.terminate == "stop"`,
			map[string]any{"terminate": "stop"},
		),
		"log": sdkAct.MustNewPolicy(
			"log",
			`Signal.log == true && Policy.log == "enabled"`,
			map[string]any{"log": "enabled"},
		),
	}
}

// BuiltinActions returns a map of built-in actions.
func BuiltinActions() map[string]*sdkAct.Action {
	return map[string]*sdkAct.Action{
		"passthrough": {
			Name:     "passthrough",
			Metadata: nil,
			Sync:     true,
			Terminal: false,
			Run:      Passthrough,
		},
		"terminate": {
			Name:     "terminate",
			Metadata: nil,
			Sync:     true,
			Terminal: true,
			Run:      Terminate,
		},
		"log": {
			Name:     "log",
			Metadata: nil,
			Sync:     false,
			Terminal: false,
			Run:      Log,
		},
	}
}

// Passthrough is a built-in action that always returns true and no error.
func Passthrough(map[string]any, ...sdkAct.Parameter) (any, error) {
	return true, nil
}

// Terminate is a built-in action that terminates the connection if the
// terminate signal is true and the policy is set to "stop". The action
// can optionally receive a result parameter.
func Terminate(_ map[string]any, params ...sdkAct.Parameter) (any, error) {
	if len(params) == 0 || params[0].Key != LoggerKey {
		// No logger parameter or the first parameter is not a logger.
		return nil, gerr.ErrLoggerRequired
	}

	logger, isValid := params[0].Value.(zerolog.Logger)
	if !isValid {
		// The first parameter is not a logger.
		return nil, gerr.ErrLoggerRequired
	}

	if len(params) < TerminateDefaultParamCount || params[1].Key != ResultKey {
		logger.Debug().Msg(
			"terminate action can optionally receive a result parameter")
		return true, nil
	}

	result, isValid := params[1].Value.(map[string]any)
	if !isValid {
		logger.Debug().Msg("terminate action received a non-map result parameter")
		return true, nil
	}

	// If the result from the plugin does not contain a response,
	// yet it is a terminal action (hence running this action),
	// add an error response to the result and terminate the connection.
	if _, exists := result["response"]; !exists {
		logger.Trace().Fields(result).Msg(
			"Terminating without response, returning an error response")
		response, err := (&pgproto3.Terminate{}).Encode(
			postgres.ErrorResponse(
				"Request terminated",
				"ERROR",
				"42000",
				"Policy terminated the request",
			),
		)
		if err != nil {
			// This should never happen, since everything is hardcoded.
			logger.Error().Err(err).Msg("Failed to encode the error response")
			return nil, gerr.ErrMsgEncodeError.Wrap(err)
		}
		result["response"] = response
	}

	return result, nil
}

// Log is a built-in action that logs the data received from the plugin.
func Log(data map[string]any, params ...sdkAct.Parameter) (any, error) {
	if len(params) == 0 || params[0].Key != LoggerKey {
		// No logger parameter or the first parameter is not a logger.
		return nil, gerr.ErrLoggerRequired
	}

	logger, ok := params[0].Value.(zerolog.Logger)
	if !ok {
		// The first parameter is not a logger.
		return nil, gerr.ErrLoggerRequired
	}

	fields := map[string]any{}
	if len(data) > LogDefaultKeyCount {
		for key, value := range data {
			// Skip these necessary fields, as they are already used by the logger.
			// level: The log level.
			// message: The log message.
			// log: The log signal.
			if key == "level" || key == "message" || key == "log" {
				continue
			}
			// Add the rest of the fields to the logger as extra fields.
			fields[key] = value
		}
	}

	logger.WithLevel(
		logging.GetZeroLogLevel(cast.ToString(data["level"])),
	).Fields(fields).Msg(cast.ToString(data["message"]))

	return true, nil
}
