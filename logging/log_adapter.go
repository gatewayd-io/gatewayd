package logging

import (
	"log"
	"regexp"
	"strings"

	"github.com/rs/zerolog"
)

// StandardLogWriter implements io.Writer to capture standard log output and redirect it to zerolog.
type StandardLogWriter struct {
	logger    zerolog.Logger
	component string
}

// stdlibLogTimestampRegex matches Go stdlib log timestamp format:
// "2009/01/23 01:23:23 ".
var stdlibLogTimestampRegex = regexp.MustCompile(`^\d{4}/\d{2}/\d{2} \d{2}:\d{2}:\d{2}(\.\d{6})?\s`)

// NewStandardLogWriter creates a new StandardLogWriter that
// redirects standard log output to zerolog.
func NewStandardLogWriter(logger zerolog.Logger, component string) *StandardLogWriter {
	return &StandardLogWriter{
		logger:    logger,
		component: component,
	}
}

// Write implements io.Writer interface and redirects log output to zerolog.
func (w *StandardLogWriter) Write(p []byte) (int, error) {
	message := string(p)

	// Remove any stdlib log timestamps as defensive measure first.
	message = stdlibLogTimestampRegex.ReplaceAllString(message, "")

	// Then trim whitespace.
	message = strings.TrimSpace(message)
	if message == "" {
		return len(p), nil
	}

	logger := w.logger.With().Str("component", w.component).Logger()

	// For now, we'll log all messages as debug,
	// until we find a better way to handle log levels.
	logger.Debug().Msg(message)

	return len(p), nil
}

// CaptureStandardLogs redirects standard log output to zerolog and returns a restore function.
func CaptureStandardLogs(logger zerolog.Logger, component string) func() {
	originalOutput := log.Writer()
	originalFlags := log.Flags()
	originalPrefix := log.Prefix()

	logWriter := NewStandardLogWriter(logger, component)
	log.SetOutput(logWriter)
	log.SetFlags(0)
	log.SetPrefix("")
	return func() {
		log.SetOutput(originalOutput)
		log.SetFlags(originalFlags)
		log.SetPrefix(originalPrefix)
	}
}
