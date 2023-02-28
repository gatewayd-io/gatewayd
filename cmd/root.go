package cmd

import (
	"os"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "gatewayd",
	Short: "A cloud-native database gateway and framework for building data-driven applications",
	Long:  `GatewayD is a cloud-native database gateway and framework for building data-driven applications. It sits between your database servers and clients and proxies all their communication.`, //nolint:lll
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}
