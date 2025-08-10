package logging

import (
	"bytes"
	"strings"
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

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a buffer to capture the log output
			var buf bytes.Buffer
			logger := zerolog.New(&buf).Level(zerolog.DebugLevel)

			// Create the writer
			writer := NewStandardLogWriter(logger, "test-component")

			// Write the test input
			n, err := writer.Write([]byte(tt.input))

			// Verify no error and correct byte count
			require.NoError(t, err)
			assert.Equal(t, len(tt.input), n)

			// Parse the output
			output := buf.String()

			if tt.expected == "" {
				// If we expect empty, the logger shouldn't have written anything
				assert.Empty(t, output)
			} else {
				// Verify the message was logged correctly
				assert.Contains(t, output, tt.expected)
				assert.Contains(t, output, `"component":"test-component"`)
				assert.Contains(t, output, `"level":"debug"`)
			}
		})
	}
}

func TestStdlibLogTimestampRegex(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "Go stdlib LstdFlags format (Ldate | Ltime)",
			input:    "2025/01/15 10:17:53 message",
			expected: "message",
		},
		{
			name:     "Go stdlib with microseconds",
			input:    "2025/01/15 10:17:53.123456 message",
			expected: "message",
		},
		{
			name:     "exact Go reference time format",
			input:    "2009/01/23 01:23:23 message",
			expected: "message",
		},
		{
			name:     "non-Go stdlib ISO format should NOT match",
			input:    "2025-01-15 10:17:53 message",
			expected: "2025-01-15 10:17:53 message",
		},
		{
			name:     "time only should NOT match",
			input:    "10:17:53 message",
			expected: "10:17:53 message",
		},
		{
			name:     "single digit day/month should NOT match (Go uses zero-padding)",
			input:    "2025/1/5 10:17:53 message",
			expected: "2025/1/5 10:17:53 message",
		},
		{
			name:     "no timestamp",
			input:    "just a message",
			expected: "just a message",
		},
		{
			name:     "timestamp at end (should not match)",
			input:    "message 2025/01/15 10:17:53",
			expected: "message 2025/01/15 10:17:53",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := stdlibLogTimestampRegex.ReplaceAllString(tt.input, "")
			result = strings.TrimSpace(result)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestCaptureStandardLogs(t *testing.T) {
	// Create a buffer to capture the log output
	var buf bytes.Buffer
	logger := zerolog.New(&buf).Level(zerolog.DebugLevel)

	// Test that the function returns a restore function
	restore := CaptureStandardLogs(logger, "test-capture")
	assert.NotNil(t, restore)

	// Test that we can call the restore function without error
	assert.NotPanics(t, func() {
		restore()
	})
}
