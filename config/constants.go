package config

import "time"

type (
	Status             uint
	VerificationPolicy uint
	CompatPolicy       uint
	LogOutput          uint
)

// Status is the status of the server.
const (
	Running Status = iota
	Stopped
)

// Policy is the policy for hook verification.
const (
	// Non-strict (permissive) mode.
	PassDown VerificationPolicy = iota // Pass down the extra keys/values in result to the next plugins
	// Strict mode.
	Ignore // Ignore errors and continue
	Abort  // Abort on first error and return results
	Remove // Remove the hook from the list on error and continue
)

// CompatPolicy is the compatibility policy for plugins.
const (
	Strict CompatPolicy = iota
	Loose
)

// LogOutput is the output type for the logger.
const (
	Console LogOutput = iota
	Stdout
	Stderr
	Buffer // Buffer the output and return it as a string (for testing).
	File
)

const (
	// Config constants.
	Default   = "default"
	EnvPrefix = "GATEWAYD_"

	// Logger constants.
	DefaultLogFileName       = "gatewayd.log"
	DefaultLogFilePermission = 0o660
	DefaultLogOutput         = "console"
	DefaultLogLevel          = "info"

	// Plugin constants.
	DefaultMinPort      = 50000
	DefaultMaxPort      = 60000
	PluginPriorityStart = 1000
	LoggerName          = "plugin"

	// Client constants.
	DefaultChunkSize          = 4096
	DefaultReceiveDeadline    = 0 // 0 means no deadline (timeout)
	DefaultSendDeadline       = 0
	DefaultTCPKeepAlivePeriod = 30 * time.Second

	// Pool constants.
	EmptyPoolCapacity        = 0
	DefaultPoolSize          = 10
	MinimumPoolSize          = 2
	DefaultHealthCheckPeriod = 60 * time.Second // This must match PostgreSQL authentication timeout.

	// Server constants.
	DefaultListenNetwork = "tcp"
	DefaultListenAddress = "0.0.0.0:15432"
	DefaultTickInterval  = 5 * time.Second
	DefaultBufferSize    = 1 << 24 // 16777216 bytes
	DefaultTCPKeepAlive  = 3 * time.Second
	DefaultLoadBalancer  = "roundrobin"

	// Utility constants.
	DefaultSeed        = 1000
	ChecksumBufferSize = 65536
)
