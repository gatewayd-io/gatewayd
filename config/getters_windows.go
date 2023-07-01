//go:build windows
// +build windows

package config

// GetSyslogPriority returns zero value for Windows.
func (l Logger) GetSyslogPriority() int {
	return 0
}
