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
	LogDefaultFieldCount       = 3
	TerminateDefaultFieldCount = 2
)

// TODO: Should we get rid of global variables?
var (
	builtinSignals = []*sdkAct.Signal{
		sdkAct.Passthrough(),
		sdkAct.Terminate(),
	}

	builtinsPolicies = []*sdkAct.Policy{
		sdkAct.MustNewPolicy("passthrough", "true", nil),
		sdkAct.MustNewPolicy(
			"terminate",
			`Signal.terminate == true && Policy.terminate == "stop"`,
			map[string]any{"terminate": "stop"},
		),
		sdkAct.MustNewPolicy(
			"log",
			`Signal.log == true && Policy.log == "enabled"`,
			map[string]any{"log": "enabled"},
		),
	}

	builtinActions = []*sdkAct.Action{
		{
			Name:     "passthrough",
			Metadata: nil,
			Sync:     true,
			Terminal: false,
			Run:      passthrough,
		},
		{
			Name:     "terminate",
			Metadata: nil,
			Sync:     true,
			Terminal: true,
			Run:      terminate,
		},
		{
			Name:     "log",
			Metadata: nil,
			Sync:     false,
			Terminal: false,
			Run:      log,
		},
	}
)

func passthrough(data map[string]any, params ...sdkAct.Parameter) (any, error) {
	return true, nil
}

func terminate(data map[string]any, params ...sdkAct.Parameter) (any, error) {
	if len(params) == 0 || params[0].Key != "logger" {
		return nil, gerr.ErrLoggerRequired
	}

	logger, ok := params[0].Value.(zerolog.Logger)
	if !ok {
		return nil, gerr.ErrLoggerRequired
	}

	if len(params) >= TerminateDefaultFieldCount {
		if params[1].Key != "result" {
			logger.Debug().Msg(
				"terminate action can optionally receive a result parameter")
			return true, nil
		}

		result, ok := params[1].Value.(map[string]any)
		if !ok {
			logger.Debug().Msg("terminate action can receive a result parameter")
			return true, nil
		}

		// If the result from the plugin does not contain a response,
		// yet it is a terminal action (hence running this action),
		// add an error response to the result and terminate the connection.
		if _, exists := result["response"]; !exists {
			logger.Trace().Fields(result).Msg(
				"Terminating without response, returning an error response")
			result["response"] = (&pgproto3.Terminate{}).Encode(
				postgres.ErrorResponse(
					"Request terminated",
					"ERROR",
					"42000",
					"Policy terminated the request",
				),
			)
		}

		return result, nil
	}

	return true, nil
}

func log(data map[string]any, params ...sdkAct.Parameter) (any, error) {
	fields := map[string]any{}
	// Only log the fields that are not level, message, or log.
	if len(data) > LogDefaultFieldCount {
		for k, v := range data {
			if k == "level" || k == "message" || k == "log" {
				continue
			}
			fields[k] = v
		}
	}

	if len(params) == 0 || params[0].Key != "logger" {
		// No logger parameter or the first parameter is not a logger.
		return false, nil
	}

	logger, ok := params[0].Value.(zerolog.Logger)
	if !ok {
		// The first parameter is not a logger.
		return false, nil
	}

	logger.WithLevel(
		logging.GetZeroLogLevel(cast.ToString(data["level"])),
	).Fields(fields).Msg(cast.ToString(data["message"]))

	return true, nil
}
