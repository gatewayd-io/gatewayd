package cmd

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"context"
	"encoding/json"
	"errors"
	"io"
	"log"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"

	"github.com/codingsince1985/checksum"
	"github.com/gatewayd-io/gatewayd/config"
	"github.com/google/go-github/v53/github"
	jsonSchemaGenerator "github.com/invopop/jsonschema"
	"github.com/knadh/koanf"
	koanfJson "github.com/knadh/koanf/parsers/json"
	"github.com/knadh/koanf/parsers/yaml"
	jsonSchemaV5 "github.com/santhosh-tekuri/jsonschema/v5"
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
func generateConfig(cmd *cobra.Command, fileType configFileType, configFile string, force bool) {
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
	if _, err := os.Stat(configFile); err == nil && !force {
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
	if exists && force {
		verb = "overwritten"
	}
	logger.Printf("Config file '%s' was %s successfully.", configFile, verb)
}

func lintConfig(cmd *cobra.Command, fileType configFileType, configFile string) {
	logger := log.New(cmd.OutOrStdout(), "", 0)

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
		logger.Fatal("Invalid config file type")
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
		logger.Fatal("Invalid config file type")
	}
	if err != nil {
		logger.Fatalf("Error marshalling %s config to JSON: %s\n", string(fileType), err)
	}

	// Unmarshal the JSON data into a map.
	var jsonBytes map[string]interface{}
	err = json.Unmarshal(jsonData, &jsonBytes)
	if err != nil {
		logger.Fatal("Error unmarshalling schema to JSON:\n", err)
	}

	// Generate a JSON schema from the config struct.
	var generatedSchema *jsonSchemaGenerator.Schema
	switch fileType {
	case Global:
		generatedSchema = jsonSchemaGenerator.Reflect(&config.GlobalConfig{})
	case Plugins:
		generatedSchema = jsonSchemaGenerator.Reflect(&config.PluginConfig{})
	default:
		logger.Fatal("Invalid config file type")
	}

	// Marshal the schema to JSON.
	schemaBytes, err := json.Marshal(generatedSchema)
	if err != nil {
		logger.Fatal("Error marshalling schema to JSON:\n", err)
	}

	// Compile the schema for validation.
	schema, err := jsonSchemaV5.CompileString("", string(schemaBytes))
	if err != nil {
		logger.Fatal("Error compiling schema:\n", err)
	}

	// Validate the config against the schema.
	err = schema.Validate(jsonBytes)
	if err != nil {
		logger.Fatalf("Error validating %s config: %s\n", string(fileType), err)
	}

	logger.Printf("%s config is valid\n", fileType)
}

func listPlugins(cmd *cobra.Command, pluginConfigFile string, onlyEnabled bool) {
	logger := log.New(cmd.OutOrStdout(), "", 0)

	// Load the plugin config file.
	conf := config.NewConfig(context.TODO(), "", pluginConfigFile)
	conf.LoadDefaults(context.TODO())
	conf.LoadPluginConfigFile(context.TODO())
	conf.UnmarshalPluginConfig(context.TODO())

	if len(conf.Plugin.Plugins) != 0 {
		logger.Printf("Total plugins: %d\n", len(conf.Plugin.Plugins))
		logger.Println("Plugins:")
	}

	// Print the list of plugins.
	for _, plugin := range conf.Plugin.Plugins {
		if onlyEnabled && !plugin.Enabled {
			continue
		}
		logger.Printf("  Name: %s\n", plugin.Name)
		logger.Printf("  Enabled: %t\n", plugin.Enabled)
		logger.Printf("  Path: %s\n", plugin.LocalPath)
		logger.Printf("  Args: %s\n", strings.Join(plugin.Args, " "))
		logger.Println("  Env:")
		for _, env := range plugin.Env {
			logger.Printf("    %s\n", env)
		}
		logger.Printf("  Checksum: %s\n", plugin.Checksum)
	}
}

