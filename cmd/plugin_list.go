package cmd

import (
	"context"
	"strings"

	"github.com/gatewayd-io/gatewayd/config"
	"github.com/getsentry/sentry-go"
	"github.com/spf13/cobra"
)

var onlyEnabled bool

// pluginListCmd represents the plugin list command.
var pluginListCmd = &cobra.Command{
	Use:   "list",
	Short: "List the GatewayD plugins",
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

		listPlugins(cmd, App.PluginConfigFile, onlyEnabled)
	},
}

func init() {
	pluginCmd.AddCommand(pluginListCmd)

	App = &GatewayDInstance{}

	pluginListCmd.Flags().StringVarP(
		&App.PluginConfigFile, // Already exists in run.go
		"plugin-config", "p", config.GetDefaultConfigFilePath(config.PluginsConfigFilename),
		"Plugin config file")
	pluginListCmd.Flags().BoolVarP(
		&onlyEnabled,
		"only-enabled", "e",
		false, "Only list enabled plugins")
	pluginListCmd.Flags().BoolVar(
		&App.EnableSentry, "sentry", true, "Enable Sentry") // Already exists in run.go
}

func listPlugins(cmd *cobra.Command, pluginConfigFile string, onlyEnabled bool) {
	// Load the plugin config file.
	conf := config.NewConfig(context.TODO(), config.Config{PluginConfigFile: pluginConfigFile})
	if err := conf.LoadDefaults(context.TODO()); err != nil {
		cmd.PrintErr(err)
		return
	}
	if err := conf.LoadPluginConfigFile(context.TODO()); err != nil {
		cmd.PrintErr(err)
		return
	}
	if err := conf.UnmarshalPluginConfig(context.TODO()); err != nil {
		cmd.PrintErr(err)
		return
	}

	if len(conf.Plugin.Plugins) != 0 {
		cmd.Printf("Total plugins: %d\n", len(conf.Plugin.Plugins))
		cmd.Println("Plugins:")
	} else {
		cmd.Println("No plugins found")
	}

	// Print the list of plugins.
	for _, plugin := range conf.Plugin.Plugins {
		if onlyEnabled && !plugin.Enabled {
			continue
		}
		cmd.Printf("  Name: %s\n", plugin.Name)
		cmd.Printf("  Enabled: %t\n", plugin.Enabled)
		cmd.Printf("  Path: %s\n", plugin.LocalPath)
		cmd.Printf("  Args: %s\n", strings.Join(plugin.Args, " "))
		cmd.Println("  Env:")
		for _, env := range plugin.Env {
			cmd.Printf("    %s\n", env)
		}
		cmd.Printf("  Checksum: %s\n", plugin.Checksum)
	}
}
