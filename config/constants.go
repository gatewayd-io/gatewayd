package config

import (
	"time"
)

type (
	Status              uint
	CompatibilityPolicy string
	LogOutput           uint
)

// Status is the status of the server.
const (
	Running Status = iota
	Stopped
)

// CompatibilityPolicy is the compatibility policy for plugins.
const (
	Strict CompatibilityPolicy = "strict" // Expect all required plugins to be loaded and present
	Loose  CompatibilityPolicy = "loose"  // Load the plugin, even if the requirements are not met
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
	Default                   = "default"
	DefaultConfigurationBlock = "writes"
	EnvPrefix                 = "GATEWAYD_"
	TracerName                = "gatewayd"
	GlobalConfigFilename      = "gatewayd.yaml"
	PluginsConfigFilename     = "gatewayd_plugins.yaml"

	// Logger constants.
	DefaultLogOutput         = "console"
	DefaultLogFileName       = "gatewayd.log"
	DefaultLogLevel          = "info"
	DefaultNoColor           = false
	DefaultTimeFormat        = "unix"
	DefaultConsoleTimeFormat = "RFC3339"
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
	DefaultPluginAddress           = "http://plugins/metrics"
	DefaultMetricsMergerPeriod     = 5 * time.Second
	DefaultPluginHealthCheckPeriod = 5 * time.Second
	DefaultPluginTimeout           = 30 * time.Second
	DefaultPluginStartTimeout      = 1 * time.Minute

	// Client constants.
	DefaultNetwork            = "tcp"
	DefaultAddress            = "localhost:5432"
	DefaultChunkSize          = 8192
	DefaultReceiveDeadline    = 0 // 0 means no deadline (timeout)
	DefaultSendDeadline       = 0
	DefaultTCPKeepAlivePeriod = 30 * time.Second
	DefaultTCPKeepAlive       = false
	DefaultReceiveTimeout     = 0
	DefaultDialTimeout        = 60 * time.Second
	DefaultRetries            = 3
	DefaultBackoff            = 1 * time.Second
	DefaultBackoffMultiplier  = 2.0
	DefaultDisableBackoffCaps = false

	// Pool constants.
	EmptyPoolCapacity        = 0
	DefaultPoolSize          = 10
	MinimumPoolSize          = 2
	DefaultHealthCheckPeriod = 60 * time.Second // This must match PostgreSQL authentication timeout.

	// Server constants.
	DefaultListenNetwork         = "tcp"
	DefaultListenAddress         = "0.0.0.0:15432"
	DefaultTickInterval          = 5 * time.Second
	DefaultHandshakeTimeout      = 5 * time.Second
	DefaultLoadBalancerStrategy  = "ROUND_ROBIN"
	DefaultLoadBalancerCondition = "DEFAULT"

	// Utility constants.
	DefaultSeed = 1000

	// Metrics constants.
	DefaultMetricsAddress       = "localhost:9090"
	DefaultMetricsPath          = "/metrics"
	DefaultReadHeaderTimeout    = 10 * time.Second
	DefaultMetricsServerTimeout = 10 * time.Second

	// Sentry constants.
	DefaultTraceSampleRate  = 0.2
	DefaultAttachStacktrace = true
	DefaultFlushTimeout     = 2 * time.Second

	// API constants.
	DefaultHTTPAPIAddress = "localhost:18080"
	DefaultGRPCAPINetwork = "tcp"
	DefaultGRPCAPIAddress = "localhost:19090"

	// Policies.
	DefaultCompatibilityPolicy = Strict

	// Act.
	DefaultPolicy             = "passthrough"
	DefaultPolicyTimeout      = 30 * time.Second
	DefaultActionTimeout      = 30 * time.Second
	DefaultActionRedisEnabled = false
	DefaultRedisAddress       = "localhost:6379"
	DefaultRedisChannel       = "gatewayd-actions"
)

// Load balancing strategies.
const (
	RoundRobinStrategy         = "ROUND_ROBIN"
	RANDOMStrategy             = "RANDOM"
	WeightedRoundRobinStrategy = "WEIGHTED_ROUND_ROBIN"
)
