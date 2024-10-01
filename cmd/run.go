package cmd

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"io"
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
	sdkAct "github.com/gatewayd-io/gatewayd-plugin-sdk/act"
	sdkPlugin "github.com/gatewayd-io/gatewayd-plugin-sdk/plugin"
	v1 "github.com/gatewayd-io/gatewayd-plugin-sdk/plugin/v1"
	"github.com/gatewayd-io/gatewayd/act"
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
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/redis/go-redis/v9"
	"github.com/rs/zerolog"
	"github.com/spf13/cobra"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
	"golang.org/x/exp/maps"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

var _ io.Writer = &cobraCmdWriter{}

type cobraCmdWriter struct {
	*cobra.Command
}

func (c *cobraCmdWriter) Write(p []byte) (int, error) {
	c.Print(string(p))
	return len(p), nil
}

// TODO: Get rid of the global variables.
// https://github.com/gatewayd-io/gatewayd/issues/324
var (
	enableTracing     bool
	enableLinting     bool
	collectorURL      string
	enableSentry      bool
	devMode           bool
	enableUsageReport bool
	pluginConfigFile  string
	globalConfigFile  string
	conf              *config.Config
	pluginRegistry    *plugin.Registry
	actRegistry       *act.Registry
	metricsServer     *http.Server

	UsageReportURL = "localhost:59091"

	loggers              = make(map[string]zerolog.Logger)
	pools                = make(map[string]map[string]*pool.Pool)
	clients              = make(map[string]map[string]*config.Client)
	proxies              = make(map[string]map[string]*network.Proxy)
	servers              = make(map[string]*network.Server)
	healthCheckScheduler = gocron.NewScheduler(time.UTC)

	stopChan = make(chan struct{})
)

func StopGracefully(
	runCtx context.Context,
	sig os.Signal,
	metricsMerger *metrics.Merger,
	metricsServer *http.Server,
	pluginRegistry *plugin.Registry,
	logger zerolog.Logger,
	servers map[string]*network.Server,
	stopChan chan struct{},
	httpServer *api.HTTPServer,
	grpcServer *api.GRPCServer,
) {
	_, span := otel.Tracer(config.TracerName).Start(runCtx, "Shutdown server")
	currentSignal := "unknown"
	if sig != nil {
		currentSignal = sig.String()
	}

	logger.Info().Msg("Notifying the plugins that the server is shutting down")
	if pluginRegistry != nil {
		pluginTimeoutCtx, cancel := context.WithTimeout(context.Background(), conf.Plugin.Timeout)
		defer cancel()

		//nolint:contextcheck
		_, err := pluginRegistry.Run(
			pluginTimeoutCtx,
			map[string]interface{}{"signal": currentSignal},
			v1.HookName_HOOK_NAME_ON_SIGNAL,
		)
		if err != nil {
			logger.Error().Err(err).Msg("Failed to run OnSignal hooks")
			span.RecordError(err)
		}
	}

	logger.Info().Msg("GatewayD is shutting down")
	span.AddEvent("GatewayD is shutting down", trace.WithAttributes(
		attribute.String("signal", currentSignal),
	))
	if healthCheckScheduler != nil {
		healthCheckScheduler.Stop()
		healthCheckScheduler.Clear()
		logger.Info().Msg("Stopped health check scheduler")
		span.AddEvent("Stopped health check scheduler")
	}
	if metricsMerger != nil {
		metricsMerger.Stop()
		logger.Info().Msg("Stopped metrics merger")
		span.AddEvent("Stopped metrics merger")
	}
	if metricsServer != nil {
		//nolint:contextcheck
		if err := metricsServer.Shutdown(context.Background()); err != nil {
			logger.Error().Err(err).Msg("Failed to stop metrics server")
			span.RecordError(err)
		} else {
			logger.Info().Msg("Stopped metrics server")
			span.AddEvent("Stopped metrics server")
		}
	}
	for name, server := range servers {
		logger.Info().Str("name", name).Msg("Stopping server")
		server.Shutdown()
		span.AddEvent("Stopped server")
	}
	logger.Info().Msg("Stopped all servers")
	if pluginRegistry != nil {
		pluginRegistry.Shutdown()
		logger.Info().Msg("Stopped plugin registry")
		span.AddEvent("Stopped plugin registry")
	}
	span.End()

	if httpServer != nil {
		httpServer.Shutdown(runCtx)
		logger.Info().Msg("Stopped HTTP Server")
		span.AddEvent("Stopped HTTP Server")
	}

	if grpcServer != nil {
		grpcServer.Shutdown(runCtx)
		logger.Info().Msg("Stopped gRPC Server")
		span.AddEvent("Stopped gRPC Server")
	}

	// Close the stop channel to notify the other goroutines to stop.
	stopChan <- struct{}{}
	close(stopChan)
}

