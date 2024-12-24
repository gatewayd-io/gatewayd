//nolint:dupl
package cmd

import (
	"log"

	"github.com/gatewayd-io/gatewayd/config"
	"github.com/getsentry/sentry-go"
	"github.com/spf13/cobra"
)

// pluginLintCmd represents the plugin lint command.
var pluginLintCmd = &cobra.Command{
	Use:   "lint",
	Short: "Lint the GatewayD plugins config",
	Run: func(cmd *cobra.Command, _ []string) {
		enableSentry, _ := cmd.Flags().GetBool("sentry")
		pluginConfigFile, _ := cmd.Flags().GetString("plugin-config")

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

		if err := lintConfig(Plugins, pluginConfigFile); err != nil {
			log.Fatal(err)
		}

		cmd.Println("plugins config is valid")
	},
}

func init() {
	pluginCmd.AddCommand(pluginLintCmd)

	pluginLintCmd.Flags().StringP(
		"plugin-config", "p", config.GetDefaultConfigFilePath(config.PluginsConfigFilename),
		"Plugin config file")
	pluginLintCmd.Flags().BoolP(
		"sentry", "s", true, "Enable Sentry")
}
