//go:build !windows
// +build !windows

package logging

import (
	"log/syslog"
	"testing"
	"time"

	"github.com/gatewayd-io/gatewayd/config"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
)

func TestSyslogAndRsyslog(t *testing.T) {
	go func() {
		testServer("tcp", "127.0.0.1:1514")
	}()

	time.Sleep(1 * time.Second) // wait for the test server to start

	logger := NewLogger(
		t.Context(),
		LoggerConfig{
			Output:            []config.LogOutput{config.Syslog, config.RSyslog},
			TimeFormat:        zerolog.TimeFormatUnix,
			Level:             zerolog.WarnLevel,
			NoColor:           true,
			ConsoleTimeFormat: time.RFC3339,
			RSyslogNetwork:    "tcp",
			RSyslogAddress:    "localhost:1514",
			SyslogPriority:    syslog.LOG_DAEMON | syslog.LOG_WARNING,
			FileName:          "",
			MaxSize:           0,
			MaxBackups:        0,
			MaxAge:            0,
			Compress:          false,
			LocalTime:         false,
			ConsoleOut:        nil,
			Name:              config.Default,
		},
	)
	assert.NotNil(t, logger)
}
