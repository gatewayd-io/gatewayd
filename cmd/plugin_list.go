package cmd

import (
	"context"
	"strings"

	"github.com/gatewayd-io/gatewayd/config"
	"github.com/getsentry/sentry-go"
	"github.com/spf13/cobra"
)

// pluginListCmd represents the plugin list command.
var pluginListCmd = &cobra.Command{
	Use:   "list",
	Short: "List the GatewayD plugins",
	Run: func(cmd *cobra.Command, _ []string) {
		pluginConfigFile, _ := cmd.Flags().GetString("plugin-config")
		onlyEnabled, _ := cmd.Flags().GetBool("only-enabled")
		enableSentry, _ := cmd.Flags().GetBool("sentry")

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

		listPlugins(cmd, pluginConfigFile, onlyEnabled)
	},
}

func init() {
	pluginCmd.AddCommand(pluginListCmd)

	pluginListCmd.Flags().StringP(
		"plugin-config", "p", config.GetDefaultConfigFilePath(config.PluginsConfigFilename),
		"Plugin config file")
	pluginListCmd.Flags().BoolP(
		"only-enabled", "e",
		false, "Only list enabled plugins")
	pluginListCmd.Flags().BoolP("sentry", "s", true, "Enable Sentry")
}

func listPlugins(cmd *cobra.Command, pluginConfigFile string, onlyEnabled bool) {
	// Load the plugin config file.
	conf := config.NewConfig(context.Background(), config.Config{PluginConfigFile: pluginConfigFile})
	if err := conf.LoadDefaults(context.Background()); err != nil {
		cmd.PrintErr(err)
		return
	}
	if err := conf.LoadPluginConfigFile(context.Background()); err != nil {
		cmd.PrintErr(err)
		return
	}
	if err := conf.UnmarshalPluginConfig(context.Background()); err != nil {
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
