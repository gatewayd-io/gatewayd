package network

import (
	"testing"

	"github.com/gatewayd-io/gatewayd/config"
	"github.com/gatewayd-io/gatewayd/logging"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
)

// TestGetRlimit tests the GetRLimit function.
func TestGetRlimit(t *testing.T) {
	cfg := logging.LoggerConfig{
		Output:            []config.LogOutput{config.Console},
		TimeFormat:        zerolog.TimeFormatUnix,
		ConsoleTimeFormat: config.DefaultConsoleTimeFormat,
		Level:             zerolog.DebugLevel,
		NoColor:           true,
	}

	logger := logging.NewLogger(cfg)
	rlimit := GetRLimit(logger)
	assert.Greater(t, rlimit.Cur, uint64(1))
	assert.Greater(t, rlimit.Max, uint64(1))
}

// TestGetID tests the GetID function.
func TestGetID(t *testing.T) {
	cfg := logging.LoggerConfig{
		Output:            []config.LogOutput{config.Console},
		TimeFormat:        zerolog.TimeFormatUnix,
		ConsoleTimeFormat: config.DefaultConsoleTimeFormat,
		Level:             zerolog.DebugLevel,
		NoColor:           true,
	}

	logger := logging.NewLogger(cfg)
	id := GetID("tcp", "localhost:5432", 1, logger)
	assert.Equal(t, "0cf47ee4e436ecb40dbd1d2d9a47179d1f6d98e2ea18d6fbd1cdfa85d3cec94f", id)
}

// TestResolve tests the Resolve function.
func TestResolve(t *testing.T) {
	cfg := logging.LoggerConfig{
		Output:            []config.LogOutput{config.Console},
		TimeFormat:        zerolog.TimeFormatUnix,
		ConsoleTimeFormat: config.DefaultConsoleTimeFormat,
		Level:             zerolog.DebugLevel,
		NoColor:           true,
	}

	logger := logging.NewLogger(cfg)
	address, err := Resolve("udp", "localhost:53", logger)
	assert.Nil(t, err)
	assert.Equal(t, "127.0.0.1:53", address)
}
