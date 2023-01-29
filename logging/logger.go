package logging

import (
	"bytes"
	"io"
	"log"
	"log/syslog"
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
	Output            []config.LogOutput
	TimeFormat        string
	Level             zerolog.Level
	NoColor           bool
	ConsoleTimeFormat string

	// (R)Syslog configuration.
	RSyslogNetwork string
	RSyslogAddress string
	SyslogPriority syslog.Priority

	// File output configuration.
	FileName   string
	MaxSize    int
	MaxBackups int
	MaxAge     int
	Compress   bool
	LocalTime  bool
}

// NewLogger creates a new logger with the given configuration.
func NewLogger(cfg LoggerConfig) zerolog.Logger {
	return NewLoggerWithBuffer(cfg, nil)
}

// NewLoggerWithBuffer creates a new logger with the given configuration.
func NewLoggerWithBuffer(cfg LoggerConfig, buffer *bytes.Buffer) zerolog.Logger {
	// Create a new logger.
	consoleWriter := zerolog.ConsoleWriter{
		Out:        os.Stdout,
		TimeFormat: cfg.ConsoleTimeFormat,
		NoColor:    cfg.NoColor,
	}

	var output []io.Writer

	for _, out := range cfg.Output {
		switch out {
		case config.Console:
			output = append(output, consoleWriter)
		case config.Stdout:
			output = append(output, os.Stdout)
		case config.Stderr:
			output = append(output, os.Stderr)
		case config.File:
			output = append(
				output, &lumberjack.Logger{
					Filename:   cfg.FileName,
					MaxSize:    cfg.MaxSize,
					MaxBackups: cfg.MaxBackups,
					MaxAge:     cfg.MaxAge,
					Compress:   cfg.Compress,
					LocalTime:  cfg.LocalTime,
				},
			)
		case config.Syslog:
			syslogWriter, err := syslog.New(cfg.SyslogPriority, config.DefaultSyslogTag)
			if err != nil {
				log.Fatal(err)
			}
			output = append(output, syslogWriter)
		case config.RSyslog:
			// TODO: Add support for TLS.
			// See: https://github.com/RackSec/srslog (deprecated)
			rsyslogWriter, err := syslog.Dial(
				cfg.RSyslogNetwork, cfg.RSyslogAddress, cfg.SyslogPriority, config.DefaultSyslogTag)
			if err != nil {
				log.Fatal(err)
			}
			output = append(output, zerolog.SyslogLevelWriter(rsyslogWriter))
		default:
			output = append(output, os.Stdout)
		}
	}

	zerolog.SetGlobalLevel(cfg.Level)
	zerolog.TimeFieldFormat = cfg.TimeFormat

	multiWriter := zerolog.MultiLevelWriter(output...)
	logger := zerolog.New(multiWriter)
	logger = logger.With().Timestamp().Logger()

	return logger
}
