package cmd

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"runtime"
	"slices"
	"strings"

	"github.com/codingsince1985/checksum"
	"github.com/gatewayd-io/gatewayd/config"
	gerr "github.com/gatewayd-io/gatewayd/errors"
	"github.com/getsentry/sentry-go"
	"github.com/google/go-github/v53/github"
	"github.com/spf13/cast"
	"github.com/spf13/cobra"
	"golang.org/x/exp/maps"
	yamlv3 "gopkg.in/yaml.v3"
)

type (
	Location       string
	Source         string
	Extension      string
	configFileType string
)

const (
	NumParts                    int            = 2
	LatestVersion               string         = "latest"
	FolderPermissions           os.FileMode    = 0o755
	FilePermissions             os.FileMode    = 0o644
	ExecFilePermissions         os.FileMode    = 0o755
	ExecFileMask                os.FileMode    = 0o111
	MaxFileSize                 int64          = 1024 * 1024 * 100 // 100 MB
	BackupFileExt               string         = ".bak"
	DefaultPluginConfigFilename string         = "./gatewayd_plugin.yaml"
	GitHubURLPrefix             string         = "github.com/"
	GitHubURLRegex              string         = `^github.com\/[a-zA-Z0-9\-]+\/[a-zA-Z0-9\-]+@(?:latest|v(=|>=|<=|=>|=<|>|<|!=|~|~>|\^)?(?P<major>0|[1-9]\d*)\.(?P<minor>0|[1-9]\d*)\.(?P<patch>0|[1-9]\d*)(?:-(?P<prerelease>(?:0|[1-9]\d*|\d*[a-zA-Z-][0-9a-zA-Z-]*)(?:\.(?:0|[1-9]\d*|\d*[a-zA-Z-][0-9a-zA-Z-]*))*))?(?:\+(?P<buildmetadata>[0-9a-zA-Z-]+(?:\.[0-9a-zA-Z-]+)*))?)$` //nolint:lll
	LocationArgs                Location       = "args"
	LocationConfig              Location       = "config"
	SourceUnknown               Source         = "unknown"
	SourceFile                  Source         = "file"
	SourceGitHub                Source         = "github"
	ExtensionZip                Extension      = ".zip"
	ExtensionTarGz              Extension      = ".tar.gz"
	Global                      configFileType = "global"
	Plugins                     configFileType = "plugins"
)

// pluginInstallCmd represents the plugin install command.
var pluginInstallCmd = &cobra.Command{
	Use:     "install",
	Short:   "Install a plugin from a local archive or a GitHub repository",
	Example: "  gatewayd plugin install <github.com/gatewayd-io/gatewayd-plugin-cache@latest|/path/to/plugin[.zip|.tar.gz]>", //nolint:lll
	Run: func(cmd *cobra.Command, args []string) {
		enableSentry, _ := cmd.Flags().GetBool("sentry")
		pluginConfigFile, _ := cmd.Flags().GetString("plugin-config")
		pluginOutputDir, _ := cmd.Flags().GetString("output-dir")
		pullOnly, _ := cmd.Flags().GetBool("pull-only")
		cleanup, _ := cmd.Flags().GetBool("cleanup")
		noPrompt, _ := cmd.Flags().GetBool("no-prompt")
		update, _ := cmd.Flags().GetBool("update")
		backupConfig, _ := cmd.Flags().GetBool("backup")
		overwriteConfig, _ := cmd.Flags().GetBool("overwrite-config")
		pluginName, _ := cmd.Flags().GetString("name")
		skipPathSlipVerification, _ := cmd.Flags().GetBool("skip-path-slip-verification")

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
			installPlugin(
				cmd,
				args[0],
				pluginOutputDir,
				pullOnly,
				cleanup,
				noPrompt,
				update,
				backupConfig,
				overwriteConfig,
				skipPathSlipVerification,
				pluginConfigFile,
				pluginName,
			)
		case LocationConfig:
			// Read the gatewayd_plugins.yaml file.
			pluginsConfig, err := os.ReadFile(pluginConfigFile)
			if err != nil {
				cmd.Println(err)
				return
			}

			// Backup the config file if requested
			if backupConfig {
				backupFilename := pluginConfigFile + BackupFileExt
				if err := os.WriteFile(backupFilename, pluginsConfig, FilePermissions); err != nil {
					cmd.Println("There was an error backing up the plugins configuration file: ", err)
					return
				}
				cmd.Println("Backup completed successfully")
			}

			// Get the registered plugins from the plugins configuration file.
			var localPluginsConfig map[string]any
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
				installPlugin(
					cmd,
					pluginURL,
					pluginOutputDir,
					pullOnly,
					cleanup,
					noPrompt,
					update,
					backupConfig,
					overwriteConfig,
					skipPathSlipVerification,
					pluginConfigFile,
					pluginName,
				)
			}
		default:
			cmd.Println("Invalid plugin URL or file path")
		}
	},
}

