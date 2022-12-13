package logging

import (
	"fmt"
	"io"
	"log"
	"reflect"

	"github.com/hashicorp/go-hclog"
	"github.com/rs/zerolog"
)

// Creates hclog.Logger adapter from a zerolog log
func NewHcLogAdapter(logger *zerolog.Logger, name string) hclog.Logger {
	return &HcLogAdapter{logger, name, nil}
}

type HcLogAdapter struct {
	logger *zerolog.Logger
	name   string

	impliedArgs []interface{}
}

func (z HcLogAdapter) Log(level hclog.Level, msg string, args ...interface{}) {
	switch level {
	case hclog.NoLevel:
		return
	case hclog.Trace:
		z.Trace(msg, args...)
	case hclog.Debug:
		z.Debug(msg, args...)
	case hclog.Info:
		z.Info(msg, args...)
	case hclog.Warn:
		z.Warn(msg, args...)
	case hclog.Error:
		z.Error(msg, args...)
	}
}

func (z HcLogAdapter) Trace(msg string, args ...interface{}) {
	z.logger.Trace().Fields(ToMap(args)).Msg(msg)
}

func (z HcLogAdapter) Debug(msg string, args ...interface{}) {
	z.logger.Debug().Fields(ToMap(args)).Msg(msg)
}

func (z HcLogAdapter) Info(msg string, args ...interface{}) {
	z.logger.Info().Fields(ToMap(args)).Msg(msg)
}

func (z HcLogAdapter) Warn(msg string, args ...interface{}) {
	z.logger.Warn().Fields(ToMap(args)).Msg(msg)
}

func (z HcLogAdapter) Error(msg string, args ...interface{}) {
	z.logger.Error().Fields(ToMap(args)).Msg(msg)
}

func (z HcLogAdapter) GetLevel() hclog.Level {
	switch z.logger.GetLevel() {
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
	}
	return hclog.NoLevel
}

func (h HcLogAdapter) IsTrace() bool {
	return h.logger.GetLevel() >= zerolog.TraceLevel
}

func (h HcLogAdapter) IsDebug() bool {
	return h.logger.GetLevel() >= zerolog.DebugLevel
}

func (h HcLogAdapter) IsInfo() bool {
	return h.logger.GetLevel() >= zerolog.InfoLevel
}

func (h HcLogAdapter) IsWarn() bool {
	return h.logger.GetLevel() >= zerolog.WarnLevel
}

func (h HcLogAdapter) IsError() bool {
	return h.logger.GetLevel() >= zerolog.ErrorLevel
}

func (h HcLogAdapter) ImpliedArgs() []interface{} {
	// Not supported
	return nil
}

func (h HcLogAdapter) With(args ...interface{}) hclog.Logger {
	logger := h.logger.With().Fields(ToMap(args)).Logger()
	return NewHcLogAdapter(&logger, h.Name())
}

func (h HcLogAdapter) Name() string {
	return h.name
}

func (h HcLogAdapter) Named(name string) hclog.Logger {
	return NewHcLogAdapter(h.logger, name)
}

func (h HcLogAdapter) ResetNamed(name string) hclog.Logger {
	return &h
}

func (h *HcLogAdapter) SetLevel(level hclog.Level) {
	leveledLog := h.logger.Level(convertLevel(level))
	h.logger = &leveledLog
}

func (h HcLogAdapter) StandardLogger(opts *hclog.StandardLoggerOptions) *log.Logger {
	if opts == nil {
		opts = &hclog.StandardLoggerOptions{}
	}
	return log.New(h.StandardWriter(opts), "", 0)
}

func (h HcLogAdapter) StandardWriter(opts *hclog.StandardLoggerOptions) io.Writer {
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

// Ref: https://github.com/nitrocode/cloudquery/blob/main/logging/keyvals/to_map.go
func ToMap(kvs []interface{}) map[string]interface{} {
	m := map[string]interface{}{}

	if len(kvs) == 0 {
		return m
	}

	if len(kvs)%2 == 1 {
		kvs = append(kvs, nil)
	}

	for i := 0; i < len(kvs); i += 2 {
		merge(m, kvs[i], kvs[i+1])
	}

	return m
}

func merge(dst map[string]interface{}, k, v interface{}) {
	var key string

	switch x := k.(type) {
	case string:
		key = x
	case fmt.Stringer:
		key = safeString(x)
	default:
		key = fmt.Sprint(x)
	}

	dst[key] = v
}

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
