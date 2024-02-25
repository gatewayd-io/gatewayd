package act

import (
	sdkAct "github.com/gatewayd-io/gatewayd-plugin-sdk/act"
	"github.com/gatewayd-io/gatewayd-plugin-sdk/logging"
	"github.com/rs/zerolog"
	"github.com/spf13/cast"
)

const LOG_DEFAULT_FIELD_COUNT = 3

var (
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
			Run: func(data map[string]any, params ...sdkAct.Parameter) (any, error) {
				return true, nil
			},
		},
		{
			Name:     "terminate",
			Metadata: nil,
			Sync:     true,
			Terminal: true,
			Run:      func(data map[string]any, params ...sdkAct.Parameter) (any, error) { return true, nil },
		},
		{
			Name:     "log",
			Metadata: nil,
			Sync:     false,
			Terminal: false,
			Run: func(data map[string]any, params ...sdkAct.Parameter) (any, error) {
				fields := map[string]any{}
				// Only log the fields that are not level, message, or log.
				if len(data) > LOG_DEFAULT_FIELD_COUNT {
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
			},
		},
	}

	// TODO: figure out if the default output should match the default policy.
	DefaultOutput = func() *sdkAct.Output {
		return &sdkAct.Output{
			MatchedPolicy: "passthrough",
			Verdict:       true,
			Terminal:      false,
			Sync:          true,
		}
	}
)