func init() {
	pluginCmd.AddCommand(pluginInstallCmd)

	pluginInstallCmd.Flags().StringP(
		"plugin-config", "p", config.GetDefaultConfigFilePath(config.PluginsConfigFilename),
		"Plugin config file")
	pluginInstallCmd.Flags().StringP(
		"output-dir", "o", "./plugins", "Output directory for the plugin")
	pluginInstallCmd.Flags().Bool(
		"pull-only", false, "Only pull the plugin, don't install it")
	pluginInstallCmd.Flags().Bool(
		"cleanup", true,
		"Delete downloaded and extracted files after installing the plugin (except the plugin binary)")
	pluginInstallCmd.Flags().Bool(
		"no-prompt", true, "Do not prompt for user input")
	pluginInstallCmd.Flags().Bool(
		"update", false, "Update the plugin if it already exists")
	pluginInstallCmd.Flags().Bool(
		"backup", false, "Backup the plugins configuration file before installing the plugin")
	pluginInstallCmd.Flags().StringP(
		"name", "n", "", "Name of the plugin (only for installing from archive files)")
	pluginInstallCmd.Flags().Bool(
		"overwrite-config", true, "Overwrite the existing plugins configuration file (overrides --update, only used for installing from the plugins configuration file)") //nolint:lll
	pluginInstallCmd.Flags().Bool(
		"skip-path-slip-verification", false, "Skip path slip verification when extracting the plugin archive from a TRUSTED source") //nolint:lll
	pluginInstallCmd.Flags().Bool(
		"sentry", true, "Enable Sentry")
}

// extractZip extracts the files from a zip archive.
func extractZip(
	filename, dest string, skipPathSlipVerification bool,
) ([]string, *gerr.GatewayDError) {
	// Open and extract the zip file.
	zipRc, err := zip.OpenReader(filename)
	if err != nil {
		return nil, gerr.ErrExtractFailed.Wrap(err)
	}
	defer zipRc.Close()

	// Create the output directory if it doesn't exist.
	if err := os.MkdirAll(dest, FolderPermissions); err != nil {
		return nil, gerr.ErrExtractFailed.Wrap(err)
	}

	// Extract the files.
	var filenames []string
	for _, fileOrDir := range zipRc.File {
		switch fileInfo := fileOrDir.FileInfo(); {
		case fileInfo.IsDir():
			// Sanitize the path.
			dirName := filepath.Clean(fileOrDir.Name)
			if !path.IsAbs(dirName) {
				// Create the directory.
				destPath := path.Join(dest, dirName)
				if err := os.MkdirAll(destPath, FolderPermissions); err != nil {
					return nil, gerr.ErrExtractFailed.Wrap(err)
				}
			}
		case fileInfo.Mode().IsRegular():
			// Sanitize the path.
			outFilename := filepath.Join(filepath.Clean(dest), filepath.Clean(fileOrDir.Name))

			// Check for ZipSlip.
			if !skipPathSlipVerification &&
				strings.HasPrefix(outFilename, string(os.PathSeparator)) {
				return nil, gerr.ErrExtractFailed.Wrap(
					fmt.Errorf("illegal file path: %s", outFilename))
			}

			// Create the file.
			outFile, err := os.Create(outFilename)
			if err != nil {
				return nil, gerr.ErrExtractFailed.Wrap(err)
			}
			defer outFile.Close()

			// Open the file in the zip archive.
			fileRc, err := fileOrDir.Open()
			if err != nil {
				os.Remove(outFilename)
				return nil, gerr.ErrExtractFailed.Wrap(err)
			}

			// Copy the file contents.
			if _, err := io.Copy(outFile, io.LimitReader(fileRc, MaxFileSize)); err != nil {
				os.Remove(outFilename)
				return nil, gerr.ErrExtractFailed.Wrap(err)
			}

			fileMode := fileOrDir.FileInfo().Mode()
			// Set the file permissions.
			if fileMode.IsRegular() && fileMode&ExecFileMask != 0 {
				if err := os.Chmod(outFilename, ExecFilePermissions); err != nil {
					return nil, gerr.ErrExtractFailed.Wrap(err)
				}
			} else {
				if err := os.Chmod(outFilename, FilePermissions); err != nil {
					return nil, gerr.ErrExtractFailed.Wrap(err)
				}
			}

			filenames = append(filenames, outFile.Name())
		default:
			return nil, gerr.ErrExtractFailed.Wrap(
				fmt.Errorf("unknown file type: %s", fileOrDir.Name))
		}
	}

	return filenames, nil
}

