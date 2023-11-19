package network

import (
	"context"
	"testing"
	"time"

	"github.com/gatewayd-io/gatewayd/config"
	"github.com/gatewayd-io/gatewayd/logging"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
)

func TestRetry(t *testing.T) {
	logger := logging.NewLogger(context.Background(), logging.LoggerConfig{
		Output:            []config.LogOutput{config.Console},
		TimeFormat:        zerolog.TimeFormatUnix,
		ConsoleTimeFormat: time.RFC3339,
		Level:             zerolog.DebugLevel,
		NoColor:           true,
	})

	t.Run("DialTimeout", func(t *testing.T) {
		t.Run("nil", func(t *testing.T) {
			// Nil retry should just dial the connection once.
			var retry *Retry
			_, err := retry.DialTimeout("", "", 0)
			assert.Error(t, err)
			assert.ErrorContains(t, err, "dial: unknown network ")
		})
		t.Run("retry without timeout", func(t *testing.T) {
			retry := NewRetry(0, 0, 0, false, logger)
			assert.Equal(t, 1, retry.Retries)
			assert.Equal(t, time.Duration(0), retry.Backoff)
			assert.Equal(t, float64(0), retry.BackoffMultiplier)
			assert.False(t, retry.DisableBackoffCaps)

			conn, err := retry.DialTimeout("tcp", "localhost:5432", 0)
			assert.NoError(t, err)
			assert.NotNil(t, conn)
			conn.Close()
		})
		t.Run("retry with timeout", func(t *testing.T) {
			retry := NewRetry(
				config.DefaultRetries,
				config.DefaultBackoff,
				config.DefaultBackoffMultiplier,
				config.DefaultDisableBackoffCaps,
				logger,
			)
			assert.Equal(t, config.DefaultRetries, retry.Retries)
			assert.Equal(t, config.DefaultBackoff, retry.Backoff)
			assert.Equal(t, config.DefaultBackoffMultiplier, retry.BackoffMultiplier)
			assert.False(t, retry.DisableBackoffCaps)

			conn, err := retry.DialTimeout("tcp", "localhost:5432", time.Second)
			assert.NoError(t, err)
			assert.NotNil(t, conn)
			conn.Close()
		})
	})
}