// runCmd represents the run command.
var runCmd = &cobra.Command{
	Use:   "run",
	Short: "Run a GatewayD instance",
	Run: func(cmd *cobra.Command, _ []string) {
		// Enable tracing with OpenTelemetry.
		if enableTracing {
			// TODO: Make this configurable.
			shutdown := tracing.OTLPTracer(true, collectorURL, config.TracerName)
			defer func() {
				if err := shutdown(context.Background()); err != nil {
					cmd.Println(err)
					os.Exit(gerr.FailedToStartTracer)
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
		if enableLinting {
			_, span := otel.Tracer(config.TracerName).Start(runCtx, "Lint configuration files")
			defer span.End()

			// Lint the global configuration file and fail if it's not valid.
			if err := lintConfig(Global, globalConfigFile); err != nil {
				log.Fatal(err)
			}

			// Lint the plugin configuration file and fail if it's not valid.
			if err := lintConfig(Plugins, pluginConfigFile); err != nil {
				log.Fatal(err)
			}
		}

		// Load global and plugin configuration.
		conf = config.NewConfig(runCtx, config.Config{GlobalConfigFile: globalConfigFile, PluginConfigFile: pluginConfigFile})
		if err := conf.InitConfig(runCtx); err != nil {
			log.Fatal(err)
		}

		// Create and initialize loggers from the config.
		// Use cobra command cmd instead of os.Stdout for the console output.
		cmdLogger := &cobraCmdWriter{cmd}
		for name, cfg := range conf.Global.Loggers {
			loggers[name] = logging.NewLogger(runCtx, logging.LoggerConfig{
				Output:     cfg.GetOutput(),
				ConsoleOut: cmdLogger,
				Level: config.If(
					config.Exists(config.LogLevels, cfg.Level),
					config.LogLevels[cfg.Level],
					config.LogLevels[config.DefaultLogLevel],
				),
				TimeFormat: config.If(
					config.Exists(config.TimeFormats, cfg.TimeFormat),
					config.TimeFormats[cfg.TimeFormat],
					config.TimeFormats[config.DefaultTimeFormat],
				),
				ConsoleTimeFormat: config.If(
					config.Exists(
						config.ConsoleTimeFormats, cfg.ConsoleTimeFormat),
					config.ConsoleTimeFormats[cfg.ConsoleTimeFormat],
					config.ConsoleTimeFormats[config.DefaultConsoleTimeFormat],
				),
				NoColor:        cfg.NoColor,
				FileName:       cfg.FileName,
				MaxSize:        cfg.MaxSize,
				MaxBackups:     cfg.MaxBackups,
				MaxAge:         cfg.MaxAge,
				Compress:       cfg.Compress,
				LocalTime:      cfg.LocalTime,
				SyslogPriority: cfg.GetSyslogPriority(),
				RSyslogNetwork: cfg.RSyslogNetwork,
				RSyslogAddress: cfg.RSyslogAddress,
				Name:           name,
			})
		}

		// Set the default logger.
		logger := loggers[config.Default]

		if devMode {
			logger.Warn().Msg(
				"Running GatewayD in development mode (not recommended for production)")
		}

		// Create a new act registry given the built-in signals, policies, and actions.
		var publisher *act.Publisher
		if conf.Plugin.ActionRedis.Enabled {
			rdb := redis.NewClient(&redis.Options{
				Addr: conf.Plugin.ActionRedis.Address,
			})
			var err error
			publisher, err = act.NewPublisher(act.Publisher{
				Logger:      logger,
				RedisDB:     rdb,
				ChannelName: conf.Plugin.ActionRedis.Channel,
			})
			if err != nil {
				logger.Error().Err(err).Msg("Failed to create publisher for act registry")
				os.Exit(gerr.FailedToCreateActRegistry)
			}
			logger.Info().Msg("Created Redis publisher for Act registry")
		}

		actRegistry = act.NewActRegistry(
			act.Registry{
				Signals:              act.BuiltinSignals(),
				Policies:             act.BuiltinPolicies(),
				Actions:              act.BuiltinActions(),
				DefaultPolicyName:    conf.Plugin.DefaultPolicy,
				PolicyTimeout:        conf.Plugin.PolicyTimeout,
				DefaultActionTimeout: conf.Plugin.ActionTimeout,
				TaskPublisher:        publisher,
				Logger:               logger,
			})

		if actRegistry == nil {
			logger.Error().Msg("Failed to create act registry")
			os.Exit(gerr.FailedToCreateActRegistry)
		}

		// Load policies from the configuration file and add them to the registry.
		for _, plc := range conf.Plugin.Policies {
			if policy, err := sdkAct.NewPolicy(
				plc.Name, plc.Policy, plc.Metadata,
			); err != nil || policy == nil {
				logger.Error().Err(err).Str("name", plc.Name).Msg("Failed to create policy")
			} else {
				actRegistry.Add(policy)
			}
		}

		logger.Info().Fields(map[string]interface{}{
			"policies": maps.Keys(actRegistry.Policies),
		}).Msg("Policies are loaded")

		// Create a new plugin registry.
		// The plugins are loaded and hooks registered before the configuration is loaded.
		pluginRegistry = plugin.NewRegistry(
			runCtx,
			plugin.Registry{
				ActRegistry: actRegistry,
				Compatibility: config.If(
					config.Exists(
						config.CompatibilityPolicies, conf.Plugin.CompatibilityPolicy,
					),
					config.CompatibilityPolicies[conf.Plugin.CompatibilityPolicy],
					config.DefaultCompatibilityPolicy),
				Logger:  logger,
				DevMode: devMode,
			},
		)

		// Load plugins and register their hooks.
		pluginRegistry.LoadPlugins(runCtx, conf.Plugin.Plugins, conf.Plugin.StartTimeout)

		// Start the metrics merger if enabled.
		var metricsMerger *metrics.Merger
		if conf.Plugin.EnableMetricsMerger {
			metricsMerger = metrics.NewMerger(runCtx, metrics.Merger{
				MetricsMergerPeriod: conf.Plugin.MetricsMergerPeriod,
				Logger:              logger,
			})
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

		// Ping the plugins to check if they are alive, and remove them if they are not.
		startDelay := time.Now().Add(conf.Plugin.HealthCheckPeriod)
		if _, err := healthCheckScheduler.Every(
			conf.Plugin.HealthCheckPeriod).SingletonMode().StartAt(startDelay).Do(func() {
			_, span := otel.Tracer(config.TracerName).Start(ctx, "Run plugin health check")
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
						pluginRegistry.LoadPlugins(runCtx, pluginConfig, conf.Plugin.StartTimeout)
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
			logger.Info().Str(
				"healthCheckPeriod", conf.Plugin.HealthCheckPeriod.String(),
			).Msg("Starting plugin health check scheduler")
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
			if err := conf.MergeGlobalConfig(runCtx, updatedGlobalConfig); err != nil {
				log.Fatal(err)
			}
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

			scheme := "http://"
			if metricsConfig.KeyFile != "" && metricsConfig.CertFile != "" {
				scheme = "https://"
			}

			fqdn, err := url.Parse(scheme + metricsConfig.Address)
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
					// The WriteHeader method intentionally does nothing, to prevent a bug
					// in the merging metrics that causes the headers to be written twice,
					// which results in an error: "http: superfluous response.WriteHeader call".
					next.ServeHTTP(
						&metrics.HeaderBypassResponseWriter{
							ResponseWriter: responseWriter,
						},
						request)
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

			mux := http.NewServeMux()
			mux.HandleFunc("/", func(responseWriter http.ResponseWriter, _ *http.Request) {
				// Serve a static page with a link to the metrics endpoint.
				if _, err := responseWriter.Write([]byte(fmt.Sprintf(
					`<html><head><title>GatewayD Prometheus Metrics Server</title></head><body><a href="%s">Metrics</a></body></html>`,
					address,
				))); err != nil {
					logger.Error().Err(err).Msg("Failed to write metrics")
					span.RecordError(err)
					sentry.CaptureException(err)
				}
			})

			if conf.Plugin.EnableMetricsMerger && metricsMerger != nil {
				handler = mergedMetricsHandler(handler)
			}

			readHeaderTimeout := config.If(
				metricsConfig.ReadHeaderTimeout > 0,
				metricsConfig.ReadHeaderTimeout,
				config.DefaultReadHeaderTimeout,
			)

			// Check if the metrics server is already running before registering the handler.
			if _, err = http.Get(address); err != nil { //nolint:gosec
				// The timeout handler limits the nested handlers from running for too long.
				mux.Handle(
					metricsConfig.Path,
					http.TimeoutHandler(
						gziphandler.GzipHandler(handler),
						readHeaderTimeout,
						"The request timed out while fetching the metrics",
					),
				)
			} else {
				logger.Warn().Msg("Metrics server is already running, consider changing the port")
				span.RecordError(err)
			}

			// Create a new metrics server.
			timeout := config.If(
				metricsConfig.Timeout > 0,
				metricsConfig.Timeout,
				config.DefaultMetricsServerTimeout,
			)
			metricsServer = &http.Server{
				Addr:              metricsConfig.Address,
				Handler:           mux,
				ReadHeaderTimeout: readHeaderTimeout,
				ReadTimeout:       timeout,
				WriteTimeout:      timeout,
				IdleTimeout:       timeout,
			}

			logger.Info().Fields(map[string]interface{}{
				"address":           address,
				"timeout":           timeout.String(),
				"readHeaderTimeout": readHeaderTimeout.String(),
			}).Msg("Metrics are exposed")

			if metricsConfig.CertFile != "" && metricsConfig.KeyFile != "" {
				// Set up TLS.
				metricsServer.TLSConfig = &tls.Config{
					MinVersion: tls.VersionTLS13,
					CurvePreferences: []tls.CurveID{
						tls.CurveP521,
						tls.CurveP384,
						tls.CurveP256,
					},
					CipherSuites: []uint16{
						tls.TLS_AES_128_GCM_SHA256,
						tls.TLS_AES_256_GCM_SHA384,
						tls.TLS_CHACHA20_POLY1305_SHA256,
					},
				}
				metricsServer.TLSNextProto = make(
					map[string]func(*http.Server, *tls.Conn, http.Handler))
				logger.Debug().Msg("Metrics server is running with TLS")

				// Start the metrics server with TLS.
				if err = metricsServer.ListenAndServeTLS(
					metricsConfig.CertFile, metricsConfig.KeyFile); !errors.Is(err, http.ErrServerClosed) {
					logger.Error().Err(err).Msg("Failed to start metrics server")
					span.RecordError(err)
				}
			} else {
				// Start the metrics server without TLS.
				if err = metricsServer.ListenAndServe(); !errors.Is(err, http.ErrServerClosed) {
					logger.Error().Err(err).Msg("Failed to start metrics server")
					span.RecordError(err)
				}
			}
		}(conf.Global.Metrics[config.Default], logger)

		// This is a notification hook, so we don't care about the result.
		pluginTimeoutCtx, cancel = context.WithTimeout(context.Background(), conf.Plugin.Timeout)
		defer cancel()

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

		// Declare httpServer and grpcServer here as it is used in the StopGracefully function ahead of their definition.
		var httpServer *api.HTTPServer
		var grpcServer *api.GRPCServer

		_, span = otel.Tracer(config.TracerName).Start(runCtx, "Create pools and clients")
		// Create and initialize pools of connections.
		for configGroupName, configGroup := range conf.Global.Pools {
			for configBlockName, cfg := range configGroup {
				logger := loggers[configGroupName]
				// Check if the pool size is greater than zero.
				currentPoolSize := config.If(
					cfg.Size > 0,
					// Check if the pool size is greater than the minimum pool size.
					config.If(
						cfg.Size > config.MinimumPoolSize,
						cfg.Size,
						config.MinimumPoolSize,
					),
					config.DefaultPoolSize,
				)

				if _, ok := pools[configGroupName]; !ok {
					pools[configGroupName] = make(map[string]*pool.Pool)
				}
				pools[configGroupName][configBlockName] = pool.NewPool(runCtx, currentPoolSize)

				span.AddEvent("Create pool", trace.WithAttributes(
					attribute.String("name", configBlockName),
					attribute.Int("size", currentPoolSize),
				))

				if _, ok := clients[configGroupName]; !ok {
					clients[configGroupName] = make(map[string]*config.Client)
				}

				// Get client config from the config file.
				if clientConfig, ok := conf.Global.Clients[configGroupName][configBlockName]; !ok {
					// This ensures that the default client config is used if the pool name is not
					// found in the clients section.
					clients[configGroupName][configBlockName] = conf.Global.Clients[config.Default][config.DefaultConfigurationBlock]
				} else {
					// Merge the default client config with the one from the pool.
					clients[configGroupName][configBlockName] = clientConfig
				}

				// Fill the missing and zero values with the default ones.
				clients[configGroupName][configBlockName].TCPKeepAlivePeriod = config.If(
					clients[configGroupName][configBlockName].TCPKeepAlivePeriod > 0,
					clients[configGroupName][configBlockName].TCPKeepAlivePeriod,
					config.DefaultTCPKeepAlivePeriod,
				)
				clients[configGroupName][configBlockName].ReceiveDeadline = config.If(
					clients[configGroupName][configBlockName].ReceiveDeadline > 0,
					clients[configGroupName][configBlockName].ReceiveDeadline,
					config.DefaultReceiveDeadline,
				)
				clients[configGroupName][configBlockName].ReceiveTimeout = config.If(
					clients[configGroupName][configBlockName].ReceiveTimeout > 0,
					clients[configGroupName][configBlockName].ReceiveTimeout,
					config.DefaultReceiveTimeout,
				)
				clients[configGroupName][configBlockName].SendDeadline = config.If(
					clients[configGroupName][configBlockName].SendDeadline > 0,
					clients[configGroupName][configBlockName].SendDeadline,
					config.DefaultSendDeadline,
				)
				clients[configGroupName][configBlockName].ReceiveChunkSize = config.If(
					clients[configGroupName][configBlockName].ReceiveChunkSize > 0,
					clients[configGroupName][configBlockName].ReceiveChunkSize,
					config.DefaultChunkSize,
				)
				clients[configGroupName][configBlockName].DialTimeout = config.If(
					clients[configGroupName][configBlockName].DialTimeout > 0,
					clients[configGroupName][configBlockName].DialTimeout,
					config.DefaultDialTimeout,
				)

				// Add clients to the pool.
				for range currentPoolSize {
					clientConfig := clients[configGroupName][configBlockName]
					clientConfig.GroupName = configGroupName
					clientConfig.BlockName = configBlockName
					client := network.NewClient(
						runCtx, clientConfig, logger,
						network.NewRetry(
							network.Retry{
								Retries: clientConfig.Retries,
								Backoff: config.If(
									clientConfig.Backoff > 0,
									clientConfig.Backoff,
									config.DefaultBackoff,
								),
								BackoffMultiplier:  clientConfig.BackoffMultiplier,
								DisableBackoffCaps: clientConfig.DisableBackoffCaps,
								Logger:             loggers[configBlockName],
							},
						),
					)

					if client != nil {
						eventOptions := trace.WithAttributes(
							attribute.String("name", configBlockName),
							attribute.String("group", configGroupName),
							attribute.String("network", client.Network),
							attribute.String("address", client.Address),
							attribute.Int("receiveChunkSize", client.ReceiveChunkSize),
							attribute.String("receiveDeadline", client.ReceiveDeadline.String()),
							attribute.String("receiveTimeout", client.ReceiveTimeout.String()),
							attribute.String("sendDeadline", client.SendDeadline.String()),
							attribute.String("dialTimeout", client.DialTimeout.String()),
							attribute.Bool("tcpKeepAlive", client.TCPKeepAlive),
							attribute.String("tcpKeepAlivePeriod", client.TCPKeepAlivePeriod.String()),
							attribute.String("localAddress", client.LocalAddr()),
							attribute.String("remoteAddress", client.RemoteAddr()),
							attribute.Int("retries", clientConfig.Retries),
							attribute.String("backoff", client.Retry().Backoff.String()),
							attribute.Float64("backoffMultiplier", clientConfig.BackoffMultiplier),
							attribute.Bool("disableBackoffCaps", clientConfig.DisableBackoffCaps),
						)
						if client.ID != "" {
							eventOptions = trace.WithAttributes(
								attribute.String("id", client.ID),
							)
						}

						span.AddEvent("Create client", eventOptions)

						pluginTimeoutCtx, cancel = context.WithTimeout(
							context.Background(), conf.Plugin.Timeout)
						defer cancel()

						clientCfg := map[string]interface{}{
							"id":                 client.ID,
							"name":               configBlockName,
							"group":              configGroupName,
							"network":            client.Network,
							"address":            client.Address,
							"receiveChunkSize":   client.ReceiveChunkSize,
							"receiveDeadline":    client.ReceiveDeadline.String(),
							"receiveTimeout":     client.ReceiveTimeout.String(),
							"sendDeadline":       client.SendDeadline.String(),
							"dialTimeout":        client.DialTimeout.String(),
							"tcpKeepAlive":       client.TCPKeepAlive,
							"tcpKeepAlivePeriod": client.TCPKeepAlivePeriod.String(),
							"localAddress":       client.LocalAddr(),
							"remoteAddress":      client.RemoteAddr(),
							"retries":            clientConfig.Retries,
							"backoff":            client.Retry().Backoff.String(),
							"backoffMultiplier":  clientConfig.BackoffMultiplier,
							"disableBackoffCaps": clientConfig.DisableBackoffCaps,
						}
						_, err := pluginRegistry.Run(
							pluginTimeoutCtx, clientCfg, v1.HookName_HOOK_NAME_ON_NEW_CLIENT)
						if err != nil {
							logger.Error().Err(err).Msg("Failed to run OnNewClient hooks")
							span.RecordError(err)
						}

						err = pools[configGroupName][configBlockName].Put(client.ID, client)
						if err != nil {
							logger.Error().Err(err).Msg("Failed to add client to the pool")
							span.RecordError(err)
						}
					} else {
						logger.Error().Msg("Failed to create client, please check the configuration")
						go func() {
							// Wait for the stop signal to exit gracefully.
							// This prevents the program from waiting indefinitely
							// after the StopGracefully function is called.
							<-stopChan
							os.Exit(gerr.FailedToCreateClient)
						}()
						StopGracefully(
							runCtx,
							nil,
							metricsMerger,
							metricsServer,
							pluginRegistry,
							logger,
							servers,
							stopChan,
							httpServer,
							grpcServer,
						)
					}
				}

				// Verify that the pool is properly populated.
				logger.Info().Fields(map[string]interface{}{
					"name":  configBlockName,
					"count": strconv.Itoa(pools[configGroupName][configBlockName].Size()),
				}).Msg("There are clients available in the pool")

				if pools[configGroupName][configBlockName].Size() != currentPoolSize {
					logger.Error().Msg(
						"The pool size is incorrect, either because " +
							"the clients cannot connect due to no network connectivity " +
							"or the server is not running. exiting...")
					pluginRegistry.Shutdown()
					os.Exit(gerr.FailedToInitializePool)
				}

				pluginTimeoutCtx, cancel = context.WithTimeout(
					context.Background(), conf.Plugin.Timeout)
				defer cancel()

				_, err = pluginRegistry.Run(
					pluginTimeoutCtx,
					map[string]interface{}{"name": configBlockName, "size": currentPoolSize},
					v1.HookName_HOOK_NAME_ON_NEW_POOL)
				if err != nil {
					logger.Error().Err(err).Msg("Failed to run OnNewPool hooks")
					span.RecordError(err)
				}
			}
		}

		span.End()

		_, span = otel.Tracer(config.TracerName).Start(runCtx, "Create proxies")
		// Create and initialize prefork proxies with each pool of clients.
		for configGroupName, configGroup := range conf.Global.Proxies {
			for configBlockName, cfg := range configGroup {
				logger := loggers[configGroupName]
				clientConfig := clients[configGroupName][configBlockName]

				// Fill the missing and zero value with the default one.
				cfg.HealthCheckPeriod = config.If(
					cfg.HealthCheckPeriod > 0,
					cfg.HealthCheckPeriod,
					config.DefaultHealthCheckPeriod,
				)

				if _, ok := proxies[configGroupName]; !ok {
					proxies[configGroupName] = make(map[string]*network.Proxy)
				}

				proxies[configGroupName][configBlockName] = network.NewProxy(
					runCtx,
					network.Proxy{
						GroupName:            configGroupName,
						BlockName:            configBlockName,
						AvailableConnections: pools[configGroupName][configBlockName],
						PluginRegistry:       pluginRegistry,
						HealthCheckPeriod:    cfg.HealthCheckPeriod,
						ClientConfig:         clientConfig,
						Logger:               logger,
						PluginTimeout:        conf.Plugin.Timeout,
					},
				)

				span.AddEvent("Create proxy", trace.WithAttributes(
					attribute.String("name", configBlockName),
					attribute.String("healthCheckPeriod", cfg.HealthCheckPeriod.String()),
				))

				pluginTimeoutCtx, cancel = context.WithTimeout(
					context.Background(), conf.Plugin.Timeout)
				defer cancel()

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
		}

		span.End()

		_, span = otel.Tracer(config.TracerName).Start(runCtx, "Create servers")
		// Create and initialize servers.
		for name, cfg := range conf.Global.Servers {
			logger := loggers[name]

			var serverProxies []network.IProxy
			for _, proxy := range proxies[name] {
				serverProxies = append(serverProxies, proxy)
			}

			servers[name] = network.NewServer(
				runCtx,
				network.Server{
					GroupName: name,
					Network:   cfg.Network,
					Address:   cfg.Address,
					TickInterval: config.If(
						cfg.TickInterval > 0,
						cfg.TickInterval,
						config.DefaultTickInterval,
					),
					Options: network.Option{
						// Can be used to send keepalive messages to the client.
						EnableTicker: cfg.EnableTicker,
					},
					Proxies:                    serverProxies,
					Logger:                     logger,
					PluginRegistry:             pluginRegistry,
					PluginTimeout:              conf.Plugin.Timeout,
					EnableTLS:                  cfg.EnableTLS,
					CertFile:                   cfg.CertFile,
					KeyFile:                    cfg.KeyFile,
					HandshakeTimeout:           cfg.HandshakeTimeout,
					LoadbalancerStrategyName:   cfg.LoadBalancer.Strategy,
					LoadbalancerRules:          cfg.LoadBalancer.LoadBalancingRules,
					LoadbalancerConsistentHash: cfg.LoadBalancer.ConsistentHash,
				},
			)

			span.AddEvent("Create server", trace.WithAttributes(
				attribute.String("name", name),
				attribute.String("network", cfg.Network),
				attribute.String("address", cfg.Address),
				attribute.String("tickInterval", cfg.TickInterval.String()),
				attribute.String("pluginTimeout", conf.Plugin.Timeout.String()),
				attribute.Bool("enableTLS", cfg.EnableTLS),
				attribute.String("certFile", cfg.CertFile),
				attribute.String("keyFile", cfg.KeyFile),
				attribute.String("handshakeTimeout", cfg.HandshakeTimeout.String()),
			))

			pluginTimeoutCtx, cancel = context.WithTimeout(
				context.Background(), conf.Plugin.Timeout)
			defer cancel()

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
				Servers:     servers,
			}

			apiObj := &api.API{
				Options:        &apiOptions,
				Config:         conf,
				PluginRegistry: pluginRegistry,
				Pools:          pools,
				Proxies:        proxies,
				Servers:        servers,
			}
			grpcServer = api.NewGRPCServer(
				runCtx,
				api.GRPCServer{
					API:           apiObj,
					HealthChecker: &api.HealthChecker{Servers: servers},
				},
			)
			if grpcServer != nil {
				go grpcServer.Start()
				logger.Info().Str("address", apiOptions.HTTPAddress).Msg("Started the HTTP API")

				httpServer = api.NewHTTPServer(&apiOptions)
				go httpServer.Start()

				logger.Info().Fields(
					map[string]interface{}{
						"network": apiOptions.GRPCNetwork,
						"address": apiOptions.GRPCAddress,
					},
				).Msg("Started the gRPC Server")
			}
		}

		// Report usage statistics.
		if enableUsageReport {
			go func() {
				conn, err := grpc.NewClient(
					UsageReportURL,
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
				defer func(conn *grpc.ClientConn) {
					err := conn.Close()
					if err != nil {
						logger.Trace().Err(err).Msg("Failed to close the connection to the usage report service")
					}
				}(conn)

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
					func(identifier sdkPlugin.Identifier, _ *plugin.Plugin) {
						report.Plugins = append(report.GetPlugins(), &usage.Plugin{
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
			metricsMerger *metrics.Merger,
			metricsServer *http.Server,
			stopChan chan struct{},
			httpServer *api.HTTPServer,
			grpcServer *api.GRPCServer,
		) {
			for sig := range signalsCh {
				for _, s := range signals {
					if sig != s {
						StopGracefully(
							runCtx,
							sig,
							metricsMerger,
							metricsServer,
							pluginRegistry,
							logger,
							servers,
							stopChan,
							httpServer,
							grpcServer,
						)
						os.Exit(0)
					}
				}
			}
		}(pluginRegistry, logger, servers, metricsMerger, metricsServer, stopChan, httpServer, grpcServer)

		_, span = otel.Tracer(config.TracerName).Start(runCtx, "Start servers")
		// Start the server.
		for name, server := range servers {
			logger := loggers[name]
			go func(
				span trace.Span,
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
			}(span, server, logger, healthCheckScheduler, metricsMerger, pluginRegistry)
		}
		span.End()

		// Wait for the server to shut down.
		<-stopChan
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
	runCmd.Flags().BoolVar(
		&enableLinting, "lint", true, "Enable linting of configuration files")
}