// extractTarGz extracts the files from a tar.gz archive.
func extractTarGz(
	filename, dest string, skipPathSlipVerification bool,
) ([]string, *gerr.GatewayDError) {
	// Open and extract the tar.gz file.
	gzipStream, err := os.Open(filename)
	if err != nil {
		return nil, gerr.ErrExtractFailed.Wrap(err)
	}
	defer gzipStream.Close()

	uncompressedStream, err := gzip.NewReader(gzipStream)
	if err != nil {
		return nil, gerr.ErrExtractFailed.Wrap(err)
	}
	defer uncompressedStream.Close()

	// Create the output directory if it doesn't exist.
	if err := os.MkdirAll(dest, FolderPermissions); err != nil {
		return nil, gerr.ErrExtractFailed.Wrap(err)
	}

	tarReader := tar.NewReader(uncompressedStream)
	var filenames []string

	for {
		header, err := tarReader.Next()

		if errors.Is(err, io.EOF) {
			break
		}

		if err != nil {
			return nil, gerr.ErrExtractFailed.Wrap(err)
		}

		switch header.Typeflag {
		case tar.TypeDir:
			// Sanitize the path
			cleanPath := filepath.Clean(header.Name)
			// Ensure it is not an absolute path
			if !path.IsAbs(cleanPath) {
				destPath := path.Join(dest, cleanPath)
				if err := os.MkdirAll(destPath, FolderPermissions); err != nil {
					return nil, gerr.ErrExtractFailed.Wrap(err)
				}
			}
		case tar.TypeReg:
			// Sanitize the path
			outFilename := path.Join(filepath.Clean(dest), filepath.Clean(header.Name))

			// Check for TarSlip.
			if !skipPathSlipVerification &&
				strings.HasPrefix(outFilename, string(os.PathSeparator)) {
				return nil, gerr.ErrExtractFailed.Wrap(
					fmt.Errorf("illegal file path: %s", outFilename))
			}

			// Create the file.
			outFile, err := os.Create(outFilename)
			if err != nil {
				return nil, gerr.ErrExtractFailed.Wrap(err)
			}
			defer outFile.Close()

			if _, err := io.Copy(outFile, io.LimitReader(tarReader, MaxFileSize)); err != nil {
				os.Remove(outFilename)
				return nil, gerr.ErrExtractFailed.Wrap(err)
			}

			fileMode := header.FileInfo().Mode()
			// Set the file permissions
			if fileMode.IsRegular() && fileMode&ExecFileMask != 0 {
				if err := os.Chmod(outFilename, ExecFilePermissions); err != nil {
					return nil, gerr.ErrExtractFailed.Wrap(err)
				}
			} else {
				if err := os.Chmod(outFilename, FilePermissions); err != nil {
					return nil, gerr.ErrExtractFailed.Wrap(err)
				}
			}

			filenames = append(filenames, outFile.Name())
		default:
			return nil, gerr.ErrExtractFailed.Wrap(
				fmt.Errorf("unknown file type: %s", header.Name))
		}
	}

	return filenames, nil
}

