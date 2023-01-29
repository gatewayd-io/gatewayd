package logging

import (
	"io"
	"os"

	"github.com/rs/zerolog"
)

type LoggerConfig struct {
	Output     io.Writer
	TimeFormat string
	Level      zerolog.Level
	NoColor    bool
}

func NewLogger(cfg LoggerConfig) zerolog.Logger {
	// Create a new logger
	consoleWriter := zerolog.ConsoleWriter{Out: os.Stdout, TimeFormat: cfg.TimeFormat}

	if cfg.Output == nil {
		// Default to stdout
		cfg.Output = consoleWriter
	}

	if cfg.TimeFormat == "" {
		cfg.TimeFormat = zerolog.TimeFieldFormat
	}

	zerolog.SetGlobalLevel(cfg.Level)
	zerolog.TimeFieldFormat = cfg.TimeFormat

	// Create a new logger
	logger := zerolog.New(cfg.Output)
	if cfg.TimeFormat != "" {
		logger = logger.With().Timestamp().Logger()
	}

	logger.Debug().Msg("Created a new logger")

	return logger
}
