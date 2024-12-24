package cmd

import (
	"github.com/gatewayd-io/gatewayd/config"
	"github.com/getsentry/sentry-go"
	"github.com/spf13/cobra"
)

// configInitCmd represents the plugin init command.
var configInitCmd = &cobra.Command{
	Use:   "init",
	Short: "Create or overwrite the GatewayD global config",
	Run: func(cmd *cobra.Command, _ []string) {
		force, _ := cmd.Flags().GetBool("force")
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

		generateConfig(cmd, Global, globalConfigFile, force)
	},
}

func init() {
	configCmd.AddCommand(configInitCmd)

	configInitCmd.Flags().BoolP(
		"force", "f", false, "Force overwrite of existing config file")
	configInitCmd.Flags().StringP(
		"config", "c", config.GetDefaultConfigFilePath(config.GlobalConfigFilename),
		"Global config file")
	configInitCmd.Flags().Bool("sentry", true, "Enable Sentry")
}