// findAsset finds the release asset that matches the given criteria in the release.
func findAsset(release *github.RepositoryRelease, match func(string) bool) (string, string, int64) {
	if release == nil {
		return "", "", 0
	}

	// Find the matching release.
	for _, asset := range release.Assets {
		if match(asset.GetName()) {
			return asset.GetName(), asset.GetBrowserDownloadURL(), asset.GetID()
		}
	}
	return "", "", 0
}

// downloadFile downloads the plugin from the given GitHub URL from the release assets.
func downloadFile(
	client *github.Client,
	account, pluginName string,
	releaseID int64,
	filename, outputDir string,
) (string, *gerr.GatewayDError) {
	// Download the plugin.
	readCloser, redirectURL, err := client.Repositories.DownloadReleaseAsset(
		context.Background(), account, pluginName, releaseID, http.DefaultClient)
	if err != nil {
		return "", gerr.ErrDownloadFailed.Wrap(err)
	}
	defer readCloser.Close()

	if redirectURL != "" {
		// Download the plugin from the redirect URL.
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		req, err := http.NewRequestWithContext(ctx, http.MethodGet, redirectURL, nil)
		if err != nil {
			return "", gerr.ErrDownloadFailed.Wrap(err)
		}

		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			return "", gerr.ErrDownloadFailed.Wrap(err)
		}
		defer resp.Body.Close()

		readCloser = resp.Body
	}

	if readCloser == nil {
		return "", gerr.ErrDownloadFailed.Wrap(
			fmt.Errorf("unable to download file: %s", filename))
	}

	// Create the output file in the current directory and write the downloaded content.
	var filePath string
	if outputDir == "" || !filepath.IsAbs(outputDir) {
		cwd, err := os.Getwd()
		if err != nil {
			return "", gerr.ErrDownloadFailed.Wrap(err)
		}
		filePath = path.Join([]string{cwd, filename}...)
	} else {
		filePath = path.Join([]string{outputDir, filename}...)
	}

	output, err := os.Create(filePath)
	if err != nil {
		return "", gerr.ErrDownloadFailed.Wrap(err)
	}
	defer output.Close()

	// Write the bytes to the file.
	_, err = io.Copy(output, readCloser)
	if err != nil {
		return "", gerr.ErrDownloadFailed.Wrap(err)
	}

	return filePath, nil
}

// deleteFiles deletes the files in the toBeDeleted list.
func deleteFiles(toBeDeleted []string) {
	for _, filename := range toBeDeleted {
		if err := os.Remove(filename); err != nil {
			fmt.Println("There was an error deleting the file: ", err) //nolint:forbidigo
			return
		}
	}
}

// detectInstallLocation detects the installation location based on the number of arguments.
func detectInstallLocation(args []string) Location {
	if len(args) == 0 {
		return LocationConfig
	}

	return LocationArgs
}

// detectSource detects the source of the path.
func detectSource(path string) Source {
	if _, err := os.Stat(path); err == nil {
		return SourceFile
	}

	// Check if the path is a URL.
	if strings.HasPrefix(path, "http://") || strings.HasPrefix(path, "https://") || strings.HasPrefix(path, GitHubURLPrefix) { //nolint:lll
		return SourceGitHub
	}

	return SourceUnknown
}

// getFileExtension returns the extension of the archive based on the OS.
func getFileExtension() Extension {
	if runtime.GOOS == "windows" {
		return ExtensionZip
	}

	return ExtensionTarGz
}

