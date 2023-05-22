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
				span.RecordError(err)
				span.End()
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
	if cfg.TimeFormat == "unix" || cfg.TimeFormat == "" {
		zerolog.TimeFieldFormat = zerolog.TimeFormatUnix
	} else {
		zerolog.TimeFieldFormat = cfg.TimeFormat
	}

	multiWriter := zerolog.MultiLevelWriter(output...)
	logger := zerolog.New(multiWriter)
	logger = logger.With().Timestamp().Logger()

	span.End()

	return logger
}
