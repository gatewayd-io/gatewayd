package cmd

import (
	"os"

	"github.com/gatewayd-io/gatewayd/config"
	"github.com/getsentry/sentry-go"
	"github.com/spf13/cast"
	"github.com/spf13/cobra"
	yamlv3 "gopkg.in/yaml.v3"
)

type (
	Location  string
	Source    string
	Extension string
)

const (
	NumParts                    int         = 2
	LatestVersion               string      = "latest"
	FolderPermissions           os.FileMode = 0o755
	DefaultPluginConfigFilename string      = "./gatewayd_plugin.yaml"
	GitHubURLPrefix             string      = "github.com/"
	GitHubURLRegex              string      = `^github.com\/[a-zA-Z0-9\-]+\/[a-zA-Z0-9\-]+@(?:latest|v(=|>=|<=|=>|=<|>|<|!=|~|~>|\^)?(?P<major>0|[1-9]\d*)\.(?P<minor>0|[1-9]\d*)\.(?P<patch>0|[1-9]\d*)(?:-(?P<prerelease>(?:0|[1-9]\d*|\d*[a-zA-Z-][0-9a-zA-Z-]*)(?:\.(?:0|[1-9]\d*|\d*[a-zA-Z-][0-9a-zA-Z-]*))*))?(?:\+(?P<buildmetadata>[0-9a-zA-Z-]+(?:\.[0-9a-zA-Z-]+)*))?)$` //nolint:lll
	LocationArgs                Location    = "args"
	LocationConfig              Location    = "config"
	SourceUnknown               Source      = "unknown"
	SourceFile                  Source      = "file"
	SourceGitHub                Source      = "github"
	ExtensionZip                Extension   = ".zip"
	ExtensionTarGz              Extension   = ".tar.gz"
)

var (
	pluginOutputDir string
	pullOnly        bool
	cleanup         bool
	update          bool
	backupConfig    bool
	noPrompt        bool
	pluginName      string
)

// pluginInstallCmd represents the plugin install command.
var pluginInstallCmd = &cobra.Command{
	Use:     "install",
	Short:   "Install a plugin from a local archive or a GitHub repository",
	Example: "  gatewayd plugin install github.com/gatewayd-io/gatewayd-plugin-cache@latest",
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
				cmd.Println("Sentry initialization failed: ", err)
				return
			}

			// Flush buffered events before the program terminates.
			defer sentry.Flush(config.DefaultFlushTimeout)
			// Recover from panics and report the error to Sentry.
			defer sentry.Recover()
		}

		switch detectInstallLocation(args) {
		case LocationArgs:
			// Install the plugin from the CLI argument.
			cmd.Println("Installing plugin from CLI argument")
			installPlugin(cmd, args[0])
		case LocationConfig:
			// Read the gatewayd_plugins.yaml file.
			pluginsConfig, err := os.ReadFile(pluginConfigFile)
			if err != nil {
				cmd.Println(err)
				return
			}

			// Get the registered plugins from the plugins configuration file.
			var localPluginsConfig map[string]interface{}
			if err := yamlv3.Unmarshal(pluginsConfig, &localPluginsConfig); err != nil {
				cmd.Println("Failed to unmarshal the plugins configuration file: ", err)
				return
			}
			pluginsList := cast.ToSlice(localPluginsConfig["plugins"])

			// Get the list of plugin download URLs.
			var pluginURLs []string
			for _, plugin := range pluginsList {
				// Get the plugin instance.
				pluginInstance := cast.ToStringMapString(plugin)

				// Append the plugin URL to the list of plugin URLs.
				name := cast.ToString(pluginInstance["name"])
				url := cast.ToString(pluginInstance["url"])
				if url != "" {
					pluginURLs = append(pluginURLs, url)
				} else {
					cmd.Println("Plugin URL or file path not found in the plugins configuration file for", name)
					return
				}
			}

			// Validate the plugin URLs.
			if len(args) == 0 && len(pluginURLs) == 0 {
				cmd.Println(
					"No plugin URLs or file path found in the plugins configuration file or CLI argument")
				return
			}

			// Install all the plugins from the plugins configuration file.
			cmd.Println("Installing plugins from plugins configuration file")
			for _, pluginURL := range pluginURLs {
				installPlugin(cmd, pluginURL)
			}
		default:
			cmd.Println("Invalid plugin URL or file path")
		}
	},
}

func init() {
	pluginCmd.AddCommand(pluginInstallCmd)

	pluginInstallCmd.Flags().StringVarP(
		&pluginConfigFile, // Already exists in run.go
		"plugin-config", "p", config.GetDefaultConfigFilePath(config.PluginsConfigFilename),
		"Plugin config file")
	pluginInstallCmd.Flags().StringVarP(
		&pluginOutputDir, "output-dir", "o", "./plugins", "Output directory for the plugin")
	pluginInstallCmd.Flags().BoolVar(
		&pullOnly, "pull-only", false, "Only pull the plugin, don't install it")
	pluginInstallCmd.Flags().BoolVar(
		&cleanup, "cleanup", true,
		"Delete downloaded and extracted files after installing the plugin (except the plugin binary)")
	pluginInstallCmd.Flags().BoolVar(
		&noPrompt, "no-prompt", true, "Do not prompt for user input")
	pluginInstallCmd.Flags().BoolVar(
		&update, "update", false, "Update the plugin if it already exists")
	pluginInstallCmd.Flags().BoolVar(
		&backupConfig, "backup", false, "Backup the plugins configuration file before installing the plugin")
	pluginInstallCmd.Flags().StringVarP(
		&pluginName, "name", "n", "", "Name of the plugin")
	pluginInstallCmd.Flags().BoolVar(
		&enableSentry, "sentry", true, "Enable Sentry") // Already exists in run.go
}
