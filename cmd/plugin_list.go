package cmd

import (
	"log"

	"github.com/gatewayd-io/gatewayd/config"
	"github.com/getsentry/sentry-go"
	"github.com/spf13/cobra"
)

var onlyEnabled bool

// pluginListCmd represents the plugin list command.
var pluginListCmd = &cobra.Command{
	Use:   "list",
	Short: "List the GatewayD plugins",
	Run: func(cmd *cobra.Command, args []string) {
		// Enable Sentry.
		if enableSentry {
			// Initialize Sentry.
			err := sentry.Init(sentry.ClientOptions{
				Dsn:              DSN,
				TracesSampleRate: config.DefaultTraceSampleRate,
				AttachStacktrace: config.DefaultAttachStacktrace,
			})
			if err != nil {
				log.Fatal("Sentry initialization failed: ", err)
			}

			// Flush buffered events before the program terminates.
			defer sentry.Flush(config.DefaultFlushTimeout)
			// Recover from panics and report the error to Sentry.
			defer sentry.Recover()
		}

		listPlugins(cmd, pluginConfigFile, onlyEnabled)
	},
}

func init() {
	pluginCmd.AddCommand(pluginListCmd)

	pluginListCmd.Flags().StringVarP(
		&pluginConfigFile, // Already exists in run.go
		"plugin-config", "p", config.GetDefaultConfigFilePath(config.PluginsConfigFilename),
		"Plugin config file")
	pluginListCmd.Flags().BoolVarP(
		&onlyEnabled,
		"only-enabled", "e",
		false, "Only list enabled plugins")
	pluginListCmd.Flags().BoolVar(
		&enableSentry, "sentry", true, "Enable Sentry") // Already exists in run.go
}
