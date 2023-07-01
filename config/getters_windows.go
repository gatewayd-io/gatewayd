//go:build windows
// +build windows

package config

import "github.com/rs/zerolog"

// GetSyslogPriority returns zero value for Windows.
func (l Logger) GetSyslogPriority() int {
	return 0
}

// GetSystemLimits returns zero values for Windows.
func (s Server) GetRLimits(logger zerolog.Logger) (uint64, uint64) {
	return 0, 0
}
