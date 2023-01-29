package logging

import (
	"io"
	"os"

	"github.com/rs/zerolog"
)

func NewLogger(
	writer io.Writer,
	timeFieldFormat string,
	level zerolog.Level,
	timestamp bool,
) zerolog.Logger {
	// Create a new logger
	consoleWriter := zerolog.ConsoleWriter{Out: os.Stdout, TimeFormat: timeFieldFormat}

	if writer == nil {
		// Default to stdout
		writer = consoleWriter
	}

	if timeFieldFormat == "" {
		timeFieldFormat = zerolog.TimeFieldFormat
	}

	zerolog.SetGlobalLevel(level)
	zerolog.TimeFieldFormat = timeFieldFormat

	// Create a new logger
	logger := zerolog.New(writer)
	if timestamp {
		logger = logger.With().Timestamp().Logger()
	}

	logger.Debug().Msg("Created a new logger")

	return logger
}
