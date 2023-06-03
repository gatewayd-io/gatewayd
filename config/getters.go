package config

import (
	"log/syslog"
	"os"
	"path/filepath"
	"syscall"
	"time"

	"github.com/panjf2000/gnet/v2"
	"github.com/rs/zerolog"
)

var (
	verificationPolicies = map[string]VerificationPolicy{
		"passdown": PassDown,
		"ignore":   Ignore,
		"abort":    Abort,
		"remove":   Remove,
	}
	compatibilityPolicies = map[string]CompatibilityPolicy{
		"strict": Strict,
		"loose":  Loose,
	}
	acceptancePolicies = map[string]AcceptancePolicy{
		"accept": Accept,
		"reject": Reject,
	}
	terminationPolicies = map[string]TerminationPolicy{
		"continue": Continue,
		"stop":     Stop,
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
		"":          zerolog.TimeFormatUnix,
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

// GetTerminationPolicy returns the termination policy from plugin config file.
func (p PluginConfig) GetTerminationPolicy() TerminationPolicy {
	if policy, ok := terminationPolicies[p.TerminationPolicy]; ok {
		return policy
	}
	return Stop
}

// GetTCPKeepAlivePeriod returns the TCP keep alive period from config file or default value.
func (c Client) GetTCPKeepAlivePeriod() time.Duration {
	if c.TCPKeepAlivePeriod <= 0 {
		return DefaultTCPKeepAlivePeriod
	}
	return c.TCPKeepAlivePeriod
}

// GetReceiveDeadline returns the receive deadline from config file or default value.
func (c Client) GetReceiveDeadline() time.Duration {
	if c.ReceiveDeadline <= 0 {
		return DefaultReceiveDeadline
	}
	return c.ReceiveDeadline
}

// GetSendDeadline returns the send deadline from config file or default value.
func (c Client) GetSendDeadline() time.Duration {
	if c.SendDeadline <= 0 {
		return DefaultSendDeadline
	}
	return c.SendDeadline
}

// GetReceiveChunkSize returns the receive chunk size from config file or default value.
func (c Client) GetReceiveChunkSize() int {
	if c.ReceiveChunkSize <= 0 {
		return DefaultChunkSize
	}
	return c.ReceiveChunkSize
}

// GetHealthCheckPeriod returns the health check period from config file or default value.
func (pr Proxy) GetHealthCheckPeriod() time.Duration {
	if pr.HealthCheckPeriod <= 0 {
		return DefaultHealthCheckPeriod
	}
	return pr.HealthCheckPeriod
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

// GetTickInterval returns the tick interval from config file or default value.
func (s Server) GetTickInterval() time.Duration {
	if s.TickInterval <= 0 {
		return DefaultTickInterval
	}
	return s.TickInterval
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
func (l Logger) GetSyslogPriority() syslog.Priority {
	if priority, ok := rSyslogPriorities[l.SyslogPriority]; ok {
		return priority | syslog.LOG_DAEMON
	}
	return syslog.LOG_DAEMON | syslog.LOG_INFO
}

// GetPlugins returns the plugins from config file.
func (p PluginConfig) GetPlugins(name ...string) []Plugin {
	var plugins []Plugin
	for _, plugin := range p.Plugins {
		for _, n := range name {
			if plugin.Name == n {
				plugins = append(plugins, plugin)
			}
		}
	}
	return plugins
}

// GetDefaultConfigFilePath returns the path of the default config file.
func GetDefaultConfigFilePath(filename string) string {
	// Try to find the config file in the current directory.
	path := filepath.Join("./", filename)
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		return path
	}

	// Try to find the config file in the /etc directory.
	path = filepath.Join("/etc/", filename)
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		return path
	}

	// The fallback is the current directory.
	return filepath.Join("./", filename)
}
