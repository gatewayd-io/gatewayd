//go:build !windows
// +build !windows

package logging

import (
	"context"
	"io"
	"log"
	"log/syslog"
	"os"

	"github.com/gatewayd-io/gatewayd/config"
	"github.com/rs/zerolog"
	"go.opentelemetry.io/otel"
	"gopkg.in/natefinch/lumberjack.v2"
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

	// group name
	Name string
}

// NewLogger creates a new logger with the given configuration.
func NewLogger(ctx context.Context, cfg LoggerConfig) zerolog.Logger {
	_, span := otel.Tracer(config.TracerName).Start(ctx, "Create new logger")

	// Create a new logger.
	consoleWriter := zerolog.ConsoleWriter{
		Out:        os.Stdout,
		TimeFormat: cfg.ConsoleTimeFormat,
		NoColor:    cfg.NoColor,
	}

	var outputs []io.Writer
	for _, out := range cfg.Output {
		switch out {
		case config.Console:
			outputs = append(outputs, consoleWriter)
		case config.Stdout:
			outputs = append(outputs, os.Stdout)
		case config.Stderr:
			outputs = append(outputs, os.Stderr)
		case config.File:
			outputs = append(
				outputs, &lumberjack.Logger{
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
				span.RecordError(err)
				span.End()
				log.Fatal(err)
			}
			outputs = append(outputs, syslogWriter)
		case config.RSyslog:
			// TODO: Add support for TLS.
			// See: https://github.com/RackSec/srslog (deprecated)
			rsyslogWriter, err := syslog.Dial(
				cfg.RSyslogNetwork, cfg.RSyslogAddress, cfg.SyslogPriority, config.DefaultSyslogTag)
			if err != nil {
				log.Fatal(err)
			}
			outputs = append(outputs, zerolog.SyslogLevelWriter(rsyslogWriter))
		default:
			outputs = append(outputs, consoleWriter)
		}
	}

	zerolog.SetGlobalLevel(cfg.Level)
	zerolog.TimeFieldFormat = cfg.TimeFormat

	multiWriter := zerolog.MultiLevelWriter(outputs...)
	logger := zerolog.New(multiWriter)
	logger = logger.With().Timestamp().Logger()
	logger = logger.With().Str("group", cfg.Name).Logger()

	span.End()

	return logger
}
