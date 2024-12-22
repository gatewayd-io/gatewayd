package cmd

import (
	"github.com/gatewayd-io/gatewayd/config"
	"github.com/getsentry/sentry-go"
	"github.com/spf13/cobra"
)

var force bool

// configInitCmd represents the plugin init command.
var configInitCmd = &cobra.Command{
	Use:   "init",
	Short: "Create or overwrite the GatewayD global config",
	Run: func(cmd *cobra.Command, _ []string) {
		// Enable Sentry.
		if App.EnableSentry {
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

		generateConfig(cmd, Global, App.GlobalConfigFile, force)
	},
}

func init() {
	configCmd.AddCommand(configInitCmd)

	App = &GatewayDInstance{}

	configInitCmd.Flags().BoolVarP(
		&force, "force", "f", false, "Force overwrite of existing config file")
	configInitCmd.Flags().StringVarP(
		&App.GlobalConfigFile, // Already exists in run.go
		"config", "c", config.GetDefaultConfigFilePath(config.GlobalConfigFilename),
		"Global config file")
	configInitCmd.Flags().BoolVar(
		&App.EnableSentry, "sentry", true, "Enable Sentry") // Already exists in run.go
}
