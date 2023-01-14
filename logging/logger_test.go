package logging

import (
	"os"
	"testing"

	"github.com/gatewayd-io/gatewayd/config"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/zenizh/go-capturer"
)

// TestNewLogger_Console tests the creation of a new logger with the console output.
func TestNewLogger_Console(t *testing.T) {
	consoleStdout := capturer.CaptureStdout(func() {
		logger := NewLogger(
			LoggerConfig{
				Output:     config.Console,
				Level:      zerolog.DebugLevel,
				TimeFormat: zerolog.TimeFormatUnix,
				StartupMsg: true,
				NoColor:    true,
			},
		)
		assert.NotNil(t, logger)

		logger.Error().Str("key", "value").Msg("This is an error")
	})

	assert.Contains(t, consoleStdout, "Created a new logger")
	assert.Contains(t, consoleStdout, "DBG")
	assert.Contains(t, consoleStdout, "This is an error")
	assert.Contains(t, consoleStdout, "ERR")
	assert.Contains(t, consoleStdout, "key=value")
}

// TestNewLogger_File tests the creation of a new logger with the file output.
func TestNewLogger_File(t *testing.T) {
	logger := NewLogger(
		LoggerConfig{
			Output:     config.File,
			FileName:   "gatewayd.log",
			MaxSize:    config.DefaultMaxSize,
			MaxBackups: config.DefaultMaxBackups,
			MaxAge:     config.DefaultMaxAge,
			Compress:   config.DefaultCompress,
			Level:      zerolog.DebugLevel,
			TimeFormat: zerolog.TimeFormatUnix,
			StartupMsg: true,
			NoColor:    true,
		},
	)
	assert.NotNil(t, logger)

	logger.Error().Str("key", "value").Msg("This is an error")

	f, err := os.ReadFile("gatewayd.log")
	assert.NoError(t, err)
	assert.NotEmpty(t, f)
	assert.Containsf(t, string(f), "Created a new logger", "The logger did not write to the file")
	os.Remove("gatewayd.log")
}

// TestNewLogger_Stdout tests the creation of a new logger with the stdout output.
func TestNewLogger_Stdout(t *testing.T) {
	stdout := capturer.CaptureStdout(func() {
		logger := NewLogger(
			LoggerConfig{
				Output:     config.Stdout,
				Level:      zerolog.DebugLevel,
				TimeFormat: zerolog.TimeFormatUnix,
				StartupMsg: true,
				NoColor:    true,
			},
		)
		assert.NotNil(t, logger)

		logger.Error().Str("key", "value").Msg("This is an error")
	})

	assert.Contains(t, stdout, "Created a new logger")
	assert.Contains(t, stdout, `"level":"debug"`)
	assert.Contains(t, stdout, "This is an error")
	assert.Contains(t, stdout, `"level":"error"`)
	assert.Contains(t, stdout, `"key":"value"`)
}

// TestNewLogger_Stderr tests the creation of a new logger with the stderr output.
func TestNewLogger_Stderr(t *testing.T) {
	stderr := capturer.CaptureStderr(func() {
		logger := NewLogger(
			LoggerConfig{
				Output:     config.Stderr,
				Level:      zerolog.DebugLevel,
				TimeFormat: zerolog.TimeFormatUnix,
				StartupMsg: true,
				NoColor:    true,
			},
		)
		assert.NotNil(t, logger)

		logger.Error().Str("key", "value").Msg("This is an error")
	})

	assert.Contains(t, stderr, "Created a new logger")
	assert.Contains(t, stderr, `"level":"debug"`)
	assert.Contains(t, stderr, "This is an error")
	assert.Contains(t, stderr, `"level":"error"`)
	assert.Contains(t, stderr, `"key":"value"`)
}