func extractZip(filename, dest string) []string {
	// Open and extract the zip file.
	zipRc, err := zip.OpenReader(filename)
	if err != nil {
		if zipRc != nil {
			zipRc.Close()
		}
		log.Panic("There was an error opening the downloaded plugin file: ", err)
	}

	// Create the output directory if it doesn't exist.
	if err := os.MkdirAll(dest, FolderPermissions); err != nil {
		log.Panic("Failed to create directories: ", err)
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
					log.Panic("Failed to create directories: ", err)
				}
			}
		case fileInfo.Mode().IsRegular():
			// Sanitize the path.
			outFilename := filepath.Join(filepath.Clean(dest), filepath.Clean(file.Name))

			// Check for ZipSlip.
			if strings.HasPrefix(outFilename, string(os.PathSeparator)) {
				log.Panic("Invalid file path in zip archive, aborting")
			}

			// Create the file.
			outFile, err := os.Create(outFilename)
			if err != nil {
				log.Panic("Failed to create file: ", err)
			}

			// Open the file in the zip archive.
			fileRc, err := file.Open()
			if err != nil {
				log.Panic("Failed to open file in zip archive: ", err)
			}

			// Copy the file contents.
			if _, err := io.Copy(outFile, io.LimitReader(fileRc, MaxFileSize)); err != nil {
				outFile.Close()
				os.Remove(outFilename)
				log.Panic("Failed to write to the file: ", err)
			}
			outFile.Close()

			fileMode := file.FileInfo().Mode()
			// Set the file permissions.
			if fileMode.IsRegular() && fileMode&ExecFileMask != 0 {
				if err := os.Chmod(outFilename, ExecFilePermissions); err != nil {
					log.Panic("Failed to set executable file permissions: ", err)
				}
			} else {
				if err := os.Chmod(outFilename, FilePermissions); err != nil {
					log.Panic("Failed to set file permissions: ", err)
				}
			}

			filenames = append(filenames, outFile.Name())
		default:
			log.Panicf("Failed to extract zip archive: unknown type: %s", file.Name)
		}
	}

	if zipRc != nil {
		zipRc.Close()
	}

	return filenames
}

func extractTarGz(filename, dest string) []string {
	// Open and extract the tar.gz file.
	gzipStream, err := os.Open(filename)
	if err != nil {
		log.Panic("There was an error opening the downloaded plugin file: ", err)
	}

	uncompressedStream, err := gzip.NewReader(gzipStream)
	if err != nil {
		if gzipStream != nil {
			gzipStream.Close()
		}
		log.Panic("Failed to extract tarball: ", err)
	}

	// Create the output directory if it doesn't exist.
	if err := os.MkdirAll(dest, FolderPermissions); err != nil {
		log.Panic("Failed to create directories: ", err)
	}

	tarReader := tar.NewReader(uncompressedStream)
	filenames := []string{}

	for {
		header, err := tarReader.Next()

		if errors.Is(err, io.EOF) {
			break
		}

		if err != nil {
			log.Panic("Failed to extract tarball: ", err)
		}

		switch header.Typeflag {
		case tar.TypeDir:
			// Sanitize the path
			cleanPath := filepath.Clean(header.Name)
			// Ensure it is not an absolute path
			if !path.IsAbs(cleanPath) {
				destPath := path.Join(dest, cleanPath)
				if err := os.MkdirAll(destPath, FolderPermissions); err != nil {
					log.Panic("Failed to create directories: ", err)
				}
			}
		case tar.TypeReg:
			// Sanitize the path
			outFilename := path.Join(filepath.Clean(dest), filepath.Clean(header.Name))

			// Check for TarSlip.
			if strings.HasPrefix(outFilename, string(os.PathSeparator)) {
				log.Panic("Invalid file path in tarball, aborting")
			}

			// Create the file.
			outFile, err := os.Create(outFilename)
			if err != nil {
				log.Panic("Failed to create file: ", err)
			}
			if _, err := io.Copy(outFile, io.LimitReader(tarReader, MaxFileSize)); err != nil {
				outFile.Close()
				os.Remove(outFilename)
				log.Panic("Failed to write to the file: ", err)
			}
			outFile.Close()

			fileMode := header.FileInfo().Mode()
			// Set the file permissions
			if fileMode.IsRegular() && fileMode&ExecFileMask != 0 {
				if err := os.Chmod(outFilename, ExecFilePermissions); err != nil {
					log.Panic("Failed to set executable file permissions: ", err)
				}
			} else {
				if err := os.Chmod(outFilename, FilePermissions); err != nil {
					log.Panic("Failed to set file permissions: ", err)
				}
			}

			filenames = append(filenames, outFile.Name())
		default:
			log.Panicf(
				"Failed to extract tarball: unknown type: %s in %s",
				string(header.Typeflag),
				header.Name)
		}
	}

	if gzipStream != nil {
		gzipStream.Close()
	}

	return filenames
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
	client *github.Client, account, pluginName, downloadURL string,
	releaseID int64, filename string,
) {
	log.Println("Downloading", downloadURL)

	// Download the plugin.
	readCloser, redirectURL, err := client.Repositories.DownloadReleaseAsset(
		context.Background(), account, pluginName, releaseID, http.DefaultClient)
	if err != nil {
		log.Panic("There was an error downloading the plugin: ", err)
	}

	var reader io.ReadCloser
	if readCloser != nil {
		reader = readCloser
		defer readCloser.Close()
	} else if redirectURL != "" {
		// Download the plugin from the redirect URL.
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		req, err := http.NewRequestWithContext(ctx, http.MethodGet, redirectURL, nil)
		if err != nil {
			log.Panic("There was an error downloading the plugin: ", err)
		}

		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			log.Panic("There was an error downloading the plugin: ", err)
		}
		defer resp.Body.Close()

		reader = resp.Body
	}

	if reader != nil {
		defer reader.Close()
	} else {
		log.Panic("The plugin could not be downloaded, please try again later")
	}

	// Create the output file in the current directory and write the downloaded content.
	cwd, err := os.Getwd()
	if err != nil {
		log.Panic("There was an error downloading the plugin: ", err)
	}
	output, err := os.Create(path.Join([]string{cwd, filename}...))
	if err != nil {
		log.Panic("There was an error downloading the plugin: ", err)
	}
	defer output.Close()

	// Write the bytes to the file.
	_, err = io.Copy(output, reader)
	if err != nil {
		log.Panic("There was an error downloading the plugin: ", err)
	}

	log.Println("Download completed successfully")
}

