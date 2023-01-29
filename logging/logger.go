package logging

import (
	"io"
	"os"

	"github.com/rs/zerolog"
)

// This is duplicated from the network package, because import cycles are not allowed
type HookDef func(map[string]interface{}) map[string]interface{}
type Signature map[string]interface{}

type OnNewLogger HookDef

type LoggerConfig struct {
	Output      io.Writer
	TimeFormat  string
	Level       zerolog.Level
	NoColor     bool
	OnNewLogger OnNewLogger
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

	if cfg.OnNewLogger != nil {
		cfg.OnNewLogger(Signature{"logger": logger})
	}

	return logger
}
