# Changelog

[emojilog](https://github.com/Abdur-rahmaanJ/emojilog) 1.0

```
0.8.8
🛩️ 378 multi-platform Docker image
🔧 fix: Resolve GitHub Actions bug for multi-architecture builds

0.8.7
🔧 Dependency updates.
🔧 Panics are now fatal errors, so that normal user errors are not reported to Sentry.
🔧 Fix a log message.

0.8.6
🔧 Small updates to the API before fully revamping it
✨ Circuit breaking all the way
🔧 Refactor getter functions

0.8.5: 🎉 TLS termination for incoming client connection

0.8.4
🔧 breaking: Update plugin in the list and backup plugins config when using plugin install
🔧 Print and return instead of log.Panic
🎉 Add healthz endpoints

0.8.3: 🔧 Refactor server and proxy and remove gnet/v2

0.8.2
🎉 Add configuration and test for multi-tenancy
🎉 Clean up downloaded and extracted files after the plugin is installed 

0.8.1
✨ Shutdown metrics server gracefully
🎉 TLS support in metrics server

0.8.0
🔧 Lint on run
✨ Remove hard and soft limits

0.7.10
🎉 Add receive timeout
🔧 Add benchmarks

0.7.9: 🔧 Add more tests
0.7.8: 🔧 Fix a regression in plugin install command

0.7.7
🛩️ Generate SBOMs for source code and Docker image and upload it to the release assets
🔧 Report coverage to coveralls
🔧 Upgrade Go and dependencies

0.7.6: 🛩️ Publish image to Docker Hub

0.7.5
🎉 Add plugin list subcommand 
🎉 Add support for installing plugins from locally downloaded archive files

0.7.4
🔧 Fix to the GitHub Actions workflows 
🔧 Fix plugin install subcommand

0.7.3: 🎉 Implement the new JSON binary protocol

0.7.2
📑 Add contribution guides
🔧 Add Windows support for gateway plugin install command

0.7.1: 🛩️ Add Dockerfile

0.7.0
🛩️ Windows executables for both AMD64 and ARM64 architectures
🔧 Update dependencies

0.6.9
🎉 Add Sentry to all commands by @mostafa in #274
🔧 Fix cancel request error by @mostafa in #275

0.6.8
🎉 Add plugin init command by @mostafa in #268
🎉 Add plugin lint command by @mostafa in #269
🎉 Add plugin install subcommand by @mostafa in #273

0.6.7
🔧 All the default values are moved from New<feature> functions to the config package.
🔧 The port of the metrics is changed from 2112 to 9090 to conform with Prometheus port.

0.6.6
🛩️ Generate checksum for APT and RPM packages
🎉 Default value for non default config groups

0.6.5
📑 Update README for the BIG RELEASE, which still needs some polishing.
🛩️ APT and RPM packages are built and released from now on for AMD64 and ARM64 platforms.
🔧 The config file location for gatewayd.yaml and gatewayd_plugins.yaml are now read from the current working directory, then the /etc, and the fallback will be the current working directory. 🔧 Loading the config is done at a later stage.

0.6.4
🔧 Fix termination issue

0.6.3
🔧 Add config subcommand

0.6.2
🔧 This release contains updates and the fix to a long-standing issue that caused clients to wait indefinitely for reading from the server.

0.6.1
🔧 Update all dependencies to their latest versions.
🔧 Fix linter timeout.
🔧 Fix check for the existence of the requires in the plugin config.
🔧 Check if plugin metadata is present and not nil.
🔧 Kill the plugin client if it can't be started or dispensed.
🔧 Fix default chunk size.
🔧 Regenerate gRPC API stubs.
🔧 Remove SHA256SUM function and its tests in favor of secure config.
🔧 Fix CI for testing the go plugin template.

0.6.0
🔧 Go is updated to 1.20 plus all the dependencies.
🔧 The gRPC and HTTP API ports are changed to 18080 and 19090 respectively.
✨ Timeouts are added to plugin hooks: previously, hooks could block the main thread indefinitely, thus making GatewayD unresponsive.

0.5.6
✨ Improve client performace

0.5.5
✨ Shutdown server(s) gracefully
🔧 Check if client.ID is not empty
🔧 Increase buffer size to 128 MB and chunk size to 8 KB
🔧 Update SDK and other dependencies

0.5.4
🔧 Use enum HookName for hook names

0.5.3
🎉 Fix bug in client config
🎉 Report usage statistics

0.5.2
🎉 Reload plugin on crash if enabled in gatewayd_plugins.yaml

0.5.1
🎉 Add development mode

0.5.0
🎉 Add Admin API

0.4.5
🎉 Add tracing with OpenTelemetry

0.4.4
🎉 Introduction of enableMetricsMerger parameter in the plugins config, gatewayd_plugins.yaml, to enable/disable metrics merger.
🔧 Disable plugin-related schedulers when there's no plugin loaded or no addresses is configured for merging metrics.

0.4.3
🔧 Refactor proxy passthrough
🔧 Handle plugin errors properly

0.4.2
🔧 Refactor run command
    Add multi-tenancy. Now, one can add multiple configurations to each config directive, and start listening on multiple ports with separate loggers, pools, proxies and clients.

0.4.1
🔧 Fix flaky tests
🔧 Fix metric merger from failing when the plugin dies
🎉 Add ping to plugins
🎉 Add health check to plugins, so that they can be removed from registry and metrics merger if they die suddenly
🔧 A few other fixes

0.4.0
🔧 Reset the author on all my commits, hence no changelog, since it changed all the commit hashes, and the tags are invalid up to this point.
🛩️ Replace the goreleaser config and release flow with good old bash and make files.
🎉 Extract plugin types, constants, protobuf files, buf configs and stubs to gatewayd-plugin-sdk (gatewayd-io/gatewayd-plugin-sdk#1).
🔧 Enable dependabot to help with upgrading dependencies.
🔧 A few other fixes and cleanups.

0.3.4
🛩️ Add build and release workflow

0.3.3
🎉 The logger is now created and initialized after reading configuration from the file, and before loading plugins. This removes the need to have two loggers, one DefaultLogger for plugin registry and the rest of the code, right before the other logger is actually initialized.
🎉 Sentry is added for capturing panics and reporting important errors

0.3.2
🎉 Remove embedded postgres by @mostafa in #130
🎉 Use multiple outputs in logger by @mostafa in #129
🔧 Add logging to (r)syslog by @mostafa in #131

0.3.1
🔧 Refactor merger slightly
🎉 Add version subcommand
🔧 Reorganize config

0.3.0
🎉 Add Prometheus metrics
🎉 Collect and merge Prometheus metrics from plugins

0.2.5
🔧 Tests
🎉 Lumberjack to rotate log files
🔧 Remove logger output to buffer in favor of go-capturer

0.2.4
🔧 Minor fixes
🔧 Fix time.Duration parsing
🔧 Clean-ups for CLI flag parsing
🔧 Hooks cleanup and merger
🔧 Make hooks more generic with generic gRPC methods
🎉 New hook: OnHook 

0.2.3
🎉 Recycle connection on timeout 
🎉 Implement onTrafficToClient

0.2.2
🎉 Load configs from env vars
🎉 New hooks for traffic control
🔧 Cleanup interfaces 
🔧 Slight refactoring of the hook system plus adding termination logic

0.2.1
🔧 Fix flaky TestRunServer test
🎉 New config parser
    breaking: default config object for loggers, clients, pool and proxy config

0.2.0
🎉 Add cmdline args to plugin cmd
🎉 Pass environment variables to plugins on load
🎉 Use koanf getter functions
🎉 Implement requires field for checking if required plugins are loaded or not

0.1.4
✨ Improve connection stability
🎉 Add TCP keepalive
🔧 Improve traffic hooks

0.1.3
✨ Retry connection after close by server
🔧 Use structured logging everywhere
📑 Add docstring to all functions

0.1.2
✨ Import increase to server stability
🔧 Improve client code by introducing configurable parameters for timeouts with defaults
🔧 Tests

0.1.1
🔧 Change parameter type of HookConfig.Run
🔧 Improve error handling

0.1.0: ✨ Fix concurrent connection handling

0.0.9
🎉 Add adapter that redirects logs from the internal logger of the plugin system, hclog, to the global logger of GatewayD, zerolog
🔧 Remove gnet logging

0.0.8
🎉 Add plugin system containing Hooks, Plugin Protocol and Registry
🔧 Tests
🔧 Refactorings related to plugins system

0.0.7
🎉 buf for linting, building and generate stubs from protocol buffer files
🎉 Add plugin system using Hashicorp's go-plugin
🎉 Chaining of hook results
🎉 Merge disk config with plugins configs

0.0.6
🎉 Add hooks to be used by plugins
🎉 Generic pool to contain client and server connections
📑 Added flow diagram
🔧 Use time.Duration instead of int for durations

0.0.5: 🎉 Add onfig parser that loads gatewayd.yaml

0.0.4
🎉 Add cli skeleton
🎉 Add run command to replace main.go 

0.0.3
🔧 Decouple proxy from server
🔧 Decouple pool and client creation from proxy
✨ Logrus replaced with zerolog
🔧 Tests

0.0.2
🔧 Fixes
🔧 CI with lint
🔧 Unit tests

0.0.1
🎉 Added conenction pool
🎉 Added proxy
```