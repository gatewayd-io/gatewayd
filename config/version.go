package config

import (
	"fmt"
	"runtime"
	"runtime/debug"
)

const Name = "GatewayD"

var (
	// Version is the semantic version of GatewayD.
	Version = "0.0.0"
	// VersionDetails is the build timestamp and the tagged commit hash.
	VersionDetails = ""
)

// VersionInfo returns the full version and build information for
// the currently running GatewayD executable.
func VersionInfo() string {
	goVersionInfo := fmt.Sprintf("%s, %s/%s", runtime.Version(), runtime.GOOS, runtime.GOARCH)
	if VersionDetails != "" {
		return fmt.Sprintf("%s %s (%s, %s)", Name, Version, VersionDetails, goVersionInfo)
	}

	if buildInfo, ok := debug.ReadBuildInfo(); ok {
		return fmt.Sprintf("%s %s (%s, %s)", Name, Version, buildInfo.Main.Version, goVersionInfo)
	}

	return fmt.Sprintf("%s %s (dev build, %s)", Name, Version, goVersionInfo)
}