func installFromGitHub(cmd *cobra.Command, args []string, pluginOutputDir string, pullOnly bool) {
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
		log.Println("Version not specified. Using latest version")
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
	account := accountRepo[0]
	pluginName := accountRepo[1]
	if account == "" || pluginName == "" {
		log.Panic(
			"Invalid URL. Use the following format: github.com/account/repository@version")
	}

	// Get the release artifact from GitHub.
	client := github.NewClient(nil)
	var release *github.RepositoryRelease
	var err error
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
	pluginFilename, downloadURL, releaseID := findAsset(release, func(name string) bool {
		return strings.Contains(name, runtime.GOOS) &&
			strings.Contains(name, runtime.GOARCH) &&
			strings.Contains(name, archiveExt)
	})
	if downloadURL != "" && releaseID != 0 {
		downloadFile(client, account, pluginName, downloadURL, releaseID, pluginFilename)
	} else {
		log.Panic("The plugin file could not be found in the release assets")
	}

	// Find and download the checksums.txt from the release assets.
	checksumsFilename, downloadURL, releaseID := findAsset(release, func(name string) bool {
		return strings.Contains(name, "checksums.txt")
	})
	if checksumsFilename != "" && downloadURL != "" && releaseID != 0 {
		downloadFile(client, account, pluginName, downloadURL, releaseID, checksumsFilename)
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

			log.Println("Checksum verification passed")
			break
		}
	}

	if pullOnly {
		log.Println("Plugin binary downloaded to", pluginFilename)
		return
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
			log.Println("Plugin binary extracted to", filename)
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

	// Remove the tar.gz file.
	err = os.Remove(pluginFilename)
	if err != nil {
		log.Panic("There was an error removing the downloaded plugin file: ", err)
	}

	// Remove the checksums.txt file.
	err = os.Remove(checksumsFilename)
	if err != nil {
		log.Panic("There was an error removing the checksums file: ", err)
	}

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

	// Get the list of files in the repository.
	var repoContents *github.RepositoryContent
	repoContents, _, _, err = client.Repositories.GetContents(
		context.Background(), account, pluginName, DefaultPluginConfigFilename, nil)
	if err != nil {
		log.Panic("There was an error getting the default plugins configuration file: ", err)
	}
	// Get the contents of the file.
	contents, err := repoContents.GetContent()
	if err != nil {
		log.Panic("There was an error getting the default plugins configuration file: ", err)
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

	// TODO: Clean up the plugin files if the installation fails.
	// TODO: Add a rollback mechanism.
	log.Println("Plugin installed successfully")
}
