package cmd

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"strconv"
	"syscall"

	"github.com/NYTimes/gziphandler"
	"github.com/gatewayd-io/gatewayd/config"
	gerr "github.com/gatewayd-io/gatewayd/errors"
	"github.com/gatewayd-io/gatewayd/logging"
	"github.com/gatewayd-io/gatewayd/metrics"
	"github.com/gatewayd-io/gatewayd/network"
	"github.com/gatewayd-io/gatewayd/plugin"
	"github.com/gatewayd-io/gatewayd/pool"
	"github.com/knadh/koanf"
	"github.com/knadh/koanf/parsers/yaml"
	"github.com/knadh/koanf/providers/confmap"
	"github.com/knadh/koanf/providers/file"
	"github.com/panjf2000/gnet/v2"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/rs/zerolog"
	"github.com/spf13/cobra"
)

var (
	DefaultLogger = logging.NewLogger(
		logging.LoggerConfig{
			Level:   zerolog.InfoLevel, // Default log level
			NoColor: true,
		},
	)
	// The plugins are loaded and hooks registered before the configuration is loaded.
	pluginRegistry = plugin.NewRegistry(config.Loose, config.PassDown, config.Accept, DefaultLogger)
	// Global koanf instance. Using "." as the key path delimiter.
	globalConfig = koanf.New(".")
	// Plugin koanf instance. Using "." as the key path delimiter.
	pluginConfig = koanf.New(".")
)

