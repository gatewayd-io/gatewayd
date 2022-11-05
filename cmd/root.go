package cmd

import (
	"os"

	"github.com/spf13/cobra"
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "gatewayd",
	Short: "A cloud-native database gateway and framework for building data-driven applications",
	Long:  `GatewayD is a cloud-native database gateway and framework for building data-driven applications. It sits in between your database(s) and your database client(s) and proxies all queries to and their responses from the database.`,
	// Uncomment the following line if your bare application
	// has an action associated with it:
	// Run: func(cmd *cobra.Command, args []string) { },
}

func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

func init() {
	rootCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}
