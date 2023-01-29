package logging

import (
	"bytes"
	"encoding/json"
	"os"
	"testing"

	"github.com/gatewayd-io/gatewayd/config"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
)

// TestNewLogger tests the creation of a new logger.
func TestNewLogger(t *testing.T) {
	var buffer bytes.Buffer
	logger := NewLoggerWithBuffer(
		LoggerConfig{
			Output:     config.Buffer, // This is only used for testing.
			Level:      zerolog.DebugLevel,
			TimeFormat: zerolog.TimeFormatUnix,
			StartupMsg: true,
			NoColor:    true,
		},
		&buffer,
	)
	assert.NotNil(t, logger)

	var msg interface{}
	err := json.Unmarshal(buffer.Bytes(), &msg)
	assert.NoError(t, err)

	if jsonMsg, ok := msg.(map[string]interface{}); ok {
		// This is created when the logger is created and
		// is used to test that the logger is working.
		assert.Equal(t, "Created a new logger", jsonMsg["message"])
		assert.Equal(t, "debug", jsonMsg["level"])
	} else {
		t.Fail()
	}

	buffer.Reset()

	logger.Error().Str("key", "value").Msg("This is an error")
	var msg2 interface{}
	err = json.Unmarshal(buffer.Bytes(), &msg2)
	assert.NoError(t, err)

	if jsonMsg, ok := msg2.(map[string]interface{}); ok {
		assert.Equal(t, "This is an error", jsonMsg["message"])
		assert.Equal(t, "error", jsonMsg["level"])
		assert.Equal(t, "value", jsonMsg["key"])
	} else {
		t.Fail()
	}
}

func TestNewLogger_File(t *testing.T) {
	logger := NewLogger(
		LoggerConfig{
			Output:     config.File,
			FileName:   "test.log",
			Permission: config.DefaultLogFilePermission,
			Level:      zerolog.DebugLevel,
			TimeFormat: zerolog.TimeFormatUnix,
			StartupMsg: true,
			NoColor:    true,
		},
	)
	assert.NotNil(t, logger)

	logger.Error().Str("key", "value").Msg("This is an error")

	f, err := os.ReadFile("test.log")
	assert.NoError(t, err)
	assert.NotEmpty(t, f)
	assert.Containsf(t, string(f), "Created a new logger", "The logger did not write to the file")
	os.Remove("./test.log")
}
