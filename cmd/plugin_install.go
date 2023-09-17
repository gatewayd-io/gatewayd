package cmd

import (
	"context"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"

	"github.com/codingsince1985/checksum"
	"github.com/gatewayd-io/gatewayd/config"
	"github.com/getsentry/sentry-go"
	"github.com/google/go-github/v53/github"
	"github.com/spf13/cobra"
	yamlv3 "gopkg.in/yaml.v3"
)

const (
	NumParts                    int         = 2
	LatestVersion               string      = "latest"
	FolderPermissions           os.FileMode = 0o755
	DefaultPluginConfigFilename string      = "./gatewayd_plugin.yaml"
	GitHubURLPrefix             string      = "github.com/"
	GitHubURLRegex              string      = `^github.com\/[a-zA-Z0-9\-]+\/[a-zA-Z0-9\-]+@(?:latest|v(=|>=|<=|=>|=<|>|<|!=|~|~>|\^)?(?P<major>0|[1-9]\d*)\.(?P<minor>0|[1-9]\d*)\.(?P<patch>0|[1-9]\d*)(?:-(?P<prerelease>(?:0|[1-9]\d*|\d*[a-zA-Z-][0-9a-zA-Z-]*)(?:\.(?:0|[1-9]\d*|\d*[a-zA-Z-][0-9a-zA-Z-]*))*))?(?:\+(?P<buildmetadata>[0-9a-zA-Z-]+(?:\.[0-9a-zA-Z-]+)*))?)$` //nolint:lll
	ExtWindows                  string      = ".zip"
	ExtOthers                   string      = ".tar.gz"
)

