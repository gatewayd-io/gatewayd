package cmd

import (
	"context"
	"crypto/tls"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"runtime"
	"strconv"
	"syscall"
	"time"

	"github.com/NYTimes/gziphandler"
	sdkPlugin "github.com/gatewayd-io/gatewayd-plugin-sdk/plugin"
	v1 "github.com/gatewayd-io/gatewayd-plugin-sdk/plugin/v1"
	"github.com/gatewayd-io/gatewayd/api"
	"github.com/gatewayd-io/gatewayd/config"
	gerr "github.com/gatewayd-io/gatewayd/errors"
	"github.com/gatewayd-io/gatewayd/logging"
	"github.com/gatewayd-io/gatewayd/metrics"
	"github.com/gatewayd-io/gatewayd/network"
	"github.com/gatewayd-io/gatewayd/plugin"
	"github.com/gatewayd-io/gatewayd/pool"
	"github.com/gatewayd-io/gatewayd/tracing"
	usage "github.com/gatewayd-io/gatewayd/usagereport/v1"
	"github.com/getsentry/sentry-go"
	"github.com/go-co-op/gocron"
	"github.com/panjf2000/gnet/v2"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/rs/zerolog"
	"github.com/spf13/cobra"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

var (
	enableTracing     bool
	collectorURL      string
	enableSentry      bool
	devMode           bool
	enableUsageReport bool
	pluginConfigFile  string
	globalConfigFile  string
	conf              *config.Config
	pluginRegistry    *plugin.Registry

	UsageReportURL = "localhost:59091"

	loggers              = make(map[string]zerolog.Logger)
	pools                = make(map[string]*pool.Pool)
	clients              = make(map[string]*config.Client)
	proxies              = make(map[string]*network.Proxy)
	servers              = make(map[string]*network.Server)
	healthCheckScheduler = gocron.NewScheduler(time.UTC)
)

// runCmd represents the run command.
var runCmd = &cobra.Command{
	Use:   "run",
	Short: "Run a GatewayD instance",
	Run: func(cmd *cobra.Command, args []string) {
		// Enable tracing with OpenTelemetry.
		if enableTracing {
			// TODO: Make this configurable.
			shutdown := tracing.OTLPTracer(true, collectorURL, config.TracerName)
			defer func() {
				if err := shutdown(context.Background()); err != nil {
					log.Fatal(err)
				}
			}()
		}

		runCtx, span := otel.Tracer(config.TracerName).Start(context.Background(), "GatewayD")
		span.End()

		// Enable Sentry.
		if enableSentry {
			_, span := otel.Tracer(config.TracerName).Start(runCtx, "Sentry")
			defer span.End()

			// Initialize Sentry.
			err := sentry.Init(sentry.ClientOptions{
				Dsn:              "https://e22f42dbb3e0433fbd9ea32453faa598@o4504550475038720.ingest.sentry.io/4504550481723392",
				TracesSampleRate: config.DefaultTraceSampleRate,
				AttachStacktrace: config.DefaultAttachStacktrace,
			})
			if err != nil {
				span.RecordError(err)
				log.Fatalf("sentry.Init: %s", err)
			}

			// Flush buffered events before the program terminates.
			defer sentry.Flush(config.DefaultFlushTimeout)
			// Recover from panics and report the error to Sentry.
			defer sentry.Recover()
		}

		// Load global and plugin configuration.
		conf = config.NewConfig(runCtx, globalConfigFile, pluginConfigFile)
		conf.InitConfig(runCtx)

		// Create and initialize loggers from the config.
		for name, cfg := range conf.Global.Loggers {
			loggers[name] = logging.NewLogger(runCtx, logging.LoggerConfig{
				Output:            cfg.GetOutput(),
				Level:             cfg.GetLevel(),
				TimeFormat:        cfg.GetTimeFormat(),
				ConsoleTimeFormat: cfg.GetConsoleTimeFormat(),
				NoColor:           cfg.NoColor,
				FileName:          cfg.FileName,
				MaxSize:           cfg.MaxSize,
				MaxBackups:        cfg.MaxBackups,
				MaxAge:            cfg.MaxAge,
				Compress:          cfg.Compress,
				LocalTime:         cfg.LocalTime,
				SyslogPriority:    cfg.GetSyslogPriority(),
				RSyslogNetwork:    cfg.RSyslogNetwork,
				RSyslogAddress:    cfg.RSyslogAddress,
			})
		}

		// Set the default logger.
		logger := loggers[config.Default]

		if devMode {
			logger.Warn().Msg(
				"Running GatewayD in development mode (not recommended for production)")
		}

		// Create a new plugin registry.
		// The plugins are loaded and hooks registered before the configuration is loaded.
		pluginRegistry = plugin.NewRegistry(
			runCtx,
			conf.Plugin.GetPluginCompatibilityPolicy(),
			conf.Plugin.GetVerificationPolicy(),
			conf.Plugin.GetAcceptancePolicy(),
			conf.Plugin.GetTerminationPolicy(),
			logger,
			devMode,
		)

		// Load plugins and register their hooks.
		pluginRegistry.LoadPlugins(runCtx, conf.Plugin.Plugins)

		// Start the metrics merger if enabled.
		var metricsMerger *metrics.Merger
		if conf.Plugin.EnableMetricsMerger {
			metricsMerger = metrics.NewMerger(runCtx, conf.Plugin.MetricsMergerPeriod, logger)
			pluginRegistry.ForEach(func(_ sdkPlugin.Identifier, plugin *plugin.Plugin) {
				if metricsEnabled, err := strconv.ParseBool(plugin.Config["metricsEnabled"]); err == nil && metricsEnabled {
					metricsMerger.Add(plugin.ID.Name, plugin.Config["metricsUnixDomainSocket"])
					logger.Debug().Str("plugin", plugin.ID.Name).Msg(
						"Added plugin to metrics merger")
				}
			})
			metricsMerger.Start()
		}

		// TODO: Move this to the plugin registry.
		ctx, span := otel.Tracer(config.TracerName).Start(runCtx, "Plugin health check")

		logger.Info().Str(
			"healthCheckPeriod", conf.Plugin.HealthCheckPeriod.String(),
		).Msg("Starting plugin health check scheduler")
		// Ping the plugins to check if they are alive, and remove them if they are not.
		startDelay := time.Now().Add(conf.Plugin.HealthCheckPeriod)
		if _, err := healthCheckScheduler.Every(
			conf.Plugin.HealthCheckPeriod).SingletonMode().StartAt(startDelay).Do(func() {
			_, span = otel.Tracer(config.TracerName).Start(ctx, "Run plugin health check")
			defer span.End()

			var plugins []string
			pluginRegistry.ForEach(func(pluginId sdkPlugin.Identifier, plugin *plugin.Plugin) {
				if err := plugin.Ping(); err != nil {
					span.RecordError(err)
					logger.Error().Err(err).Msg("Failed to ping plugin")
					if conf.Plugin.EnableMetricsMerger && metricsMerger != nil {
						metricsMerger.Remove(pluginId.Name)
					}
					pluginRegistry.Remove(pluginId)

					if !conf.Plugin.ReloadOnCrash {
						return // Do not reload the plugins.
					}

					// Reload the plugins and register their hooks upon crash.
					logger.Info().Str("name", pluginId.Name).Msg("Reloading crashed plugin")
					pluginConfig := conf.Plugin.GetPlugins(pluginId.Name)
					if pluginConfig != nil {
						pluginRegistry.LoadPlugins(runCtx, pluginConfig)
					}
				} else {
					logger.Trace().Str("name", pluginId.Name).Msg("Successfully pinged plugin")
					plugins = append(plugins, pluginId.Name)
				}
			})
			span.SetAttributes(attribute.StringSlice("plugins", plugins))
		}); err != nil {
			logger.Error().Err(err).Msg("Failed to start plugin health check scheduler")
			span.RecordError(err)
		}
		if pluginRegistry.Size() > 0 {
			healthCheckScheduler.StartAsync()
		}

		span.End()

		// Set the plugin timeout context.
		pluginTimeoutCtx, cancel := context.WithTimeout(context.Background(), conf.Plugin.Timeout)
		defer cancel()

		// The config will be passed to the plugins that register to the "OnConfigLoaded" plugin.
		// The plugins can modify the config and return it.
		updatedGlobalConfig, err := pluginRegistry.Run(
			pluginTimeoutCtx, conf.GlobalKoanf.All(), v1.HookName_HOOK_NAME_ON_CONFIG_LOADED)
		if err != nil {
			logger.Error().Err(err).Msg("Failed to run OnConfigLoaded hooks")
			span.RecordError(err)
		}

		// If the config was modified by the plugins, merge it with the one loaded from the file.
		// Only global configuration is merged, which means that plugins cannot modify the plugin
		// configurations.
		if updatedGlobalConfig != nil {
			// Merge the config with the one loaded from the file (in memory).
			// The changes won't be persisted to disk.
			conf.MergeGlobalConfig(runCtx, updatedGlobalConfig)
		}

		// Start the metrics server if enabled.
		// TODO: Start multiple metrics servers. For now, only one default is supported.
		// I should first find a use case for those multiple metrics servers.
		go func(metricsConfig *config.Metrics, logger zerolog.Logger) {
			_, span := otel.Tracer(config.TracerName).Start(runCtx, "Start metrics server")
			defer span.End()

			// TODO: refactor this to a separate function.
			if !metricsConfig.Enabled {
				logger.Info().Msg("Metrics server is disabled")
				return
			}

			fqdn, err := url.Parse("http://" + metricsConfig.Address)
			if err != nil {
				logger.Error().Err(err).Msg("Failed to parse metrics address")
				span.RecordError(err)
				return
			}

			address, err := url.JoinPath(fqdn.String(), metricsConfig.Path)
			if err != nil {
				logger.Error().Err(err).Msg("Failed to parse metrics path")
				span.RecordError(err)
				return
			}

			// Merge the metrics from the plugins with the ones from GatewayD.
			mergedMetricsHandler := func(next http.Handler) http.Handler {
				handler := func(responseWriter http.ResponseWriter, request *http.Request) {
					if _, err := responseWriter.Write(metricsMerger.OutputMetrics); err != nil {
						logger.Error().Err(err).Msg("Failed to write metrics")
						span.RecordError(err)
						sentry.CaptureException(err)
					}
					next.ServeHTTP(responseWriter, request)
				}
				return http.HandlerFunc(handler)
			}

			handler := func() http.Handler {
				return promhttp.InstrumentMetricHandler(
					prometheus.DefaultRegisterer,
					promhttp.HandlerFor(prometheus.DefaultGatherer, promhttp.HandlerOpts{
						DisableCompression: true,
					}),
				)
			}()

			logger.Info().Str("address", address).Msg("Metrics are exposed")

			if conf.Plugin.EnableMetricsMerger && metricsMerger != nil {
				handler = mergedMetricsHandler(handler)
			}
			http.Handle(metricsConfig.Path, gziphandler.GzipHandler(handler))

			//nolint:gosec
			if err = http.ListenAndServe(
				metricsConfig.Address, nil); err != nil {
				logger.Error().Err(err).Msg("Failed to start metrics server")
				span.RecordError(err)
			}
		}(conf.Global.Metrics[config.Default], logger)

		// This is a notification hook, so we don't care about the result.
		// TODO: Use a context with a timeout
		if data, ok := conf.GlobalKoanf.Get("loggers").(map[string]interface{}); ok {
			_, err = pluginRegistry.Run(
				pluginTimeoutCtx, data, v1.HookName_HOOK_NAME_ON_NEW_LOGGER)
			if err != nil {
				logger.Error().Err(err).Msg("Failed to run OnNewLogger hooks")
				span.RecordError(err)
			}
		} else {
			logger.Error().Msg("Failed to get loggers from config")
		}

		_, span = otel.Tracer(config.TracerName).Start(runCtx, "Create pools and clients")
		// Create and initialize pools of connections.
		for name, cfg := range conf.Global.Pools {
			logger := loggers[name]
			pools[name] = pool.NewPool(runCtx, cfg.GetSize())

			span.AddEvent("Create pool", trace.WithAttributes(
				attribute.String("name", name),
				attribute.Int("size", cfg.GetSize()),
			))

			// Get client config from the config file.
			if clientConfig, ok := conf.Global.Clients[name]; !ok {
				// This ensures that the default client config is used if the pool name is not
				// found in the clients section.
				clients[name] = conf.Global.Clients[config.Default]
			} else {
				// Merge the default client config with the one from the pool.
				clients[name] = clientConfig
			}

			// Add clients to the pool.
			for i := 0; i < cfg.GetSize(); i++ {
				clientConfig := clients[name]
				client := network.NewClient(runCtx, clientConfig, logger)

				if client != nil {
					eventOptions := trace.WithAttributes(
						attribute.String("name", name),
						attribute.String("network", client.Network),
						attribute.String("address", client.Address),
						attribute.Int("receiveChunkSize", client.ReceiveChunkSize),
						attribute.String("receiveDeadline", client.ReceiveDeadline.String()),
						attribute.String("sendDeadline", client.SendDeadline.String()),
						attribute.Bool("tcpKeepAlive", client.TCPKeepAlive),
						attribute.String("tcpKeepAlivePeriod", client.TCPKeepAlivePeriod.String()),
						attribute.String("localAddress", client.Conn.LocalAddr().String()),
						attribute.String("remoteAddress", client.Conn.RemoteAddr().String()),
					)
					if client.ID != "" {
						eventOptions = trace.WithAttributes(
							attribute.String("id", client.ID),
						)
					}

					span.AddEvent("Create client", eventOptions)

					clientCfg := map[string]interface{}{
						"id":                 client.ID,
						"network":            client.Network,
						"address":            client.Address,
						"receiveChunkSize":   client.ReceiveChunkSize,
						"receiveDeadline":    client.ReceiveDeadline.String(),
						"sendDeadline":       client.SendDeadline.String(),
						"tcpKeepAlive":       client.TCPKeepAlive,
						"tcpKeepAlivePeriod": client.TCPKeepAlivePeriod.String(),
					}
					_, err := pluginRegistry.Run(
						pluginTimeoutCtx, clientCfg, v1.HookName_HOOK_NAME_ON_NEW_CLIENT)
					if err != nil {
						logger.Error().Err(err).Msg("Failed to run OnNewClient hooks")
						span.RecordError(err)
					}

					err = pools[name].Put(client.ID, client)
					if err != nil {
						logger.Error().Err(err).Msg("Failed to add client to the pool")
						span.RecordError(err)
					}
				} else {
					logger.Error().Msg("Failed to create client, please check the configuration")
					os.Exit(gerr.FailedToCreateClient)
				}
			}

			// Verify that the pool is properly populated.
			logger.Info().Fields(map[string]interface{}{
				"name":  name,
				"count": fmt.Sprint(pools[name].Size()),
			}).Msg("There are clients available in the pool")

			if pools[name].Size() != cfg.GetSize() {
				logger.Error().Msg(
					"The pool size is incorrect, either because " +
						"the clients cannot connect due to no network connectivity " +
						"or the server is not running. exiting...")
				pluginRegistry.Shutdown()
				os.Exit(gerr.FailedToInitializePool)
			}

			_, err = pluginRegistry.Run(
				pluginTimeoutCtx,
				map[string]interface{}{"name": name, "size": cfg.GetSize()},
				v1.HookName_HOOK_NAME_ON_NEW_POOL)
			if err != nil {
				logger.Error().Err(err).Msg("Failed to run OnNewPool hooks")
				span.RecordError(err)
			}
		}

		span.End()

		_, span = otel.Tracer(config.TracerName).Start(runCtx, "Create proxies")
		// Create and initialize prefork proxies with each pool of clients.
		for name, cfg := range conf.Global.Proxies {
			logger := loggers[name]
			clientConfig := clients[name]
			proxies[name] = network.NewProxy(
				runCtx,
				pools[name],
				pluginRegistry,
				cfg.Elastic,
				cfg.ReuseElasticClients,
				cfg.HealthCheckPeriod,
				clientConfig,
				logger,
				conf.Plugin.Timeout,
			)

			span.AddEvent("Create proxy", trace.WithAttributes(
				attribute.String("name", name),
				attribute.Bool("elastic", cfg.Elastic),
				attribute.Bool("reuseElasticClients", cfg.ReuseElasticClients),
				attribute.String("healthCheckPeriod", cfg.HealthCheckPeriod.String()),
			))

			if data, ok := conf.GlobalKoanf.Get("proxies").(map[string]interface{}); ok {
				_, err = pluginRegistry.Run(
					pluginTimeoutCtx, data, v1.HookName_HOOK_NAME_ON_NEW_PROXY)
				if err != nil {
					logger.Error().Err(err).Msg("Failed to run OnNewProxy hooks")
					span.RecordError(err)
				}
			} else {
				logger.Error().Msg("Failed to get proxy from config")
			}
		}

		span.End()

		_, span = otel.Tracer(config.TracerName).Start(runCtx, "Create servers")
		// Create and initialize servers.
		for name, cfg := range conf.Global.Servers {
			logger := loggers[name]
			servers[name] = network.NewServer(
				runCtx,
				cfg.Network,
				cfg.Address,
				cfg.SoftLimit,
				cfg.HardLimit,
				cfg.TickInterval,
				[]gnet.Option{
					// Scheduling options
					gnet.WithMulticore(cfg.MultiCore),
					gnet.WithLockOSThread(cfg.LockOSThread),
					// NumEventLoop overrides Multicore option.
					// gnet.WithNumEventLoop(1),

					// Can be used to send keepalive messages to the client.
					gnet.WithTicker(cfg.EnableTicker),

					// Internal event-loop load balancing options
					gnet.WithLoadBalancing(cfg.GetLoadBalancer()),

					// Buffer options
					gnet.WithReadBufferCap(cfg.ReadBufferCap),
					gnet.WithWriteBufferCap(cfg.WriteBufferCap),
					gnet.WithSocketRecvBuffer(cfg.SocketRecvBuffer),
					gnet.WithSocketSendBuffer(cfg.SocketSendBuffer),

					// TCP options
					gnet.WithReuseAddr(cfg.ReuseAddress),
					gnet.WithReusePort(cfg.ReusePort),
					gnet.WithTCPKeepAlive(cfg.TCPKeepAlive),
					gnet.WithTCPNoDelay(cfg.GetTCPNoDelay()),
				},
				proxies[name],
				logger,
				pluginRegistry,
				conf.Plugin.Timeout,
			)

			span.AddEvent("Create server", trace.WithAttributes(
				attribute.String("name", name),
				attribute.String("network", cfg.Network),
				attribute.String("address", cfg.Address),
				attribute.Int64("softLimit", int64(cfg.SoftLimit)),
				attribute.Int64("hardLimit", int64(cfg.HardLimit)),
				attribute.String("tickInterval", cfg.TickInterval.String()),
				attribute.Bool("multiCore", cfg.MultiCore),
				attribute.Bool("lockOSThread", cfg.LockOSThread),
				attribute.Bool("enableTicker", cfg.EnableTicker),
				attribute.String("loadBalancer", cfg.LoadBalancer),
				attribute.Int("readBufferCap", cfg.ReadBufferCap),
				attribute.Int("writeBufferCap", cfg.WriteBufferCap),
				attribute.Int("socketRecvBuffer", cfg.SocketRecvBuffer),
				attribute.Int("socketSendBuffer", cfg.SocketSendBuffer),
				attribute.Bool("reuseAddress", cfg.ReuseAddress),
				attribute.Bool("reusePort", cfg.ReusePort),
				attribute.String("tcpKeepAlive", cfg.TCPKeepAlive.String()),
				attribute.Bool("tcpNoDelay", cfg.TCPNoDelay),
				attribute.String("pluginTimeout", conf.Plugin.Timeout.String()),
			))

			if data, ok := conf.GlobalKoanf.Get("servers").(map[string]interface{}); ok {
				_, err = pluginRegistry.Run(
					pluginTimeoutCtx, data, v1.HookName_HOOK_NAME_ON_NEW_SERVER)
				if err != nil {
					logger.Error().Err(err).Msg("Failed to run OnNewServer hooks")
					span.RecordError(err)
				}
			} else {
				logger.Error().Msg("Failed to get the servers configuration")
			}
		}

		span.End()

		// Start the HTTP and gRPC APIs.
		if conf.Global.API.Enabled {
			apiOptions := api.Options{
				Logger:      logger,
				GRPCNetwork: conf.Global.API.GRPCNetwork,
				GRPCAddress: conf.Global.API.GRPCAddress,
				HTTPAddress: conf.Global.API.HTTPAddress,
			}

			go api.StartGRPCAPI(&api.API{
				Options:        &apiOptions,
				Config:         conf,
				PluginRegistry: pluginRegistry,
				Pools:          pools,
				Proxies:        proxies,
				Servers:        servers,
			})
			logger.Info().Str("address", apiOptions.HTTPAddress).Msg("Started the HTTP API")

			go api.StartHTTPAPI(&apiOptions)
			logger.Info().Fields(
				map[string]interface{}{
					"network": apiOptions.GRPCNetwork,
					"address": apiOptions.GRPCAddress,
				},
			).Msg("Started the gRPC API")
		}

		// Report usage statistics.
		if enableUsageReport {
			go func() {
				conn, err := grpc.Dial(UsageReportURL,
					grpc.WithTransportCredentials(
						credentials.NewTLS(
							&tls.Config{
								MinVersion: tls.VersionTLS12,
							},
						),
					),
				)
				if err != nil {
					logger.Trace().Err(err).Msg(
						"Failed to dial to the gRPC server for usage reporting")
				}
				defer conn.Close()

				client := usage.NewUsageReportServiceClient(conn)
				report := usage.UsageReportRequest{
					Version:        config.Version,
					RuntimeVersion: runtime.Version(),
					Goos:           runtime.GOOS,
					Goarch:         runtime.GOARCH,
					Service:        "gatewayd",
					DevMode:        devMode,
					Plugins:        []*usage.Plugin{},
				}
				pluginRegistry.ForEach(
					func(identifier sdkPlugin.Identifier, plugin *plugin.Plugin) {
						report.Plugins = append(report.Plugins, &usage.Plugin{
							Name:     identifier.Name,
							Version:  identifier.Version,
							Checksum: identifier.Checksum,
						})
					},
				)
				_, err = client.Report(context.Background(), &report)
				if err != nil {
					logger.Trace().Err(err).Msg("Failed to report usage statistics")
				}
			}()
		}

		// Shutdown the server gracefully.
		var signals []os.Signal
		signals = append(signals,
			os.Interrupt,
			os.Kill,
			syscall.SIGTERM,
			syscall.SIGABRT,
			syscall.SIGQUIT,
			syscall.SIGHUP,
			syscall.SIGINT,
		)
		signalsCh := make(chan os.Signal, 1)
		signal.Notify(signalsCh, signals...)
		go func(pluginRegistry *plugin.Registry,
			logger zerolog.Logger,
			servers map[string]*network.Server,
		) {
			for sig := range signalsCh {
				for _, s := range signals {
					if sig != s {
						_, span := otel.Tracer(config.TracerName).Start(runCtx, "Shutdown server")

						logger.Info().Msg("Notifying the plugins that the server is shutting down")
						_, err := pluginRegistry.Run(
							pluginTimeoutCtx,
							map[string]interface{}{"signal": sig.String()},
							v1.HookName_HOOK_NAME_ON_SIGNAL,
						)
						if err != nil {
							logger.Error().Err(err).Msg("Failed to run OnSignal hooks")
							span.RecordError(err)
						}

						logger.Info().Msg("Stopping GatewayD")
						span.AddEvent("Stopping GatewayD", trace.WithAttributes(
							attribute.String("signal", sig.String()),
						))
						healthCheckScheduler.Clear()
						logger.Info().Msg("Stopped health check scheduler")
						span.AddEvent("Stopped health check scheduler")
						if metricsMerger != nil {
							metricsMerger.Stop()
							logger.Info().Msg("Stopped metrics merger")
							span.AddEvent("Stopped metrics merger")
						}
						for name, server := range servers {
							logger.Info().Str("name", name).Msg("Stopping server")
							server.Shutdown()
							span.AddEvent("Stopped server")
						}
						logger.Info().Msg("Stopped all servers")
						pluginRegistry.Shutdown()
						logger.Info().Msg("Stopped plugin registry")
						span.AddEvent("Stopped plugin registry")

						span.End()
						os.Exit(0)
					}
				}
			}
		}(pluginRegistry, logger, servers)

		_, span = otel.Tracer(config.TracerName).Start(runCtx, "Start servers")
		// Start the server.
		for name, server := range servers {
			logger := loggers[name]
			go func(
				server *network.Server,
				logger zerolog.Logger,
				healthCheckScheduler *gocron.Scheduler,
				metricsMerger *metrics.Merger,
				pluginRegistry *plugin.Registry,
			) {
				span.AddEvent("Start server")
				if err := server.Run(); err != nil {
					logger.Error().Err(err).Msg("Failed to start server")
					span.RecordError(err)

					healthCheckScheduler.Clear()
					if metricsMerger != nil {
						metricsMerger.Stop()
					}
					server.Shutdown()
					pluginRegistry.Shutdown()
					os.Exit(gerr.FailedToStartServer)
				}
			}(server, logger, healthCheckScheduler, metricsMerger, pluginRegistry)
		}
		span.End()

		// Wait for the server to shutdown.
		<-make(chan struct{})
	},
}

func init() {
	rootCmd.AddCommand(runCmd)

	runCmd.Flags().StringVarP(
		&globalConfigFile,
		"config", "c", config.GetDefaultConfigFilePath(config.GlobalConfigFilename),
		"Global config file")
	runCmd.Flags().StringVarP(
		&pluginConfigFile,
		"plugin-config", "p", config.GetDefaultConfigFilePath(config.PluginsConfigFilename),
		"Plugin config file")
	runCmd.Flags().BoolVar(
		&devMode, "dev", false, "Enable development mode for plugin development")
	runCmd.Flags().BoolVar(
		&enableTracing, "tracing", false, "Enable tracing with OpenTelemetry via gRPC")
	runCmd.Flags().StringVar(
		&collectorURL, "collector-url", "localhost:4317",
		"Collector URL of OpenTelemetry gRPC endpoint")
	runCmd.Flags().BoolVar(
		&enableSentry, "sentry", true, "Enable Sentry")
	runCmd.Flags().BoolVar(
		&enableUsageReport, "usage-report", true, "Enable usage report")
}
