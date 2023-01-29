package config

import (
	"time"
)

type (
	Status       uint
	Policy       uint
	CompatPolicy uint
	LogOutput    uint
)

const (
	Running Status = iota
	Stopped
)

const (
	// Non-strict (permissive) mode.
	PassDown Policy = iota // Pass down the extra keys/values in result to the next plugins
	// Strict mode.
	Ignore // Ignore errors and continue
	Abort  // Abort on first error and return results
	Remove // Remove the hook from the list on error and continue
)

const (
	Strict CompatPolicy = iota
	Loose
)

const (
	Console LogOutput = iota
	Stdout
	Stderr
	Buffer // Buffer the output and return it as a string (for testing).
	File
)

const (
	// Config constants.
	Default = "default"

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
	EmptyPoolCapacity = 0
	DefaultPoolSize   = 10
	MinimumPoolSize   = 2

	// Server constants.
	DefaultTickInterval = 5 * time.Second
	DefaultBufferSize   = 1 << 24 // 16777216 bytes
	DefaultTCPKeepAlive = 3 * time.Second

	// Utility constants.
	DefaultSeed        = 1000
	ChecksumBufferSize = 65536
)