var (
	pluginOutputDir string
	pullOnly        bool
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
				log.Panic("Sentry initialization failed: ", err)
			}

			// Flush buffered events before the program terminates.
			defer sentry.Flush(config.DefaultFlushTimeout)
			// Recover from panics and report the error to Sentry.
			defer sentry.Recover()
		}

		// Validate the number of arguments.
		if len(args) < 1 {
			log.Panic(
				"Invalid URL. Use the following format: github.com/account/repository@version")
		}

		var releaseID int64
		var downloadURL string
		var pluginFilename string
		var pluginName string
		var err error
		var checksumsFilename string
		var client *github.Client
		var account string

		if strings.HasPrefix(args[0], GitHubURLPrefix) {
			// Validate the URL.
			validGitHubURL := regexp.MustCompile(GitHubURLRegex)
			if !validGitHubURL.MatchString(args[0]) {
				log.Panic(
					"Invalid URL. Use the following format: github.com/account/repository@version")
			}

			// Get the plugin version.
			pluginVersion := LatestVersion
			splittedURL := strings.Split(args[0], "@")
			// If the version is not specified, use the latest version.
			if len(splittedURL) < NumParts {
				cmd.Println("Version not specified. Using latest version")
			}
			if len(splittedURL) >= NumParts {
				pluginVersion = splittedURL[1]
			}

			// Get the plugin account and repository.
			accountRepo := strings.Split(strings.TrimPrefix(splittedURL[0], GitHubURLPrefix), "/")
			if len(accountRepo) != NumParts {
				log.Panic(
					"Invalid URL. Use the following format: github.com/account/repository@version")
			}
			account = accountRepo[0]
			pluginName = accountRepo[1]
			if account == "" || pluginName == "" {
				log.Panic(
					"Invalid URL. Use the following format: github.com/account/repository@version")
			}

			// Get the release artifact from GitHub.
			client = github.NewClient(nil)
			var release *github.RepositoryRelease

			if pluginVersion == LatestVersion || pluginVersion == "" {
				// Get the latest release.
				release, _, err = client.Repositories.GetLatestRelease(
					context.Background(), account, pluginName)
			} else if strings.HasPrefix(pluginVersion, "v") {
				// Get an specific release.
				release, _, err = client.Repositories.GetReleaseByTag(
					context.Background(), account, pluginName, pluginVersion)
			}
			if err != nil {
				log.Panic("The plugin could not be found")
			}

			if release == nil {
				log.Panic("The plugin could not be found")
			}

			// Get the archive extension.
			archiveExt := ExtOthers
			if runtime.GOOS == "windows" {
				archiveExt = ExtWindows
			}

			// Find and download the plugin binary from the release assets.
			pluginFilename, downloadURL, releaseID = findAsset(release, func(name string) bool {
				return strings.Contains(name, runtime.GOOS) &&
					strings.Contains(name, runtime.GOARCH) &&
					strings.Contains(name, archiveExt)
			})
			if downloadURL != "" && releaseID != 0 {
				cmd.Println("Downloading", downloadURL)
				downloadFile(client, account, pluginName, downloadURL, releaseID, pluginFilename)
				cmd.Println("Download completed successfully")
			} else {
				log.Panic("The plugin file could not be found in the release assets")
			}

			// Find and download the checksums.txt from the release assets.
			checksumsFilename, downloadURL, releaseID = findAsset(release, func(name string) bool {
				return strings.Contains(name, "checksums.txt")
			})
			if checksumsFilename != "" && downloadURL != "" && releaseID != 0 {
				cmd.Println("Downloading", downloadURL)
				downloadFile(client, account, pluginName, downloadURL, releaseID, checksumsFilename)
				cmd.Println("Download completed successfully")
			} else {
				log.Panic("The checksum file could not be found in the release assets")
			}

			// Read the checksums text file.
			checksums, err := os.ReadFile(checksumsFilename)
			if err != nil {
				log.Panic("There was an error reading the checksums file: ", err)
			}

			// Get the checksum for the plugin binary.
			sum, err := checksum.SHA256sum(pluginFilename)
			if err != nil {
				log.Panic("There was an error calculating the checksum: ", err)
			}

			// Verify the checksums.
			checksumLines := strings.Split(string(checksums), "\n")
			for _, line := range checksumLines {
				if strings.Contains(line, pluginFilename) {
					checksum := strings.Split(line, " ")[0]
					if checksum != sum {
						log.Panic("Checksum verification failed")
					}

					cmd.Println("Checksum verification passed")
					break
				}
			}

			if pullOnly {
				cmd.Println("Plugin binary downloaded to", pluginFilename)
				return
			}
		} else {
			// Pull the plugin from a local archive.
			pluginFilename = filepath.Clean(args[0])
			if _, err := os.Stat(pluginFilename); os.IsNotExist(err) {
				log.Panic("The plugin file could not be found")
			}
		}

		// Extract the archive.
		var filenames []string
		if runtime.GOOS == "windows" {
			filenames = extractZip(pluginFilename, pluginOutputDir)
		} else {
			filenames = extractTarGz(pluginFilename, pluginOutputDir)
		}

		// Find the extracted plugin binary.
		localPath := ""
		pluginFileSum := ""
		for _, filename := range filenames {
			if strings.Contains(filename, pluginName) {
				cmd.Println("Plugin binary extracted to", filename)
				localPath = filename
				// Get the checksum for the extracted plugin binary.
				// TODO: Should we verify the checksum using the checksum.txt file instead?
				pluginFileSum, err = checksum.SHA256sum(filename)
				if err != nil {
					log.Panic("There was an error calculating the checksum: ", err)
				}
				break
			}
		}

		// TODO: Clean up after installing the plugin.
		// https://github.com/gatewayd-io/gatewayd/issues/311

		// Create a new gatewayd_plugins.yaml file if it doesn't exist.
		if _, err := os.Stat(pluginConfigFile); os.IsNotExist(err) {
			generateConfig(cmd, Plugins, pluginConfigFile, false)
		}

		// Read the gatewayd_plugins.yaml file.
		pluginsConfig, err := os.ReadFile(pluginConfigFile)
		if err != nil {
			log.Panic(err)
		}

		// Get the registered plugins from the plugins configuration file.
		var localPluginsConfig map[string]interface{}
		if err := yamlv3.Unmarshal(pluginsConfig, &localPluginsConfig); err != nil {
			log.Panic("Failed to unmarshal the plugins configuration file: ", err)
		}
		pluginsList, ok := localPluginsConfig["plugins"].([]interface{}) //nolint:varnamelen
		if !ok {
			log.Panic("There was an error reading the plugins file from disk")
		}

		var contents string
		if strings.HasPrefix(args[0], GitHubURLPrefix) {
			// Get the list of files in the repository.
			var repoContents *github.RepositoryContent
			repoContents, _, _, err = client.Repositories.GetContents(
				context.Background(), account, pluginName, DefaultPluginConfigFilename, nil)
			if err != nil {
				log.Panic("There was an error getting the default plugins configuration file: ", err)
			}
			// Get the contents of the file.
			contents, err = repoContents.GetContent()
			if err != nil {
				log.Panic("There was an error getting the default plugins configuration file: ", err)
			}
		} else {
			// Get the contents of the file.
			contentsBytes, err := os.ReadFile(
				filepath.Join(pluginOutputDir, DefaultPluginConfigFilename))
			if err != nil {
				log.Panic("There was an error getting the default plugins configuration file: ", err)
			}
			contents = string(contentsBytes)
		}

		// Get the plugin configuration from the downloaded plugins configuration file.
		var downloadedPluginConfig map[string]interface{}
		if err := yamlv3.Unmarshal([]byte(contents), &downloadedPluginConfig); err != nil {
			log.Panic("Failed to unmarshal the downloaded plugins configuration file: ", err)
		}
		defaultPluginConfig, ok := downloadedPluginConfig["plugins"].([]interface{})
		if !ok {
			log.Panic("There was an error reading the plugins file from the repository")
		}
		// Get the plugin configuration.
		pluginConfig, ok := defaultPluginConfig[0].(map[string]interface{})
		if !ok {
			log.Panic("There was an error reading the default plugin configuration")
		}

		// Update the plugin's local path and checksum.
		pluginConfig["localPath"] = localPath
		pluginConfig["checksum"] = pluginFileSum

		// TODO: Check if the plugin is already installed.
		// https://github.com/gatewayd-io/gatewayd/issues/312

		// Add the plugin config to the list of plugin configs.
		pluginsList = append(pluginsList, pluginConfig)
		// Merge the result back into the config map.
		localPluginsConfig["plugins"] = pluginsList

		// Marshal the map into YAML.
		updatedPlugins, err := yamlv3.Marshal(localPluginsConfig)
		if err != nil {
			log.Panic("There was an error marshalling the plugins configuration: ", err)
		}

		// Write the YAML to the plugins config file.
		if err = os.WriteFile(pluginConfigFile, updatedPlugins, FilePermissions); err != nil {
			log.Panic("There was an error writing the plugins configuration file: ", err)
		}

		// TODO: Add a rollback mechanism.
		cmd.Println("Plugin installed successfully")
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
		&enableSentry, "sentry", true, "Enable Sentry") // Already exists in run.go
}
