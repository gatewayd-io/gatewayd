package cmd

import (
	"context"
	"log"
	"maps"
	"os"
	"syscall"

	"github.com/gatewayd-io/gatewayd/config"
	gerr "github.com/gatewayd-io/gatewayd/errors"
	"github.com/gatewayd-io/gatewayd/raft"
	"github.com/gatewayd-io/gatewayd/tracing"
	"github.com/getsentry/sentry-go"
	"github.com/spf13/cobra"
	"go.opentelemetry.io/otel"
)

var (
	testMode bool
	testApp  *GatewayDApp
)

// EnableTestMode enables test mode and returns the previous value.
// This should only be used in tests.
func EnableTestMode() bool {
	previous := testMode
	testMode = true
	return previous
}

// runCmd represents the run command.
var runCmd = &cobra.Command{
	Use:   "run",
	Short: "Run a GatewayD instance",
	Run: func(cmd *cobra.Command, _ []string) {
		app := NewGatewayDApp(cmd)

		// If test mode is enabled, we need to access the app instance from the test,
		// so we can stop the server gracefully.
		if testMode {
			testApp = app
		}

		runCtx, span := otel.Tracer(config.TracerName).Start(context.Background(), "GatewayD")
		span.End()

		// Handle signals from the user.
		app.handleSignals(
			runCtx,
			[]os.Signal{
				os.Interrupt,
				os.Kill,
				syscall.SIGTERM,
				syscall.SIGABRT,
				syscall.SIGQUIT,
				syscall.SIGHUP,
				syscall.SIGINT,
			},
		)

		// Stop the server gracefully when the program terminates cleanly.
		defer app.stopGracefully(runCtx, nil)

		// Enable tracing with OpenTelemetry.
		if app.EnableTracing {
			// TODO: Make this configurable.
			shutdown := tracing.OTLPTracer(true, app.CollectorURL, config.TracerName)
			defer func() {
				if err := shutdown(context.Background()); err != nil {
					cmd.Println(err)
					app.stopGracefully(runCtx, nil)
					os.Exit(gerr.FailedToStartTracer)
				}
			}()
		}

		span.End()

		// Enable Sentry.
		if app.EnableSentry {
			_, span := otel.Tracer(config.TracerName).Start(runCtx, "Sentry")
			defer span.End()

			// Initialize Sentry.
			err := sentry.Init(sentry.ClientOptions{
				Dsn:              DSN,
				TracesSampleRate: config.DefaultTraceSampleRate,
				AttachStacktrace: config.DefaultAttachStacktrace,
			})
			if err != nil {
				span.RecordError(err)
				cmd.Println("Sentry initialization failed: ", err)
				return
			}

			// Flush buffered events before the program terminates.
			defer sentry.Flush(config.DefaultFlushTimeout)
			// Recover from panics and report the error to Sentry.
			defer sentry.Recover()
		}

		// Lint the configuration files before loading them.
		if app.EnableLinting {
			_, span := otel.Tracer(config.TracerName).Start(runCtx, "Lint configuration files")
			defer span.End()

			// Lint the global configuration file and fail if it's not valid.
			if err := lintConfig(Global, app.GlobalConfigFile); err != nil {
				log.Fatal(err)
			}

			// Lint the plugin configuration file and fail if it's not valid.
			if err := lintConfig(Plugins, app.PluginConfigFile); err != nil {
				log.Fatal(err)
			}
		}

		// Load the configuration files.
		if err := app.loadConfig(runCtx); err != nil {
			log.Fatal(err)
		}

		// Create and initialize loggers from the config.
		// And then set the default logger.
		logger := app.createLoggers(runCtx, cmd)

		if app.DevMode {
			logger.Warn().Msg(
				"Running GatewayD in development mode (not recommended for production)")
		}

		// Create the Act registry.
		if err := app.createActRegistry(logger); err != nil {
			logger.Error().Err(err).Msg("Failed to create act registry")
			app.stopGracefully(runCtx, nil)
			os.Exit(gerr.FailedToCreateActRegistry)
		}

		// Load policies from the configuration file and add them to the registry.
		if err := app.loadPolicies(logger); err != nil {
			logger.Error().Err(err).Msg("Failed to load policies")
			app.stopGracefully(runCtx, nil)
			os.Exit(gerr.FailedToLoadPolicies)
		}

		logger.Info().Fields(map[string]any{
			"policies": maps.Keys(app.actRegistry.Policies),
		}).Msg("Policies are loaded")

		// Create the plugin registry.
		app.createPluginRegistry(runCtx, logger)

		// Load plugins and register their hooks.
		app.pluginRegistry.LoadPlugins(
			runCtx,
			app.conf.Plugin.Plugins,
			app.conf.Plugin.StartTimeout,
		)

		// Start the metrics merger if enabled.
		app.startMetricsMerger(runCtx, logger)

		// TODO: Move this to the plugin registry.
		ctx, span := otel.Tracer(config.TracerName).Start(runCtx, "Plugin health check")

		// Start the health check scheduler only if there are plugins.
		app.startHealthCheckScheduler(runCtx, ctx, span, logger)

		span.End()

		// Merge the global config with the one from the plugins.
		if err := app.onConfigLoaded(runCtx, span, logger); err != nil {
			app.stopGracefully(runCtx, nil)
			os.Exit(gerr.FailedToMergeGlobalConfig)
		}

		// Start the metrics server if enabled.
		go func(app *GatewayDApp) {
			if err := app.startMetricsServer(runCtx, logger); err != nil {
				logger.Error().Err(err).Msg("Failed to start metrics server")
				span.RecordError(err)
			}
		}(app)

		// Run the OnNewLogger hook.
		app.onNewLogger(span, logger)

		_, span = otel.Tracer(config.TracerName).Start(runCtx, "Create pools and clients")

		// Create pools and clients.
		if err := app.createPoolAndClients(runCtx, span); err != nil {
			logger.Error().Err(err).Msg("Failed to create pools and clients")
			span.RecordError(err)
			app.stopGracefully(runCtx, nil)
			os.Exit(gerr.FailedToCreatePoolAndClients)
		}

		span.End()

		_, span = otel.Tracer(config.TracerName).Start(runCtx, "Create proxies")

		// Create proxies.
		app.createProxies(runCtx, span)

		span.End()

		_, span = otel.Tracer(config.TracerName).Start(runCtx, "Create Raft Node")
		defer span.End()

		// Create the Raft node.
		raftNode, originalErr := raft.NewRaftNode(logger, app.conf.Global.Raft)
		if originalErr != nil {
			logger.Error().Err(originalErr).Msg("Failed to start raft node")
			span.RecordError(originalErr)
			app.stopGracefully(runCtx, nil)
			os.Exit(gerr.FailedToStartRaftNode)
		}

		_, span = otel.Tracer(config.TracerName).Start(runCtx, "Create servers")

		// Create servers.
		app.createServers(runCtx, span, raftNode)

		span.End()

		// Start the API servers.
		app.startAPIServers(runCtx, logger, raftNode)

		// Report usage.
		app.reportUsage(logger)

		_, span = otel.Tracer(config.TracerName).Start(runCtx, "Start servers")

		// Start the servers.
		app.startServers(runCtx, span)

		span.End()

		// Wait for the server to shut down.
		<-app.stopChan
	},
}

func init() {
	rootCmd.AddCommand(runCmd)

	runCmd.Flags().StringP(
		"config", "c", config.GetDefaultConfigFilePath(config.GlobalConfigFilename),
		"Global config file")
	runCmd.Flags().StringP(
		"plugin-config", "p", config.GetDefaultConfigFilePath(config.PluginsConfigFilename),
		"Plugin config file")
	runCmd.Flags().Bool("dev", false, "Enable development mode for plugin development")
	runCmd.Flags().Bool("tracing", false, "Enable tracing with OpenTelemetry via gRPC")
	runCmd.Flags().String(
		"collector-url", "localhost:4317", "Collector URL of OpenTelemetry gRPC endpoint")
	runCmd.Flags().Bool("sentry", true, "Enable Sentry")
	runCmd.Flags().Bool("usage-report", true, "Enable usage report")
	runCmd.Flags().Bool("lint", true, "Enable linting of configuration files")
	runCmd.Flags().Bool("metrics-merger", true, "Enable metrics merger")
}
