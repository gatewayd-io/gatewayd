package cmd

import (
	"os"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "gatewayd",
	Short: "A cloud-native database gateway and framework for building data-driven applications",
	Long:  `GatewayD is a cloud-native database gateway and framework for building data-driven applications. It sits in between your database(s) and your database client(s) and proxies all queries to and their responses from the database.`,
}

func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

func init() {}
