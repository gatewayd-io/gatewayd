package config

import (
	"time"

	"github.com/rs/zerolog"
)

type (
	Status              uint
	VerificationPolicy  uint
	CompatibilityPolicy uint
	AcceptancePolicy    uint
	LogOutput           uint
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

// CompatibilityPolicy is the compatibility policy for plugins.
const (
	Strict CompatibilityPolicy = iota // Expect all required plugins to be loaded and present
	Loose                             // Load the plugin, even if the requirements are not met
)

// AcceptancePolicy is the acceptance policy for custom hooks.
const (
	Accept AcceptancePolicy = iota // Accept all custom hooks
	Reject                         // Reject all custom hooks
)

// LogOutput is the output type for the logger.
const (
	Console LogOutput = iota
	Stdout
	Stderr
	File
	Syslog
	RSyslog
)

const (
	// Config constants.
	Default    = "default"
	EnvPrefix  = "GATEWAYD_"
	TracerName = "gatewayd"

	// Logger constants.
	DefaultLogOutput         = "console"
	DefaultLogFileName       = "gatewayd.log"
	DefaultLogLevel          = "info"
	DefaultTimeFormat        = zerolog.TimeFormatUnix
	DefaultConsoleTimeFormat = time.RFC3339
	DefaultMaxSize           = 500 // megabytes
	DefaultMaxBackups        = 5
	DefaultMaxAge            = 30 // days
	DefaultCompress          = true
	DefaultLocalTime         = false
	DefaultSyslogTag         = "gatewayd"
	DefaultRSyslogNetwork    = "tcp"
	DefaultRSyslogAddress    = "localhost:514"
	DefaultSyslogPriority    = "info"

	// Plugin constants.
	DefaultMinPort                 = 50000
	DefaultMaxPort                 = 60000
	PluginPriorityStart            = 1000
	LoggerName                     = "plugin"
	DefaultPluginAddress           = "http://plugins/metrics"
	DefaultMetricsMergerPeriod     = 5 * time.Second
	DefaultPluginHealthCheckPeriod = 5 * time.Second
	DefaultPluginTimeout           = 30 * time.Second

	// Client constants.
	DefaultChunkSize          = 8192
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

	// Metrics constants.
	DefaultMetricsAddress = "localhost:2112"
	DefaultMetricsPath    = "/metrics"

	// Sentry constants.
	DefaultTraceSampleRate  = 0.2
	DefaultAttachStacktrace = true
	DefaultFlushTimeout     = 2 * time.Second

	// API constants.
	DefaultHTTPAPIAddress = "localhost:8080"
	DefaultGRPCAPINetwork = "tcp"
	DefaultGRPCAPIAddress = "localhost:9090"
)
