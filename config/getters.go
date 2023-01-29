package config

import (
	"log/syslog"
	"time"

	"github.com/panjf2000/gnet/v2"
	"github.com/rs/zerolog"
)

// GetVerificationPolicy returns the hook verification policy from plugin config file.
func (p PluginConfig) GetVerificationPolicy() VerificationPolicy {
	switch p.VerificationPolicy {
	case "ignore":
		return Ignore
	case "abort":
		return Abort
	case "remove":
		return Remove
	default:
		return PassDown
	}
}

// GetPluginCompatibilityPolicy returns the plugin compatibility policy from plugin config file.
func (p PluginConfig) GetPluginCompatibilityPolicy() CompatibilityPolicy {
	switch p.CompatibilityPolicy {
	case "strict":
		return Strict
	case "loose":
		return Loose
	default:
		return Strict
	}
}

// GetAcceptancePolicy returns the acceptance policy from plugin config file.
func (p PluginConfig) GetAcceptancePolicy() AcceptancePolicy {
	switch p.AcceptancePolicy {
	case "accept":
		return Accept
	case "reject":
		return Reject
	default:
		return Accept
	}
}

// GetLoadBalancer returns the load balancing algorithm to use.
func (s Server) GetLoadBalancer() gnet.LoadBalancing {
	switch s.LoadBalancer {
	case "roundrobin":
		return gnet.RoundRobin
	case "leastconnections":
		return gnet.LeastConnections
	case "sourceaddrhash":
		return gnet.SourceAddrHash
	default:
		return gnet.RoundRobin
	}
}

// GetTCPNoDelay returns the TCP no delay option from config file.
func (s Server) GetTCPNoDelay() gnet.TCPSocketOpt {
	if s.TCPNoDelay {
		return gnet.TCPNoDelay
	}

	return gnet.TCPDelay
}

// GetSize returns the pool size from config file.
func (p Pool) GetSize() int {
	if p.Size == 0 {
		return DefaultPoolSize
	}

	// Minimum pool size is 2.
	if p.Size < MinimumPoolSize {
		p.Size = MinimumPoolSize
	}

	return p.Size
}

// GetOutput returns the logger output from config file.
func (l Logger) GetOutput() []LogOutput {
	var outputs []LogOutput
	for _, output := range l.Output {
		switch output {
		case "file":
			outputs = append(outputs, File)
		case "stdout":
			outputs = append(outputs, Stdout)
		case "stderr":
			outputs = append(outputs, Stderr)
		case "syslog":
			outputs = append(outputs, Syslog)
		case "rsyslog":
			outputs = append(outputs, RSyslog)
		default:
			outputs = append(outputs, Console)
		}
	}
	return outputs
}

// GetTimeFormat returns the logger time format from config file.
func (l Logger) GetTimeFormat() string {
	switch l.TimeFormat {
	case "unixms":
		return zerolog.TimeFormatUnixMs
	case "unixmicro":
		return zerolog.TimeFormatUnixMicro
	case "unixnano":
		return zerolog.TimeFormatUnixNano
	case "unix":
		return zerolog.TimeFormatUnix
	default:
		return zerolog.TimeFormatUnix
	}
}

// GetConsoleTimeFormat returns the console logger's time format from config file.
func (l Logger) GetConsoleTimeFormat() string {
	switch l.ConsoleTimeFormat {
	case "Layout":
		return time.Layout
	case "ANSIC":
		return time.ANSIC
	case "UnixDate":
		return time.UnixDate
	case "RubyDate":
		return time.RubyDate
	case "RFC822":
		return time.RFC822
	case "RFC822Z":
		return time.RFC822Z
	case "RFC850":
		return time.RFC850
	case "RFC1123":
		return time.RFC1123
	case "RFC1123Z":
		return time.RFC1123Z
	case "RFC3339":
		return time.RFC3339
	case "RFC3339Nano":
		return time.RFC3339Nano
	case "Kitchen":
		return time.Kitchen
	case "Stamp":
		return time.Stamp
	case "StampMilli":
		return time.StampMilli
	case "StampMicro":
		return time.StampMicro
	case "StampNano":
		return time.StampNano
	default:
		return time.RFC3339
	}
}

// GetLevel returns the logger level from config file.
func (l Logger) GetLevel() zerolog.Level {
	switch l.Level {
	case "debug":
		return zerolog.DebugLevel
	case "info":
		return zerolog.InfoLevel
	case "warn":
		return zerolog.WarnLevel
	case "error":
		return zerolog.ErrorLevel
	case "fatal":
		return zerolog.FatalLevel
	case "panic":
		return zerolog.PanicLevel
	case "disabled":
		return zerolog.Disabled
	case "trace":
		return zerolog.TraceLevel
	default:
		return zerolog.InfoLevel
	}
}

// GetRSyslogPriority returns the rsyslog facility from config file.
//
//nolint:nosnakecase
func (l Logger) GetRSyslogPriority() syslog.Priority {
	switch l.RSyslogPriority {
	case "emerg":
		return syslog.LOG_EMERG | syslog.LOG_DAEMON
	case "alert":
		return syslog.LOG_ALERT | syslog.LOG_DAEMON
	case "crit":
		return syslog.LOG_CRIT | syslog.LOG_DAEMON
	case "err":
		return syslog.LOG_ERR | syslog.LOG_DAEMON
	case "warning":
		return syslog.LOG_WARNING | syslog.LOG_DAEMON
	case "notice":
		return syslog.LOG_NOTICE | syslog.LOG_DAEMON
	case "info":
		return syslog.LOG_INFO | syslog.LOG_DAEMON
	case "debug":
		return syslog.LOG_DEBUG | syslog.LOG_DAEMON
	default:
		return DefaultRSyslogPriority
	}
}
