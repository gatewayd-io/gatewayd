package cmd

import (
	"github.com/gatewayd-io/gatewayd/config"
	"github.com/getsentry/sentry-go"
	"github.com/spf13/cobra"
)

// pluginInitCmd represents the plugin init command.
var pluginInitCmd = &cobra.Command{
	Use:   "init",
	Short: "Create or overwrite the GatewayD plugins config",
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

		generateConfig(cmd, Plugins, App.PluginConfigFile, force)
	},
}

func init() {
	pluginCmd.AddCommand(pluginInitCmd)

	App = &GatewayDInstance{}

	pluginInitCmd.Flags().BoolVarP(
		&force, "force", "f", false, "Force overwrite of existing config file")
	pluginInitCmd.Flags().StringVarP(
		&App.PluginConfigFile, // Already exists in run.go
		"plugin-config", "p", config.GetDefaultConfigFilePath(config.PluginsConfigFilename),
		"Plugin config file")
	pluginInitCmd.Flags().BoolVar(
		&App.EnableSentry, "sentry", true, "Enable Sentry") // Already exists in run.go
}
