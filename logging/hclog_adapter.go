package logging

import (
	"fmt"
	"io"
	"log"
	"reflect"

	"github.com/hashicorp/go-hclog"
	"github.com/rs/zerolog"
)

// NewHcLogAdapter creates a new hclog.Logger that wraps a zerolog.Logger.
func NewHcLogAdapter(logger *zerolog.Logger, name string) hclog.Logger {
	return &HcLogAdapter{logger, name, nil}
}

type HcLogAdapter struct {
	logger *zerolog.Logger
	name   string

	impliedArgs []any
}

func (h *HcLogAdapter) Log(level hclog.Level, msg string, args ...any) {
	switch level {
	case hclog.Off:
		return
	case hclog.NoLevel:
		return
	case hclog.Trace:
		h.Trace(msg, args...)
	case hclog.Debug:
		h.Debug(msg, args...)
	case hclog.Info:
		h.Info(msg, args...)
	case hclog.Warn:
		h.Warn(msg, args...)
	case hclog.Error:
		h.Error(msg, args...)
	}
}

func (h *HcLogAdapter) Trace(msg string, args ...any) {
	extraArgs := ToMap(args)
	extraArgs["plugin"] = h.name
	h.logger.Trace().Fields(extraArgs).Msg(msg)
}

func (h *HcLogAdapter) Debug(msg string, args ...any) {
	extraArgs := ToMap(args)
	extraArgs["plugin"] = h.name
	h.logger.Debug().Fields(extraArgs).Msg(msg)
}

func (h *HcLogAdapter) Info(msg string, args ...any) {
	extraArgs := ToMap(args)
	extraArgs["plugin"] = h.name
	h.logger.Info().Fields(extraArgs).Msg(msg)
}

func (h *HcLogAdapter) Warn(msg string, args ...any) {
	extraArgs := ToMap(args)
	extraArgs["plugin"] = h.name
	h.logger.Warn().Fields(extraArgs).Msg(msg)
}

func (h *HcLogAdapter) Error(msg string, args ...any) {
	extraArgs := ToMap(args)
	extraArgs["plugin"] = h.name
	h.logger.Error().Fields(extraArgs).Msg(msg)
}

func (h *HcLogAdapter) GetLevel() hclog.Level {
	switch h.logger.GetLevel() {
	case zerolog.Disabled:
		return hclog.Off
	case zerolog.NoLevel:
		return hclog.NoLevel
	case zerolog.TraceLevel:
		return hclog.Trace
	case zerolog.DebugLevel:
		return hclog.Debug
	case zerolog.InfoLevel:
		return hclog.Info
	case zerolog.WarnLevel:
		return hclog.Warn
	case zerolog.ErrorLevel:
		return hclog.Error
	case zerolog.FatalLevel:
		return hclog.Error
	case zerolog.PanicLevel:
		return hclog.Error
	}
	return hclog.NoLevel
}

func (h *HcLogAdapter) IsTrace() bool {
	return h.logger.GetLevel() >= zerolog.TraceLevel
}

func (h *HcLogAdapter) IsDebug() bool {
	return h.logger.GetLevel() >= zerolog.DebugLevel
}

func (h *HcLogAdapter) IsInfo() bool {
	return h.logger.GetLevel() >= zerolog.InfoLevel
}

func (h *HcLogAdapter) IsWarn() bool {
	return h.logger.GetLevel() >= zerolog.WarnLevel
}

func (h *HcLogAdapter) IsError() bool {
	return h.logger.GetLevel() >= zerolog.ErrorLevel
}

func (h *HcLogAdapter) ImpliedArgs() []any {
	// Not supported
	return nil
}

func (h *HcLogAdapter) With(args ...any) hclog.Logger {
	logger := h.logger.With().Fields(ToMap(args)).Logger()
	return NewHcLogAdapter(&logger, h.Name())
}

func (h *HcLogAdapter) Name() string {
	return h.name
}

func (h *HcLogAdapter) Named(name string) hclog.Logger {
	return NewHcLogAdapter(h.logger, name)
}

func (h *HcLogAdapter) ResetNamed(_ string) hclog.Logger {
	return h
}

func (h *HcLogAdapter) SetLevel(level hclog.Level) {
	leveledLog := h.logger.Level(convertLevel(level))
	h.logger = &leveledLog
}

func (h *HcLogAdapter) StandardLogger(opts *hclog.StandardLoggerOptions) *log.Logger {
	if opts == nil {
		opts = &hclog.StandardLoggerOptions{}
	}
	return log.New(h.StandardWriter(opts), "", 0)
}

func (h *HcLogAdapter) StandardWriter(_ *hclog.StandardLoggerOptions) io.Writer {
	v := reflect.ValueOf(h.logger)
	w := v.FieldByName("w")
	writer, ok := w.Interface().(zerolog.LevelWriter)
	if !ok {
		return nil
	}
	return writer
}

func convertLevel(level hclog.Level) zerolog.Level {
	switch level {
	case hclog.Off:
		return zerolog.Disabled
	case hclog.NoLevel:
		return zerolog.NoLevel
	case hclog.Trace:
		return zerolog.TraceLevel
	case hclog.Debug:
		return zerolog.DebugLevel
	case hclog.Info:
		return zerolog.InfoLevel
	case hclog.Warn:
		return zerolog.WarnLevel
	case hclog.Error:
		return zerolog.ErrorLevel
	}
	return zerolog.NoLevel
}

func ToMap(keyValues []any) map[string]any {
	mapped := map[string]any{}

	if len(keyValues) == 0 {
		return mapped
	}

	if len(keyValues)%2 == 1 {
		keyValues = append(keyValues, nil)
	}

	for i := 0; i < len(keyValues); i += 2 {
		if formatString, ok := keyValues[i+1].(hclog.Format); ok {
			merge(mapped, keyValues[i], fmt.Sprintf(formatString[0].(string), formatString[1:]...))
			continue
		}

		merge(mapped, keyValues[i], keyValues[i+1])
	}

	return mapped
}

func merge(mapped map[string]any, key, value any) {
	var casted string

	switch castedKey := key.(type) {
	case string:
		casted = castedKey
	case fmt.Stringer:
		casted = safeString(castedKey)
	default:
		casted = fmt.Sprint(castedKey)
	}

	mapped[casted] = value
}

//nolint:nonamedreturns
func safeString(str fmt.Stringer) (s string) {
	defer func() {
		if panicVal := recover(); panicVal != nil {
			if v := reflect.ValueOf(str); v.Kind() == reflect.Ptr && v.IsNil() {
				s = "NULL"
			} else {
				panic(panicVal)
			}
		}
	}()

	s = str.String()

	return
}
