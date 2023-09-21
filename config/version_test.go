package config

import (
	"runtime"
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestVersionInfo tests the VersionInfo function.
func TestVersionInfo(t *testing.T) {
	versionInfo := VersionInfo()
	assert.Contains(t, versionInfo, "GatewayD")
	assert.Contains(t, versionInfo, "0.0.0")
	assert.Contains(t, versionInfo, "go")
	assert.Contains(t, versionInfo, runtime.GOOS)
	assert.Contains(t, versionInfo, runtime.GOARCH)
}
