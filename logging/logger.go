package logging

import (
	"io"
	"os"

	"github.com/rs/zerolog"
)

// This is duplicated from the network package, because import cycles are not allowed.
type (
	Signature   map[string]interface{}
	HookDef     func(Signature) Signature
	OnNewLogger HookDef
)

type LoggerConfig struct {
	Output     io.Writer
	TimeFormat string
	Level      zerolog.Level
	NoColor    bool
	StartupMsg bool
	hook       OnNewLogger
}

// NewLogger creates a new logger with the given configuration.
func NewLogger(cfg LoggerConfig) zerolog.Logger {
	// Create a new logger.
	consoleWriter := zerolog.ConsoleWriter{
		Out:        os.Stdout,
		TimeFormat: cfg.TimeFormat,
		NoColor:    cfg.NoColor,
	}

	if cfg.Output == nil {
		// Default to stdout.
		cfg.Output = consoleWriter
	}

	if cfg.TimeFormat == "" {
		cfg.TimeFormat = zerolog.TimeFieldFormat
	}

	zerolog.SetGlobalLevel(cfg.Level)
	zerolog.TimeFieldFormat = cfg.TimeFormat

	// Create a new logger.
	logger := zerolog.New(cfg.Output)
	if cfg.TimeFormat != "" {
		logger = logger.With().Timestamp().Logger()
	}

	if cfg.StartupMsg {
		logger.Debug().Msg("Created a new logger")
	}

	if cfg.hook != nil {
		cfg.hook(Signature{"logger": logger})
	}

	return logger
}
