package cmd

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
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
	"github.com/google/go-github/v53/github"
	jsonSchemaGenerator "github.com/invopop/jsonschema"
	"github.com/knadh/koanf"
	koanfJson "github.com/knadh/koanf/parsers/json"
	"github.com/knadh/koanf/parsers/yaml"
	jsonSchemaV5 "github.com/santhosh-tekuri/jsonschema/v5"
	"github.com/spf13/cast"
	"github.com/spf13/cobra"
	yamlv3 "gopkg.in/yaml.v3"
)

type (
	configFileType string
)

const (
	FilePermissions     os.FileMode = 0o644
	ExecFilePermissions os.FileMode = 0o755
	ExecFileMask        os.FileMode = 0o111
	MaxFileSize         int64       = 1024 * 1024 * 100 // 10MB
)

var (
	Global  configFileType = "global"
	Plugins configFileType = "plugins"

	DSN = "https://e22f42dbb3e0433fbd9ea32453faa598@o4504550475038720.ingest.sentry.io/4504550481723392"
)

// generateConfig generates a config file of the given type.
func generateConfig(
	cmd *cobra.Command, fileType configFileType, configFile string, forceRewriteFile bool,
) {
	logger := log.New(cmd.OutOrStdout(), "", 0)

	// Create a new config object and load the defaults.
	conf := &config.Config{
		GlobalKoanf: koanf.New("."),
		PluginKoanf: koanf.New("."),
	}
	conf.LoadDefaults(context.TODO())

	// Marshal the config file to YAML.
	var konfig *koanf.Koanf
	switch fileType {
	case Global:
		konfig = conf.GlobalKoanf
	case Plugins:
		konfig = conf.PluginKoanf
	default:
		logger.Fatal("Invalid config file type")
	}
	cfg, err := konfig.Marshal(yaml.Parser())
	if err != nil {
		logger.Fatal(err)
	}

	// Check if the config file already exists and if we should overwrite it.
	exists := false
	if _, err := os.Stat(configFile); err == nil && !forceRewriteFile {
		logger.Fatal(
			"Config file already exists. Use --force to overwrite or choose a different filename.")
	} else if err == nil {
		exists = true
	}

	// Create or overwrite the config file.
	if err := os.WriteFile(configFile, cfg, FilePermissions); err != nil {
		logger.Fatal(err)
	}

	verb := "created"
	if exists && forceRewriteFile {
		verb = "overwritten"
	}
	cmd.Printf("Config file '%s' was %s successfully.", configFile, verb)
}

func lintConfig(fileType configFileType, configFile string) error {
	// Load the config file and check it for errors.
	var conf *config.Config
	switch fileType {
	case Global:
		conf = config.NewConfig(context.TODO(), configFile, "")
		conf.LoadDefaults(context.TODO())
		conf.LoadGlobalConfigFile(context.TODO())
		conf.UnmarshalGlobalConfig(context.TODO())
	case Plugins:
		conf = config.NewConfig(context.TODO(), "", configFile)
		conf.LoadDefaults(context.TODO())
		conf.LoadPluginConfigFile(context.TODO())
		conf.UnmarshalPluginConfig(context.TODO())
	default:
		return gerr.ErrLintingFailed
	}

	// Marshal the config to JSON.
	var jsonData []byte
	var err error
	switch fileType {
	case Global:
		jsonData, err = conf.GlobalKoanf.Marshal(koanfJson.Parser())
	case Plugins:
		jsonData, err = conf.PluginKoanf.Marshal(koanfJson.Parser())
	default:
		return gerr.ErrLintingFailed
	}
	if err != nil {
		return gerr.ErrLintingFailed.Wrap(err)
	}

	// Unmarshal the JSON data into a map.
	var jsonBytes map[string]interface{}
	err = json.Unmarshal(jsonData, &jsonBytes)
	if err != nil {
		return gerr.ErrLintingFailed.Wrap(err)
	}

	// Generate a JSON schema from the config struct.
	var generatedSchema *jsonSchemaGenerator.Schema
	switch fileType {
	case Global:
		generatedSchema = jsonSchemaGenerator.Reflect(&config.GlobalConfig{})
	case Plugins:
		generatedSchema = jsonSchemaGenerator.Reflect(&config.PluginConfig{})
	default:
		return gerr.ErrLintingFailed
	}

	// Marshal the schema to JSON.
	schemaBytes, err := json.Marshal(generatedSchema)
	if err != nil {
		return gerr.ErrLintingFailed.Wrap(err)
	}

	// Compile the schema for validation.
	schema, err := jsonSchemaV5.CompileString("", string(schemaBytes))
	if err != nil {
		return gerr.ErrLintingFailed.Wrap(err)
	}

	// Validate the config against the schema.
	err = schema.Validate(jsonBytes)
	if err != nil {
		return gerr.ErrLintingFailed.Wrap(err)
	}

	return nil
}