// runCmd represents the run command.
var runCmd = &cobra.Command{
	Use:   "run",
	Short: "Run a gatewayd instance",
	Run: func(cmd *cobra.Command, args []string) {
		// Load default plugin configuration.
		config.LoadPluginConfigDefaults(pluginConfig)

		// Load the plugin configuration file.
		if f, err := cmd.Flags().GetString("plugin-config"); err == nil {
			if err := pluginConfig.Load(file.Provider(f), yaml.Parser()); err != nil {
				DefaultLogger.Fatal().Err(err).Msg("Failed to load plugin configuration")
				os.Exit(gerr.FailedToLoadPluginConfig)
			}
		}

		// Load environment variables for the global configuration.
		config.LoadEnvVars(pluginConfig)

		// Unmarshal the plugin configuration for easier access.
		var pConfig config.PluginConfig
		if err := pluginConfig.Unmarshal("", &pConfig); err != nil {
			DefaultLogger.Fatal().Err(err).Msg("Failed to unmarshal plugin configuration")
			os.Exit(gerr.FailedToLoadPluginConfig)
		}

		// Set the plugin requirement's compatibility policy.
		pluginRegistry.Compatibility = pConfig.GetPluginCompatibilityPolicy()
		// Set hooks' signature verification policy.
		pluginRegistry.Verification = pConfig.GetVerificationPolicy()
		// Set custom hook acceptance policy.
		pluginRegistry.Acceptance = pConfig.GetAcceptancePolicy()

		// Load plugins and register their hooks.
		pluginRegistry.LoadPlugins(pConfig.Plugins)

		// Start the metrics merger.
		metricsMerger := metrics.NewMerger(pConfig.MetricsMergerPeriod, DefaultLogger)
		pluginRegistry.ForEach(func(_ plugin.Identifier, plugin *plugin.Plugin) {
			if metricsEnabled, err := strconv.ParseBool(plugin.Config["metricsEnabled"]); err == nil && metricsEnabled {
				metricsMerger.Add(plugin.ID.Name, plugin.Config["metricsUnixDomainSocket"])
			}
		})
		metricsMerger.Start()

		// Load default global configuration.
		config.LoadGlobalConfigDefaults(globalConfig)

		// Load the global configuration file.
		if f, err := cmd.Flags().GetString("config"); err == nil {
			if err := globalConfig.Load(file.Provider(f), yaml.Parser()); err != nil {
				DefaultLogger.Fatal().Err(err).Msg("Failed to load configuration")
				pluginRegistry.Shutdown()
				os.Exit(gerr.FailedToLoadGlobalConfig)
			}
		}

		// Load environment variables for the global configuration.
		config.LoadEnvVars(globalConfig)

		// Unmarshal the global configuration for easier access.
		var gConfig config.GlobalConfig
		if err := globalConfig.Unmarshal("", &gConfig); err != nil {
			DefaultLogger.Fatal().Err(err).Msg("Failed to unmarshal global configuration")
			pluginRegistry.Shutdown()
			os.Exit(gerr.FailedToLoadGlobalConfig)
		}

		// The config will be passed to the plugins that register to the "OnConfigLoaded" plugin.
		// The plugins can modify the config and return it.
		updatedGlobalConfig, err := pluginRegistry.Run(
			context.Background(),
			globalConfig.All(),
			plugin.OnConfigLoaded)
		if err != nil {
			DefaultLogger.Error().Err(err).Msg("Failed to run OnConfigLoaded hooks")
		}

		// If the config was modified by the plugins, merge it with the one loaded from the file.
		// Only global configuration is merged, which means that plugins cannot modify the plugin
		// configurations.
		if updatedGlobalConfig != nil {
			// Merge the config with the one loaded from the file (in memory).
			// The changes won't be persisted to disk.
			if err := globalConfig.Load(
				confmap.Provider(updatedGlobalConfig, "."), nil); err != nil {
				DefaultLogger.Fatal().Err(err).Msg("Failed to merge configuration")
			}
		}
		if err := globalConfig.Unmarshal("", &gConfig); err != nil {
			DefaultLogger.Fatal().Err(err).Msg("Failed to unmarshal updated global configuration")
			pluginRegistry.Shutdown()
			os.Exit(gerr.FailedToLoadGlobalConfig)
		}

		// Start the metrics server if enabled.
		go func(metricsConfig config.Metrics, logger zerolog.Logger) {
			// TODO: refactor this to a separate function.
			if !metricsConfig.Enabled {
				logger.Info().Msg("Metrics server is disabled")
				return
			}

			fqdn, err := url.Parse("http://" + metricsConfig.Address)
			if err != nil {
				logger.Error().Err(err).Msg("Failed to parse metrics address")
				return
			}

			address, err := url.JoinPath(fqdn.String(), metricsConfig.Path)
			if err != nil {
				logger.Error().Err(err).Msg("Failed to parse metrics path")
				return
			}

			// Merge the metrics from the plugins with the ones from GatewayD.
			mergedMetricsHandler := func(next http.Handler) http.Handler {
				handler := func(w http.ResponseWriter, r *http.Request) {
					if _, err := w.Write(metricsMerger.OutputMetrics); err != nil {
						logger.Error().Err(err).Msg("Failed to write metrics")
					}
					next.ServeHTTP(w, r)
				}
				return http.HandlerFunc(handler)
			}

			decompressedGatewayDMetricsHandler := func() http.Handler {
				return promhttp.InstrumentMetricHandler(
					prometheus.DefaultRegisterer,
					promhttp.HandlerFor(prometheus.DefaultGatherer, promhttp.HandlerOpts{
						DisableCompression: true,
					}),
				)
			}

			logger.Info().Str("address", address).Msg("Metrics are exposed")
			http.Handle(
				metricsConfig.Path,
				gziphandler.GzipHandler(
					mergedMetricsHandler(
						decompressedGatewayDMetricsHandler(),
					),
				),
			)

			//nolint:gosec
			if err = http.ListenAndServe(
				metricsConfig.Address, nil); err != nil {
				logger.Error().Err(err).Msg("Failed to start metrics server")
			}
		}(gConfig.Metrics[config.Default], DefaultLogger)

		// Create a new logger from the config.
		loggerCfg := gConfig.Loggers[config.Default]
		logger := logging.NewLogger(logging.LoggerConfig{
			Output:     loggerCfg.GetOutput(),
			Level:      loggerCfg.GetLevel(),
			TimeFormat: loggerCfg.GetTimeFormat(),
			NoColor:    loggerCfg.NoColor,
			FileName:   loggerCfg.FileName,
		})

		// Replace the default logger with the new one from the config.
		pluginRegistry.Logger = logger
		metricsMerger.Logger = logger

		// This is a notification hook, so we don't care about the result.
		data := map[string]interface{}{
			"output":     loggerCfg.Output,
			"level":      loggerCfg.Level,
			"timeFormat": loggerCfg.TimeFormat,
			"noColor":    loggerCfg.NoColor,
			"fileName":   loggerCfg.FileName,
		}
		// TODO: Use a context with a timeout
		_, err = pluginRegistry.Run(context.Background(), data, plugin.OnNewLogger)
		if err != nil {
			logger.Error().Err(err).Msg("Failed to run OnNewLogger hooks")
		}

		// Create and initialize a pool of connections.
		poolSize := gConfig.Pools[config.Default].GetSize()
		pool := pool.NewPool(poolSize)

		// Get client config from the config file.
		clientConfig := gConfig.Clients[config.Default]

		// Add clients to the pool.
		for i := 0; i < poolSize; i++ {
			client := network.NewClient(&clientConfig, logger)

			if client != nil {
				clientCfg := map[string]interface{}{
					"id":                 client.ID,
					"network":            client.Network,
					"address":            client.Address,
					"receiveBufferSize":  client.ReceiveBufferSize,
					"receiveChunkSize":   client.ReceiveChunkSize,
					"receiveDeadline":    client.ReceiveDeadline.String(),
					"sendDeadline":       client.SendDeadline.String(),
					"tcpKeepAlive":       client.TCPKeepAlive,
					"tcpKeepAlivePeriod": client.TCPKeepAlivePeriod.String(),
				}
				_, err := pluginRegistry.Run(context.Background(), clientCfg, plugin.OnNewClient)
				if err != nil {
					logger.Error().Err(err).Msg("Failed to run OnNewClient hooks")
				}

				err = pool.Put(client.ID, client)
				if err != nil {
					logger.Error().Err(err).Msg("Failed to add client to the pool")
				}
			}
		}

		// Verify that the pool is properly populated.
		logger.Info().Str("count", fmt.Sprint(pool.Size())).Msg(
			"There are clients available in the pool")
		if pool.Size() != poolSize {
			logger.Error().Msg(
				"The pool size is incorrect, either because " +
					"the clients cannot connect due to no network connectivity " +
					"or the server is not running. exiting...")
			pluginRegistry.Shutdown()
			os.Exit(gerr.FailedToInitializePool)
		}

		_, err = pluginRegistry.Run(
			context.Background(),
			map[string]interface{}{"size": poolSize},
			plugin.OnNewPool)
		if err != nil {
			logger.Error().Err(err).Msg("Failed to run OnNewPool hooks")
		}

		// Create a prefork proxy with the pool of clients.
		elastic := gConfig.Proxy[config.Default].Elastic
		reuseElasticClients := gConfig.Proxy[config.Default].ReuseElasticClients
		healthCheckPeriod := gConfig.Proxy[config.Default].HealthCheckPeriod
		proxy := network.NewProxy(
			pool,
			pluginRegistry,
			elastic,
			reuseElasticClients,
			healthCheckPeriod,
			&clientConfig,
			logger,
		)

		proxyCfg := map[string]interface{}{
			"elastic":             elastic,
			"reuseElasticClients": reuseElasticClients,
			"healthCheckPeriod":   healthCheckPeriod.String(),
			"clientConfig": map[string]interface{}{
				"network":            clientConfig.Network,
				"address":            clientConfig.Address,
				"receiveBufferSize":  clientConfig.ReceiveBufferSize,
				"receiveChunkSize":   clientConfig.ReceiveChunkSize,
				"receiveDeadline":    clientConfig.ReceiveDeadline.String(),
				"sendDeadline":       clientConfig.SendDeadline.String(),
				"tcpKeepAlive":       clientConfig.TCPKeepAlive,
				"tcpKeepAlivePeriod": clientConfig.TCPKeepAlivePeriod.String(),
			},
		}
		_, err = pluginRegistry.Run(context.Background(), proxyCfg, plugin.OnNewProxy)
		if err != nil {
			logger.Error().Err(err).Msg("Failed to run OnNewProxy hooks")
		}

		// Create a server
		server := network.NewServer(
			gConfig.Server.Network,
			gConfig.Server.Address,
			gConfig.Server.SoftLimit,
			gConfig.Server.HardLimit,
			gConfig.Server.TickInterval,
			[]gnet.Option{
				// Scheduling options
				gnet.WithMulticore(gConfig.Server.MultiCore),
				gnet.WithLockOSThread(gConfig.Server.LockOSThread),
				// NumEventLoop overrides Multicore option.
				// gnet.WithNumEventLoop(1),

				// Can be used to send keepalive messages to the client.
				gnet.WithTicker(gConfig.Server.EnableTicker),

				// Internal event-loop load balancing options
				gnet.WithLoadBalancing(gConfig.Server.GetLoadBalancer()),

				// Buffer options
				gnet.WithReadBufferCap(gConfig.Server.ReadBufferCap),
				gnet.WithWriteBufferCap(gConfig.Server.WriteBufferCap),
				gnet.WithSocketRecvBuffer(gConfig.Server.SocketRecvBuffer),
				gnet.WithSocketSendBuffer(gConfig.Server.SocketSendBuffer),

				// TCP options
				gnet.WithReuseAddr(gConfig.Server.ReuseAddress),
				gnet.WithReusePort(gConfig.Server.ReusePort),
				gnet.WithTCPKeepAlive(gConfig.Server.TCPKeepAlive),
				gnet.WithTCPNoDelay(gConfig.Server.GetTCPNoDelay()),
			},
			proxy,
			logger,
			pluginRegistry,
		)

		serverCfg := map[string]interface{}{
			"network":          gConfig.Server.Network,
			"address":          gConfig.Server.Address,
			"softLimit":        gConfig.Server.SoftLimit,
			"hardLimit":        gConfig.Server.HardLimit,
			"tickInterval":     gConfig.Server.TickInterval.String(),
			"multiCore":        gConfig.Server.MultiCore,
			"lockOSThread":     gConfig.Server.LockOSThread,
			"enableTicker":     gConfig.Server.EnableTicker,
			"loadBalancer":     gConfig.Server.LoadBalancer,
			"readBufferCap":    gConfig.Server.ReadBufferCap,
			"writeBufferCap":   gConfig.Server.WriteBufferCap,
			"socketRecvBuffer": gConfig.Server.SocketRecvBuffer,
			"socketSendBuffer": gConfig.Server.SocketSendBuffer,
			"reuseAddress":     gConfig.Server.ReuseAddress,
			"reusePort":        gConfig.Server.ReusePort,
			"tcpKeepAlive":     gConfig.Server.TCPKeepAlive.String(),
			"tcpNoDelay":       gConfig.Server.TCPNoDelay,
		}
		_, err = pluginRegistry.Run(context.Background(), serverCfg, plugin.OnNewServer)
		if err != nil {
			logger.Error().Err(err).Msg("Failed to run OnNewServer hooks")
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
		go func(pluginRegistry *plugin.Registry, logger zerolog.Logger, server *network.Server) {
			for sig := range signalsCh {
				for _, s := range signals {
					if sig != s {
						// Notify the hooks that the server is shutting down.
						_, err := pluginRegistry.Run(
							context.Background(),
							map[string]interface{}{"signal": sig.String()},
							plugin.OnSignal,
						)
						if err != nil {
							logger.Error().Err(err).Msg("Failed to run OnSignal hooks")
						}

						metricsMerger.Stop()
						server.Shutdown()
						pluginRegistry.Shutdown()
						os.Exit(0)
					}
				}
			}
		}(pluginRegistry, logger, server)

		// Run the server.
		if err := server.Run(); err != nil {
			logger.Error().Err(err).Msg("Failed to start server")
			metricsMerger.Stop()
			server.Shutdown()
			pluginRegistry.Shutdown()
			os.Exit(gerr.FailedToStartServer)
		}
	},
}

func init() {
	rootCmd.AddCommand(runCmd)

	runCmd.Flags().StringP(
		"config", "c", "./gatewayd.yaml",
		"Global config file")
	runCmd.Flags().StringP(
		"plugin-config", "p", "./gatewayd_plugins.yaml",
		"Plugin config file")
}
