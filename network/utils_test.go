package network

import (
	"testing"

	"github.com/gatewayd-io/gatewayd/logging"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
)

func TestGetRlimit(t *testing.T) {
	logger := logging.NewLogger(nil, zerolog.TimeFormatUnix, zerolog.DebugLevel, true)
	rlimit := GetRLimit(logger)
	assert.Greater(t, rlimit.Cur, uint64(1))
	assert.Greater(t, rlimit.Max, uint64(1))
}

func TestGetID(t *testing.T) {
	logger := logging.NewLogger(nil, zerolog.TimeFormatUnix, zerolog.DebugLevel, true)
	id := GetID("tcp", "localhost:5432", 1, logger)
	assert.Equal(t, "0cf47ee4e436ecb40dbd1d2d9a47179d1f6d98e2ea18d6fbd1cdfa85d3cec94f", id)
}

func TestResolve(t *testing.T) {
	logger := logging.NewLogger(nil, zerolog.TimeFormatUnix, zerolog.DebugLevel, true)
	address, err := Resolve("udp", "localhost:53", logger)
	assert.NoError(t, err)
	assert.Equal(t, "127.0.0.1:53", address)
}