func listPlugins(cmd *cobra.Command, pluginConfigFile string, onlyEnabled bool) {
	// Load the plugin config file.
	conf := config.NewConfig(context.TODO(), "", pluginConfigFile)
	conf.LoadDefaults(context.TODO())
	conf.LoadPluginConfigFile(context.TODO())
	conf.UnmarshalPluginConfig(context.TODO())

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

func extractZip(filename, dest string) ([]string, error) {
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
	filenames := []string{}
	for _, file := range zipRc.File {
		switch fileInfo := file.FileInfo(); {
		case fileInfo.IsDir():
			// Sanitize the path.
			filename := filepath.Clean(file.Name)
			if !path.IsAbs(filename) {
				destPath := path.Join(dest, filename)
				// Create the directory.

				if err := os.MkdirAll(destPath, FolderPermissions); err != nil {
					return nil, gerr.ErrExtractFailed.Wrap(err)
				}
			}
		case fileInfo.Mode().IsRegular():
			// Sanitize the path.
			outFilename := filepath.Join(filepath.Clean(dest), filepath.Clean(file.Name))

			// Check for ZipSlip.
			if strings.HasPrefix(outFilename, string(os.PathSeparator)) {
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
			fileRc, err := file.Open()
			if err != nil {
				os.Remove(outFilename)
				return nil, gerr.ErrExtractFailed.Wrap(err)
			}

			// Copy the file contents.
			if _, err := io.Copy(outFile, io.LimitReader(fileRc, MaxFileSize)); err != nil {
				os.Remove(outFilename)
				return nil, gerr.ErrExtractFailed.Wrap(err)
			}

			fileMode := file.FileInfo().Mode()
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
				fmt.Errorf("unknown file type: %s", file.Name))
		}
	}

	return filenames, nil
}

func extractTarGz(filename, dest string) ([]string, error) {
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

	// Create the output directory if it doesn't exist.
	if err := os.MkdirAll(dest, FolderPermissions); err != nil {
		return nil, gerr.ErrExtractFailed.Wrap(err)
	}

	tarReader := tar.NewReader(uncompressedStream)
	filenames := []string{}

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
			if strings.HasPrefix(outFilename, string(os.PathSeparator)) {
				return nil, gerr.ErrExtractFailed.Wrap(err)
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

func downloadFile(
	client *github.Client, account, pluginName string, releaseID int64, filename string,
) (string, error) {
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
	cwd, err := os.Getwd()
	if err != nil {
		return "", gerr.ErrDownloadFailed.Wrap(err)
	}
	filePath := path.Join([]string{cwd, filename}...)
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

// detectInstallLocation detects the install location based on the number of arguments.
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
func installPlugin(cmd *cobra.Command, pluginURL string) {
	var (
		// This is a list of files that will be deleted after the plugin is installed.
		toBeDeleted = []string{}

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
			// Get an specific release.
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

		// Find and download the plugin binary from the release assets.
		pluginFilename, downloadURL, releaseID = findAsset(release, func(name string) bool {
			return strings.Contains(name, runtime.GOOS) &&
				strings.Contains(name, runtime.GOARCH) &&
				strings.Contains(name, string(archiveExt))
		})
		var filePath string
		if downloadURL != "" && releaseID != 0 {
			cmd.Println("Downloading", downloadURL)
			filePath, err = downloadFile(client, account, pluginName, releaseID, pluginFilename)
			toBeDeleted = append(toBeDeleted, filePath)
			if err != nil {
				cmd.Println("Download failed: ", err)
				if cleanup {
					deleteFiles(toBeDeleted)
				}
				return
			}
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
			filePath, err = downloadFile(client, account, pluginName, releaseID, checksumsFilename)
			toBeDeleted = append(toBeDeleted, filePath)
			if err != nil {
				cmd.Println("Download failed: ", err)
				if cleanup {
					deleteFiles(toBeDeleted)
				}
				return
			}
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
				checksum := strings.Split(line, " ")[0]
				if checksum != sum {
					cmd.Println("Checksum verification failed")
					return
				}

				cmd.Println("Checksum verification passed")
				break
			}
		}

		if pullOnly {
			cmd.Println("Plugin binary downloaded to", pluginFilename)
			// Only the checksums file will be deleted if the --pull-only flag is set.
			if err := os.Remove(checksumsFilename); err != nil {
				cmd.Println("There was an error deleting the file: ", err)
			}
			return
		}
	case SourceUnknown:
	default:
		cmd.Println("Invalid URL or file path")
	}

	// NOTE: The rest of the code is executed regardless of the source,
	// since the plugin binary is already available (or downloaded) at this point.

	// Create a new "gatewayd_plugins.yaml" file if it doesn't exist.
	if _, err := os.Stat(pluginConfigFile); os.IsNotExist(err) {
		generateConfig(cmd, Plugins, pluginConfigFile, false)
	} else if !backupConfig && !noPrompt {
		// If the config file exists, we should prompt the user to backup
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
	var localPluginsConfig map[string]interface{}
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
		backupFilename := fmt.Sprintf("%s.bak", pluginConfigFile)
		if err := os.WriteFile(backupFilename, pluginsConfig, FilePermissions); err != nil {
			cmd.Println("There was an error backing up the plugins configuration file: ", err)
		}
		cmd.Println("Backup completed successfully")
	}

	// Extract the archive.
	var filenames []string
	switch archiveExt {
	case ExtensionZip:
		filenames, err = extractZip(pluginFilename, pluginOutputDir)
	case ExtensionTarGz:
		filenames, err = extractTarGz(pluginFilename, pluginOutputDir)
	default:
		cmd.Println("Invalid archive extension")
		return
	}

	if err != nil {
		cmd.Println("There was an error extracting the plugin archive: ", err)
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
			filepath.Join(pluginOutputDir, DefaultPluginConfigFilename))
		if err != nil {
			cmd.Println(
				"There was an error getting the default plugins configuration file: ", err)
			return
		}
		contents = string(contentsBytes)
	}

	// Get the plugin configuration from the downloaded plugins configuration file.
	var downloadedPluginConfig map[string]interface{}
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
