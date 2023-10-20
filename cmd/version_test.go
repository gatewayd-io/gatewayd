package cmd

import (
	"regexp"
	"testing"

	"github.com/gatewayd-io/gatewayd/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_versionCmd(t *testing.T) {
	// Test versionCmd with no arguments.
	config.Version = "SEMVER"
	config.VersionDetails = "COMMIT-HASH"
	output, err := executeCommandC(rootCmd, "version")
	require.NoError(t, err, "versionCmd should not return an error")
	assert.Regexp(t,
		// The regexp matches something like the following output:
		// GatewayD v0.7.7 (2023-09-16T19:27:38+0000/038f75b, go1.21.0, linux/amd64)
		regexp.MustCompile(`^GatewayD SEMVER \(COMMIT-HASH, go\d+\.\d+\.\d+, \w+/\w+\)\n$`),
		output,
		"versionCmd should print the correct output")
}