// installPlugin installs a plugin from a given URL.
func installPlugin(
	cmd *cobra.Command,
	pluginURL string,
	outputDir string,
	pullOnly bool,
	cleanup bool,
	noPrompt bool,
	update bool,
	backupConfig bool,
	overwriteConfig bool,
	skipPathSlipVerification bool,
	pluginConfigFile string,
	pluginName string,
) {
	var (
		// This is a list of files that will be deleted after the plugin is installed.
		toBeDeleted []string

		// Source of the plugin: file or GitHub.
		source = detectSource(pluginURL)

		// The extension of the archive based on the OS: .zip or .tar.gz.
		archiveExt = getFileExtension()

		releaseID         int64
		downloadURL       string
		pluginFilename    string
		checksumsFilename string
		account           string
		err               error
		client            *github.Client
	)

	switch source {
	case SourceFile:
		// Pull the plugin from a local archive.
		pluginFilename = filepath.Clean(pluginURL)
		if _, err := os.Stat(pluginFilename); os.IsNotExist(err) {
			cmd.Println("The plugin file could not be found")
			return
		}

		if pluginName == "" {
			cmd.Println("Plugin name not specified")
			return
		}
	case SourceGitHub:
		// Strip scheme from the plugin URL.
		pluginURL = strings.TrimPrefix(strings.TrimPrefix(pluginURL, "http://"), "https://")

		// Validate the URL.
		splittedURL := strings.Split(pluginURL, "@")
		if len(splittedURL) < NumParts {
			if pluginFilename == "" {
				// If the version is not specified, use the latest version.
				pluginURL = fmt.Sprintf("%s@%s", pluginURL, LatestVersion)
			}
		}

		validGitHubURL := regexp.MustCompile(GitHubURLRegex)
		if !validGitHubURL.MatchString(pluginURL) {
			cmd.Println(
				"Invalid URL. Use the following format: github.com/account/repository@version")
			return
		}

		// Get the plugin version.
		pluginVersion := LatestVersion
		splittedURL = strings.Split(pluginURL, "@")
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
			cmd.Println(
				"Invalid URL. Use the following format: github.com/account/repository@version")
			return
		}
		account = accountRepo[0]
		pluginName = accountRepo[1]
		if account == "" || pluginName == "" {
			cmd.Println(
				"Invalid URL. Use the following format: github.com/account/repository@version")
			return
		}

		// Get the release artifact from GitHub.
		client = github.NewClient(nil)
		var release *github.RepositoryRelease

		if pluginVersion == LatestVersion || pluginVersion == "" {
			// Get the latest release.
			release, _, err = client.Repositories.GetLatestRelease(
				context.Background(), account, pluginName)
		} else if strings.HasPrefix(pluginVersion, "v") {
			// Get a specific release.
			release, _, err = client.Repositories.GetReleaseByTag(
				context.Background(), account, pluginName, pluginVersion)
		}

		if err != nil {
			cmd.Println("The plugin could not be found: ", err.Error())
			return
		}

		if release == nil {
			cmd.Println("The plugin could not be found in the release assets")
			return
		}

		// Create the output directory if it doesn't exist.
		if err := os.MkdirAll(outputDir, FolderPermissions); err != nil {
			cmd.Println("There was an error creating the output directory: ", err)
			return
		}

		// Find and download the plugin binary from the release assets.
		pluginFilename, downloadURL, releaseID = findAsset(release, func(name string) bool {
			return strings.Contains(name, runtime.GOOS) &&
				strings.Contains(name, runtime.GOARCH) &&
				strings.Contains(name, string(archiveExt))
		})
		if downloadURL != "" && releaseID != 0 {
			cmd.Println("Downloading", downloadURL)
			filePath, gErr := downloadFile(
				client, account, pluginName, releaseID, pluginFilename, outputDir)
			toBeDeleted = append(toBeDeleted, filePath)
			if gErr != nil {
				cmd.Println("Download failed: ", gErr)
				if cleanup {
					deleteFiles(toBeDeleted)
				}
				return
			}
			cmd.Println("File downloaded to", filePath)
			cmd.Println("Download completed successfully")
		} else {
			cmd.Println("The plugin file could not be found in the release assets")
			return
		}

		// Find and download the checksums.txt from the release assets.
		checksumsFilename, downloadURL, releaseID = findAsset(release, func(name string) bool {
			return strings.Contains(name, "checksums.txt")
		})
		if checksumsFilename != "" && downloadURL != "" && releaseID != 0 {
			cmd.Println("Downloading", downloadURL)
			filePath, gErr := downloadFile(
				client, account, pluginName, releaseID, checksumsFilename, outputDir)
			toBeDeleted = append(toBeDeleted, filePath)
			if gErr != nil {
				cmd.Println("Download failed: ", gErr)
				if cleanup {
					deleteFiles(toBeDeleted)
				}
				return
			}
			cmd.Println("File downloaded to", filePath)
			cmd.Println("Download completed successfully")
		} else {
			cmd.Println("The checksum file could not be found in the release assets")
			return
		}

		// Read the checksums text file.
		checksums, err := os.ReadFile(checksumsFilename)
		if err != nil {
			cmd.Println("There was an error reading the checksums file: ", err)
			return
		}

		// Get the checksum for the plugin binary.
		sum, err := checksum.SHA256sum(pluginFilename)
		if err != nil {
			cmd.Println("There was an error calculating the checksum: ", err)
			return
		}

		// Verify the checksums.
		checksumLines := strings.Split(string(checksums), "\n")
		for _, line := range checksumLines {
			if strings.Contains(line, pluginFilename) {
				checksumPart := strings.Split(line, " ")[0]
				if checksumPart != sum {
					cmd.Println("Checksum verification failed")
					return
				}

				cmd.Println("Checksum verification passed")
				break
			}
		}

		if pullOnly {
			cmd.Println("Plugin binary downloaded to", pluginFilename)
			// Only delete the checksums file if the --pull-only flag is set
			if cleanup {
				if err := os.Remove(checksumsFilename); err != nil {
					cmd.Println("There was an error deleting the checksums file: ", err)
				}
			}
			return
		}
	case SourceUnknown:
		fallthrough
	default:
		cmd.Println("Invalid URL or file path")
		return
	}

	// NOTE: The rest of the code is executed regardless of the source,
	// since the plugin binary is already available (or downloaded) at this point.

	// Create a new "gatewayd_plugins.yaml" file if it doesn't exist.
	if _, err := os.Stat(pluginConfigFile); os.IsNotExist(err) {
		generateConfig(cmd, Plugins, pluginConfigFile, false)
	} else if !backupConfig && !noPrompt {
		// If the config file exists, we should prompt the user to back up
		// the plugins configuration file.
		cmd.Print("Do you want to backup the plugins configuration file? [Y/n] ")
		var backupOption string
		_, err := fmt.Scanln(&backupOption)
		if err == nil && strings.ToLower(backupOption) == "n" {
			backupConfig = false
		} else {
			backupConfig = true
		}
	}

	// Read the "gatewayd_plugins.yaml" file.
	pluginsConfig, err := os.ReadFile(pluginConfigFile)
	if err != nil {
		cmd.Println(err)
		return
	}

	// Get the registered plugins from the plugins configuration file.
	var localPluginsConfig map[string]any
	if err := yamlv3.Unmarshal(pluginsConfig, &localPluginsConfig); err != nil {
		cmd.Println("Failed to unmarshal the plugins configuration file: ", err)
		return
	}
	pluginsList := cast.ToSlice(localPluginsConfig["plugins"])

	// Check if the plugin is already installed.
	for _, plugin := range pluginsList {
		// User already chosen to update the plugin using the --update CLI flag.
		if update {
			break
		}

		pluginInstance := cast.ToStringMap(plugin)
		if pluginInstance["name"] == pluginName {
			// Show a list of options to the user.
			cmd.Println("Plugin is already installed.")
			if !noPrompt {
				cmd.Print("Do you want to update the plugin? [y/N] ")

				var updateOption string
				_, err := fmt.Scanln(&updateOption)
				if err != nil && strings.ToLower(updateOption) == "y" {
					break
				}
			}

			cmd.Println("Aborting...")
			if cleanup {
				deleteFiles(toBeDeleted)
			}
			return
		}
	}

	// Check if the user wants to take a backup of the plugins configuration file.
	if backupConfig {
		backupFilename := pluginConfigFile + BackupFileExt
		if err := os.WriteFile(backupFilename, pluginsConfig, FilePermissions); err != nil {
			cmd.Println("There was an error backing up the plugins configuration file: ", err)
		}
		cmd.Println("Backup completed successfully")
	}

	// Extract the archive.
	var filenames []string
	var gErr *gerr.GatewayDError
	switch archiveExt {
	case ExtensionZip:
		filenames, gErr = extractZip(pluginFilename, outputDir, skipPathSlipVerification)
	case ExtensionTarGz:
		filenames, gErr = extractTarGz(pluginFilename, outputDir, skipPathSlipVerification)
	default:
		cmd.Println("Invalid archive extension")
		return
	}

	if gErr != nil {
		cmd.Println("There was an error extracting the plugin archive:", gErr)
		if cleanup {
			deleteFiles(toBeDeleted)
		}
		return
	}

	// Delete all the files except the extracted plugin binary,
	// which will be deleted from the list further down.
	toBeDeleted = append(toBeDeleted, filenames...)

	// Find the extracted plugin binary.
	localPath := ""
	pluginFileSum := ""
	for _, filename := range filenames {
		if strings.Contains(filename, pluginName) {
			cmd.Println("Plugin binary extracted to", filename)

			// Remove the plugin binary from the list of files to be deleted.
			toBeDeleted = slices.DeleteFunc[[]string, string](toBeDeleted, func(s string) bool {
				return s == filename
			})

			localPath = filename
			// Get the checksum for the extracted plugin binary.
			// TODO: Should we verify the checksum using the checksum.txt file instead?
			pluginFileSum, err = checksum.SHA256sum(filename)
			if err != nil {
				cmd.Println("There was an error calculating the checksum: ", err)
				return
			}
			break
		}
	}

	var contents string
	if source == SourceGitHub {
		// Get the list of files in the repository.
		var repoContents *github.RepositoryContent
		repoContents, _, _, err = client.Repositories.GetContents(
			context.Background(), account, pluginName, DefaultPluginConfigFilename, nil)
		if err != nil {
			cmd.Println(
				"There was an error getting the default plugins configuration file: ", err)
			return
		}
		// Get the contents of the file.
		contents, err = repoContents.GetContent()
		if err != nil {
			cmd.Println(
				"There was an error getting the default plugins configuration file: ", err)
			return
		}
	} else {
		// Get the contents of the file.
		contentsBytes, err := os.ReadFile(
			filepath.Join(outputDir, DefaultPluginConfigFilename))
		if err != nil {
			cmd.Println(
				"There was an error getting the default plugins configuration file: ", err)
			return
		}
		contents = string(contentsBytes)
	}

	// Get the plugin configuration from the downloaded plugins configuration file.
	var downloadedPluginConfig map[string]any
	if err := yamlv3.Unmarshal([]byte(contents), &downloadedPluginConfig); err != nil {
		cmd.Println("Failed to unmarshal the downloaded plugins configuration file: ", err)
		return
	}
	defaultPluginConfig := cast.ToSlice(downloadedPluginConfig["plugins"])

	// Get the plugin configuration.
	pluginConfig := cast.ToStringMap(defaultPluginConfig[0])

	// Update the plugin's local path and checksum.
	pluginConfig["localPath"] = localPath
	pluginConfig["checksum"] = pluginFileSum

	// Add the plugin config to the list of plugin configs.
	added := false
	for idx, plugin := range pluginsList {
		pluginInstance := cast.ToStringMap(plugin)
		if pluginInstance["name"] == pluginName {
			pluginsList[idx] = pluginConfig
			added = true
			break
		}
	}
	if !added {
		pluginsList = append(pluginsList, pluginConfig)
	}

	// Merge the result back into the config map.
	localPluginsConfig["plugins"] = pluginsList

	// Marshal the map into YAML.
	updatedPlugins, err := yamlv3.Marshal(localPluginsConfig)
	if err != nil {
		cmd.Println("There was an error marshalling the plugins configuration: ", err)
		return
	}

	// Write the YAML to the plugins config file if the --overwrite-config flag is set.
	if overwriteConfig {
		if err = os.WriteFile(pluginConfigFile, updatedPlugins, FilePermissions); err != nil {
			cmd.Println("There was an error writing the plugins configuration file: ", err)
			return
		}
	}

	// Delete the downloaded and extracted files, except the plugin binary,
	// if the --cleanup flag is set.
	if cleanup {
		deleteFiles(toBeDeleted)
	}

	// TODO: Add a rollback mechanism.
	cmd.Println("Plugin installed successfully")
}
