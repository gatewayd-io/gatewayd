package config

import (
	"github.com/panjf2000/gnet/v2"
	"github.com/rs/zerolog"
)

// verificationPolicy returns the hook verification policy from plugin config file.
func (p PluginConfig) GetVerificationPolicy() Policy {
	// vPolicy := pluginConfig.String("plugins.verificationPolicy")
	verificationPolicy := PassDown // default
	switch p.VerificationPolicy {
	case "ignore":
		verificationPolicy = Ignore
	case "abort":
		verificationPolicy = Abort
	case "remove":
		verificationPolicy = Remove
	}

	return verificationPolicy
}

// pluginCompatPolicy returns the plugin compatibility policy from plugin config file.
func (p PluginConfig) GetPluginCompatPolicy() CompatPolicy {
	// vPolicy := pluginConfig.String("plugins.compatibilityPolicy")
	compatPolicy := Strict // default
	switch p.CompatibilityPolicy {
	case "strict":
		compatPolicy = Strict
	case "loose":
		compatPolicy = Loose
	}

	return compatPolicy
}

// loadBalancer returns the load balancing algorithm to use.
func (s Server) GetLoadBalancer() gnet.LoadBalancing {
	loadBalancer := map[string]gnet.LoadBalancing{
		"roundrobin":       gnet.RoundRobin,
		"leastconnections": gnet.LeastConnections,
		"sourceaddrhash":   gnet.SourceAddrHash,
	}

	if lb, ok := loadBalancer[s.LoadBalancer]; ok {
		return lb
	}

	return gnet.RoundRobin
}

// tcpNoDelay returns the TCP no delay option from config file.
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

// output returns the logger output from config file.
func (l Logger) GetOutput() LogOutput {
	switch l.Output {
	case "file":
		return File
	case "stdout":
		return Stdout
	case "stderr":
		return Stderr
	default:
		return Console
	}
}

// timeFormat returns the logger time format from config file.
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

// level returns the logger level from config file.
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
