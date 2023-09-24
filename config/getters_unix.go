//go:build !windows
// +build !windows

package config

import (
	"log/syslog"
)

var rSyslogPriorities = map[string]syslog.Priority{
	"emerg":   syslog.LOG_EMERG,
	"alert":   syslog.LOG_ALERT,
	"crit":    syslog.LOG_CRIT,
	"err":     syslog.LOG_ERR,
	"warning": syslog.LOG_WARNING,
	"notice":  syslog.LOG_NOTICE,
	"info":    syslog.LOG_INFO,
	"debug":   syslog.LOG_DEBUG,
}

// GetSyslogPriority returns the rsyslog facility from config file.
func (l Logger) GetSyslogPriority() syslog.Priority {
	if priority, ok := rSyslogPriorities[l.SyslogPriority]; ok {
		return priority | syslog.LOG_DAEMON
	}
	return syslog.LOG_DAEMON | syslog.LOG_INFO
}
