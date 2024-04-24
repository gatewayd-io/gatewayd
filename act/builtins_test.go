package act

import (
	"testing"

	sdkAct "github.com/gatewayd-io/gatewayd-plugin-sdk/act"
	"github.com/gatewayd-io/gatewayd-plugin-sdk/databases/postgres"
	gerr "github.com/gatewayd-io/gatewayd/errors"
	"github.com/jackc/pgx/v5/pgproto3"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_Terminate_Action(t *testing.T) {
	response, err := (&pgproto3.Terminate{}).Encode(
		postgres.ErrorResponse(
			"Request terminated",
			"ERROR",
			"42000",
			"Policy terminated the request",
		),
	)
	require.NoError(t, err)

	tests := []struct {
		params []sdkAct.Parameter
		result any
		err    error
	}{
		{
			params: []sdkAct.Parameter{},
			result: nil,
			err:    gerr.ErrLoggerRequired,
		},
		{
			params: []sdkAct.Parameter{
				{
					Key:   LoggerKey,
					Value: nil,
				},
			},
			result: nil,
			err:    gerr.ErrLoggerRequired,
		},
		{
			params: []sdkAct.Parameter{
				{
					Key:   LoggerKey,
					Value: zerolog.New(nil),
				},
			},
			result: true,
		},
		{
			params: []sdkAct.Parameter{
				{
					Key:   LoggerKey,
					Value: zerolog.New(nil),
				},
				{
					Key:   ResultKey,
					Value: nil,
				},
			},
			result: true,
		},
		{
			params: []sdkAct.Parameter{
				{
					Key:   LoggerKey,
					Value: zerolog.New(nil),
				},
				{
					Key:   ResultKey,
					Value: map[string]any{},
				},
			},
			result: map[string]any{"response": response},
		},
	}

	for _, test := range tests {
		t.Run("Test_Terminate_Action", func(t *testing.T) {
			result, err := Terminate(nil, test.params...)
			assert.ErrorIs(t, err, test.err)
			assert.Equal(t, result, test.result)
		})
	}
}

func Test_Log_Action(t *testing.T) {
	tests := []struct {
		params []sdkAct.Parameter
		result any
		err    error
	}{
		{
			params: []sdkAct.Parameter{},
			result: nil,
			err:    gerr.ErrLoggerRequired,
		},
		{
			params: []sdkAct.Parameter{
				{
					Key:   LoggerKey,
					Value: nil,
				},
			},
			result: nil,
			err:    gerr.ErrLoggerRequired,
		},
		{
			params: []sdkAct.Parameter{
				{
					Key:   LoggerKey,
					Value: zerolog.New(nil),
				},
			},
			result: true,
		},
	}

	for _, test := range tests {
		t.Run("Test_Log_Action", func(t *testing.T) {
			result, err := Log(nil, test.params...)
			assert.ErrorIs(t, err, test.err)
			assert.Equal(t, result, test.result)
		})
	}
}
