package network

import (
	"context"
	"testing"
	"time"

	"github.com/gatewayd-io/gatewayd/config"
	"github.com/gatewayd-io/gatewayd/logging"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/zenizh/go-capturer"
)

func TestEngine(t *testing.T) {
	output := capturer.CaptureOutput(func() {
		logger := logging.NewLogger(context.Background(), logging.LoggerConfig{
			Output:            []config.LogOutput{config.Console},
			TimeFormat:        zerolog.TimeFormatUnix,
			ConsoleTimeFormat: time.RFC3339,
			Level:             zerolog.WarnLevel,
			NoColor:           true,
		})
		engine := NewEngine(logger)
		assert.NotNil(t, engine)
		assert.NotNil(t, engine.logger)
		assert.Zero(t, engine.connections)
		assert.Zero(t, engine.CountConnections())
		assert.Empty(t, engine.host)
		assert.Empty(t, engine.port)
		assert.False(t, engine.running.Load())

		go func(engine Engine) {
			v := <-engine.stopServer
			// Empty struct is expected to be received and
			// it means that the server is stopped.
			assert.Equal(t, struct{}{}, v)
		}(engine)

		err := engine.Stop(context.Background())
		// This is expected to fail because the engine is not running.
		assert.Nil(t, err)
		assert.False(t, engine.running.Load())
		assert.Zero(t, engine.connections)
	})

	// Expected output:
	assert.Contains(t, output, "ERR Listener is not initialized")
}
