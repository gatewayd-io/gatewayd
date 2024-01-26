package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/gatewayd-io/gatewayd/config"
	"github.com/getsentry/sentry-go"
	"github.com/spf13/cast"
	"github.com/spf13/cobra"
	"golang.org/x/exp/maps"
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
	overwriteConfig bool
)

// pluginInstallCmd represents the plugin install command.
var pluginInstallCmd = &cobra.Command{
	Use:     "install",
	Short:   "Install a plugin from a local archive or a GitHub repository",
	Example: "  gatewayd plugin install <github.com/gatewayd-io/gatewayd-plugin-cache@latest|/path/to/plugin[.zip|.tar.gz]>", //nolint:lll
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
			pluginURLs := map[string]string{}
			existingPluginURLs := map[string]string{}
			for _, plugin := range pluginsList {
				// Get the plugin instance.
				pluginInstance := cast.ToStringMapString(plugin)

				// Append the plugin URL to the list of plugin URLs.
				name := cast.ToString(pluginInstance["name"])
				url := cast.ToString(pluginInstance["url"])
				if url == "" {
					cmd.Println("Plugin URL or file path not found in the plugins configuration file for", name)
					return
				}

				// Check if duplicate plugin names exist in the plugins configuration file.
				if _, ok := pluginURLs[name]; ok {
					cmd.Println("Duplicate plugin name found in the plugins configuration file:", name)
					return
				}

				// Update list of plugin URLs based on
				// whether the plugin is already installed or not.
				localPath := cast.ToString(pluginInstance["localPath"])
				if _, err := os.Stat(localPath); err == nil {
					existingPluginURLs[name] = url
				} else {
					pluginURLs[name] = url
				}
			}

			// Check if the plugin is already installed and prompt the user to confirm the update.
			if len(existingPluginURLs) > 0 {
				pluginNames := strings.Join(maps.Keys[map[string]string](existingPluginURLs), ", ")
				cmd.Printf("The following plugins are already installed: %s\n", pluginNames)

				if noPrompt {
					if !update {
						cmd.Println("Use the --update flag to update the plugins")
						cmd.Println("Aborting...")
						return
					}

					// Merge the existing plugin URLs with the plugin URLs.
					for name, url := range existingPluginURLs {
						pluginURLs[name] = url
					}
				} else {
					cmd.Print("Do you want to update the existing plugins? [y/N] ")
					var response string
					_, err := fmt.Scanln(&response)
					if err == nil && strings.ToLower(response) == "y" {
						// Set the update flag to true, so that the installPlugin function
						// can update the existing plugins and doesn't ask for user input again.
						update = true

						// Merge the existing plugin URLs with the plugin URLs.
						for name, url := range existingPluginURLs {
							pluginURLs[name] = url
						}
					} else {
						cmd.Println("Existing plugins will not be updated")
					}
				}
			}

			// Validate the plugin URLs.
			if len(args) == 0 && len(pluginURLs) == 0 {
				if len(existingPluginURLs) > 0 && !update {
					cmd.Println("Use the --update flag to update the plugins")
				} else {
					cmd.Println(
						"No plugin URLs or file path found in the plugins configuration file or CLI argument")
					cmd.Println("Aborting...")
				}
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
		&overwriteConfig, "overwrite-config", true, "Overwrite the existing plugins configuration file")
	pluginInstallCmd.Flags().BoolVar(
		&enableSentry, "sentry", true, "Enable Sentry") // Already exists in run.go
}
