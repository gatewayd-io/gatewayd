//go:build !windows
// +build !windows

package config

import (
	"log/syslog"
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestSyslogPriority tests the GetSyslogPriority function.
func TestSyslogPriority(t *testing.T) {
	logger := Logger{SyslogPriority: "warning"}
	assert.Equal(t, logger.GetSyslogPriority(), syslog.LOG_DAEMON|syslog.LOG_WARNING)
}

// TestSyslogPriorityWithInvalidPriority tests the GetSyslogPriority function with
// an invalid priority, which should return the default priority.
func TestSyslogPriorityWithInvalidPriority(t *testing.T) {
	logger := Logger{SyslogPriority: "invalid"}
	assert.Equal(t, logger.GetSyslogPriority(), syslog.LOG_DAEMON|syslog.LOG_INFO)
}
