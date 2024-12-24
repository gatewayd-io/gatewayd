//nolint:dupl
package cmd

import (
	"log"

	"github.com/gatewayd-io/gatewayd/config"
	"github.com/getsentry/sentry-go"
	"github.com/spf13/cobra"
)

// configLintCmd represents the config lint command.
var configLintCmd = &cobra.Command{
	Use:   "lint",
	Short: "Lint the GatewayD global config",
	Run: func(cmd *cobra.Command, _ []string) {
		enableSentry, _ := cmd.Flags().GetBool("sentry")
		globalConfigFile, _ := cmd.Flags().GetString("config")

		// Enable Sentry.
		if enableSentry {
			// Initialize Sentry.
			err := sentry.Init(sentry.ClientOptions{
				Dsn:              DSN,
				TracesSampleRate: config.DefaultTraceSampleRate,
				AttachStacktrace: config.DefaultAttachStacktrace,
			})
			if err != nil {
				cmd.Println("Sentry initialization failed: ", err)
				return
			}

			// Flush buffered events before the program terminates.
			defer sentry.Flush(config.DefaultFlushTimeout)
			// Recover from panics and report the error to Sentry.
			defer sentry.Recover()
		}

		if err := lintConfig(Global, globalConfigFile); err != nil {
			log.Fatal(err)
		}

		cmd.Println("global config is valid")
	},
}

func init() {
	configCmd.AddCommand(configLintCmd)

	configLintCmd.Flags().StringP(
		"config", "c", config.GetDefaultConfigFilePath(config.GlobalConfigFilename),
		"Global config file")
	configLintCmd.Flags().Bool("sentry", true, "Enable Sentry")
}
