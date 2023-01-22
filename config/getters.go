package config

import (
	"log/syslog"
	"time"

	"github.com/panjf2000/gnet/v2"
	"github.com/rs/zerolog"
)

var (
	verificationPolicies = map[string]VerificationPolicy{
		"ignore":   Ignore,
		"abort":    Abort,
		"remove":   Remove,
		"passdown": PassDown,
	}
	compatibilityPolicies = map[string]CompatibilityPolicy{
		"strict": Strict,
		"loose":  Loose,
	}
	acceptancePolicies = map[string]AcceptancePolicy{
		"accept": Accept,
		"reject": Reject,
	}
	loadBalancers = map[string]gnet.LoadBalancing{
		"roundrobin":       gnet.RoundRobin,
		"leastconnections": gnet.LeastConnections,
		"sourceaddrhash":   gnet.SourceAddrHash,
	}
	logOutputs = map[string]LogOutput{
		"console": Console,
		"stdout":  Stdout,
		"stderr":  Stderr,
		"file":    File,
		"syslog":  Syslog,
		"rsyslog": RSyslog,
	}
	timeFormats = map[string]string{
		"unix":      zerolog.TimeFormatUnix,
		"unixms":    zerolog.TimeFormatUnixMs,
		"unixmicro": zerolog.TimeFormatUnixMicro,
		"unixnano":  zerolog.TimeFormatUnixNano,
	}
	consoleTimeFormats = map[string]string{
		"Layout":      time.Layout,
		"ANSIC":       time.ANSIC,
		"UnixDate":    time.UnixDate,
		"RubyDate":    time.RubyDate,
		"RFC822":      time.RFC822,
		"RFC822Z":     time.RFC822Z,
		"RFC850":      time.RFC850,
		"RFC1123":     time.RFC1123,
		"RFC1123Z":    time.RFC1123Z,
		"RFC3339":     time.RFC3339,
		"RFC3339Nano": time.RFC3339Nano,
		"Kitchen":     time.Kitchen,
		"Stamp":       time.Stamp,
		"StampMilli":  time.StampMilli,
		"StampMicro":  time.StampMicro,
		"StampNano":   time.StampNano,
	}
	logLevels = map[string]zerolog.Level{
		"trace":    zerolog.TraceLevel,
		"debug":    zerolog.DebugLevel,
		"info":     zerolog.InfoLevel,
		"warn":     zerolog.WarnLevel,
		"error":    zerolog.ErrorLevel,
		"fatal":    zerolog.FatalLevel,
		"panic":    zerolog.PanicLevel,
		"disabled": zerolog.Disabled,
	}
	//nolint:nosnakecase
	rSyslogPriorities = map[string]syslog.Priority{
		"emerg":   syslog.LOG_EMERG,
		"alert":   syslog.LOG_ALERT,
		"crit":    syslog.LOG_CRIT,
		"err":     syslog.LOG_ERR,
		"warning": syslog.LOG_WARNING,
		"notice":  syslog.LOG_NOTICE,
		"info":    syslog.LOG_INFO,
		"debug":   syslog.LOG_DEBUG,
	}
)

// GetVerificationPolicy returns the hook verification policy from plugin config file.
func (p PluginConfig) GetVerificationPolicy() VerificationPolicy {
	if policy, ok := verificationPolicies[p.VerificationPolicy]; ok {
		return policy
	}
	return PassDown
}

// GetPluginCompatibilityPolicy returns the plugin compatibility policy from plugin config file.
func (p PluginConfig) GetPluginCompatibilityPolicy() CompatibilityPolicy {
	if policy, ok := compatibilityPolicies[p.CompatibilityPolicy]; ok {
		return policy
	}
	return Strict
}

// GetAcceptancePolicy returns the acceptance policy from plugin config file.
func (p PluginConfig) GetAcceptancePolicy() AcceptancePolicy {
	if policy, ok := acceptancePolicies[p.AcceptancePolicy]; ok {
		return policy
	}
	return Accept
}

// GetLoadBalancer returns the load balancing algorithm to use.
func (s Server) GetLoadBalancer() gnet.LoadBalancing {
	if lb, ok := loadBalancers[s.LoadBalancer]; ok {
		return lb
	}
	return gnet.RoundRobin
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
		if logOutput, ok := logOutputs[output]; ok {
			outputs = append(outputs, logOutput)
		} else {
			outputs = append(outputs, Console)
		}
	}
	return outputs
}

// GetTimeFormat returns the logger time format from config file.
func (l Logger) GetTimeFormat() string {
	if format, ok := timeFormats[l.TimeFormat]; ok {
		return format
	}
	return zerolog.TimeFormatUnix
}

// GetConsoleTimeFormat returns the console logger's time format from config file.
func (l Logger) GetConsoleTimeFormat() string {
	if format, ok := consoleTimeFormats[l.ConsoleTimeFormat]; ok {
		return format
	}
	return time.RFC3339
}

// GetLevel returns the logger level from config file.
func (l Logger) GetLevel() zerolog.Level {
	if level, ok := logLevels[l.Level]; ok {
		return level
	}
	return zerolog.InfoLevel
}

// GetSyslogPriority returns the rsyslog facility from config file.
//
//nolint:nosnakecase
func (l Logger) GetSyslogPriority() syslog.Priority {
	if priority, ok := rSyslogPriorities[l.SyslogPriority]; ok {
		return priority | syslog.LOG_DAEMON
	}
	return DefaultSyslogPriority
}
