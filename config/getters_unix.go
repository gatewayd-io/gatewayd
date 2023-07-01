//go:build !windows
// +build !windows

package config

import (
	"log/syslog"
	"syscall"

	"github.com/rs/zerolog"
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

// GetSystemLimits returns the current system limits or the configured limits.
func (s Server) GetRLimits(logger zerolog.Logger) (uint64, uint64) {
	var limits syscall.Rlimit
	if err := syscall.Getrlimit(syscall.RLIMIT_NOFILE, &limits); err != nil {
		logger.Debug().Msg("failed to get system limits")
	}

	if s.SoftLimit <= 0 && limits != (syscall.Rlimit{}) {
		s.SoftLimit = limits.Cur
		logger.Debug().Uint64("soft_limit", s.SoftLimit).Msg(
			"Soft limit is not set, using system limit")
	}

	if s.HardLimit <= 0 && limits != (syscall.Rlimit{}) {
		s.HardLimit = limits.Max
		logger.Debug().Uint64("hard_limit", s.HardLimit).Msg(
			"Hard limit is not set, using system limit")
	}

	return s.HardLimit, s.SoftLimit
}
