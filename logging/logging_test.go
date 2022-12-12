package logging

import (
	"bytes"
	"encoding/json"
	"testing"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
)

func TestNewLogger(t *testing.T) {
	var buffer bytes.Buffer
	logger := NewLogger(
		LoggerConfig{
			Output:     &buffer,
			Level:      zerolog.DebugLevel,
			TimeFormat: zerolog.TimeFormatUnix,
			NoColor:    true,
		},
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

	logger.Error().Str("key", "key").Msg("This is an error")
	var msg2 interface{}
	err = json.Unmarshal(buffer.Bytes(), &msg2)
	assert.NoError(t, err)

	if jsonMsg, ok := msg2.(map[string]interface{}); ok {
		assert.Equal(t, "This is an error", jsonMsg["message"])
		assert.Equal(t, "error", jsonMsg["level"])
		assert.Equal(t, "key", jsonMsg["key"])
	} else {
		t.Fail()
	}
}
