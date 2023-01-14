package logging

import (
	"bytes"
	"io"
	"os"

	"github.com/gatewayd-io/gatewayd/config"
	"github.com/rs/zerolog"
	"gopkg.in/natefinch/lumberjack.v2"
)

// TODO: Remove this once we have a proper hooks package.
// This is duplicated from the network package, because import cycles are not allowed.
type (
	Signature   map[string]interface{}
	HookDef     func(Signature) Signature
	OnNewLogger HookDef
)

type LoggerConfig struct {
	Output     config.LogOutput
	TimeFormat string
	Level      zerolog.Level
	NoColor    bool
	StartupMsg bool

	FileName   string
	MaxSize    int
	MaxBackups int
	MaxAge     int
	Compress   bool

	hook OnNewLogger
}

// NewLogger creates a new logger with the given configuration.
func NewLogger(cfg LoggerConfig) zerolog.Logger {
	return NewLoggerWithBuffer(cfg)
}

// NewLoggerWithBuffer creates a new logger with the given configuration.
//
//nolint:funlen
func NewLoggerWithBuffer(cfg LoggerConfig, buffer ...*bytes.Buffer) zerolog.Logger {
	// Create a new logger.
	consoleWriter := zerolog.ConsoleWriter{
		Out:        os.Stdout,
		TimeFormat: cfg.TimeFormat,
		NoColor:    cfg.NoColor,
	}

	var output io.Writer

	if cfg.FileName == "" {
		cfg.FileName = config.DefaultLogFileName
	}

	switch cfg.Output {
	case config.Console:
		output = consoleWriter
	case config.Stdout:
		output = os.Stdout
	case config.Stderr:
		output = os.Stderr
	case config.File:
		output = &lumberjack.Logger{
			Filename:   cfg.FileName,
			MaxSize:    cfg.MaxSize,
			MaxBackups: cfg.MaxBackups,
			MaxAge:     cfg.MaxAge,
			Compress:   cfg.Compress,
		}
	case config.Buffer:
		if len(buffer) == 0 {
			output = os.Stdout
		} else {
			output = buffer[0]
		}
	default:
		output = os.Stdout
	}

	if cfg.TimeFormat == "" {
		cfg.TimeFormat = zerolog.TimeFieldFormat
	}

	zerolog.SetGlobalLevel(cfg.Level)
	zerolog.TimeFieldFormat = cfg.TimeFormat

	// Create a new logger.
	logger := zerolog.New(output)
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
