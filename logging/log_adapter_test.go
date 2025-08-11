package logging

import (
	"bytes"
	"testing"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestStandardLogWriter_Write(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "message without timestamp",
			input:    "simple log message",
			expected: "simple log message",
		},
		{
			name:     "message with Go stdlib LstdFlags format (default)",
			input:    "2025/01/15 10:17:53 log message here",
			expected: "log message here",
		},
		{
			name:     "message with Go stdlib Ldate|Ltime|Lmicroseconds format",
			input:    "2025/01/15 10:17:53.123456 log message here",
			expected: "log message here",
		},
		{
			name:     "message with exact Go stdlib format (zero-padded)",
			input:    "2009/01/23 01:23:23 log message here",
			expected: "log message here",
		},
		{
			name:     "empty message",
			input:    "",
			expected: "",
		},
		{
			name:     "only Go stdlib timestamp",
			input:    "2025/01/15 10:17:53 ",
			expected: "",
		},
		{
			name:     "message with multiple spaces after Go stdlib timestamp",
			input:    "2025/01/15 10:17:53   log message with spaces",
			expected: "log message with spaces",
		},
		{
			name:     "non-Go stdlib timestamp format should not be removed",
			input:    "2025-01-15 10:17:53 log message",
			expected: "2025-01-15 10:17:53 log message",
		},
		{
			name:     "time-only format should not be removed",
			input:    "10:17:53 log message",
			expected: "10:17:53 log message",
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			// Save current global log level and restore it after test
			originalLevel := zerolog.GlobalLevel()
			t.Cleanup(func() {
				zerolog.SetGlobalLevel(originalLevel)
			})

			// Ensure debug level is enabled for this test
			zerolog.SetGlobalLevel(zerolog.DebugLevel)

			// Create a buffer to capture the log output
			var buf bytes.Buffer
			logger := zerolog.New(&buf).Level(zerolog.DebugLevel)

			// Create the writer
			writer := NewStandardLogWriter(logger, "test-component")

			// Write the test input
			n, err := writer.Write([]byte(testCase.input))

			// Verify no error and correct byte count
			require.NoError(t, err)
			assert.Equal(t, len(testCase.input), n)

			// Parse the output
			output := buf.String()

			if testCase.expected == "" {
				// If we expect empty, the logger shouldn't have written anything
				assert.Empty(t, output)
			} else {
				// Verify the message was logged correctly
				assert.Contains(t, output, testCase.expected)
				assert.Contains(t, output, `"component":"test-component"`)
				assert.Contains(t, output, `"level":"debug"`)
			}
		})
	}
}

func TestCaptureStandardLogs(t *testing.T) {
	// Save current global log level and restore it after test
	originalLevel := zerolog.GlobalLevel()

	// Ensure debug level is enabled for this test
	zerolog.SetGlobalLevel(zerolog.DebugLevel)

	// Create a buffer to capture the log output
	var buf bytes.Buffer
	logger := zerolog.New(&buf).Level(zerolog.DebugLevel)

	// Test that the function returns a restore function
	restore := CaptureStandardLogs(logger, "test-capture")
	assert.NotNil(t, restore)
	t.Cleanup(func() {
		restore()
		zerolog.SetGlobalLevel(originalLevel)
	})
}
