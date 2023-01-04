package cmd

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	gerr "github.com/gatewayd-io/gatewayd/errors"
	"github.com/gatewayd-io/gatewayd/logging"
	"github.com/gatewayd-io/gatewayd/network"
	"github.com/gatewayd-io/gatewayd/plugin"
	"github.com/gatewayd-io/gatewayd/pool"
	"github.com/knadh/koanf/parsers/yaml"
	"github.com/knadh/koanf/providers/confmap"
	"github.com/knadh/koanf/providers/file"
	"github.com/panjf2000/gnet/v2"
	"github.com/rs/zerolog"
	"github.com/spf13/cobra"
)

const (
	DefaultTCPKeepAlive = 3 * time.Second
)

var (
	globalConfigFile string
	pluginConfigFile string
)

var (
	hooksConfig   = plugin.NewHookConfig()
	DefaultLogger = logging.NewLogger(
		logging.LoggerConfig{
			Level:   zerolog.DebugLevel,
			NoColor: true,
		})
	pluginRegistry = plugin.NewRegistry(hooksConfig)
)

// runCmd represents the run command.
var runCmd = &cobra.Command{
	Use:   "run",
	Short: "Run a gatewayd instance",
	Run: func(cmd *cobra.Command, args []string) {
		// The plugins are loaded and hooks registered
		// before the configuration is loaded.
		hooksConfig.Logger = DefaultLogger

		// Load the plugin configuration file
		if f, err := cmd.Flags().GetString("plugin-config"); err == nil {
			if err := pluginConfig.Load(file.Provider(f), yaml.Parser()); err != nil {
				DefaultLogger.Fatal().Err(err).Msg("Failed to load plugin configuration")
				os.Exit(gerr.FailedToLoadPluginConfig)
			}
		}

		// Load plugins and register their hooks
		pluginRegistry.LoadPlugins(pluginConfig)

		if f, err := cmd.Flags().GetString("config"); err == nil {
			if err := globalConfig.Load(file.Provider(f), yaml.Parser()); err != nil {
				DefaultLogger.Fatal().Err(err).Msg("Failed to load configuration")
				pluginRegistry.Shutdown()
				os.Exit(gerr.FailedToLoadGlobalConfig)
			}
		}

		// Get hooks signature verification policy.
		hooksConfig.Verification = verificationPolicy()

		// The config will be passed to the hooks, and in turn to the plugins that
		// register to this hook.
		updatedGlobalConfig, err := hooksConfig.Run(
			context.Background(),
			globalConfig.All(),
			plugin.OnConfigLoaded,
			hooksConfig.Verification)
		if err != nil {
			DefaultLogger.Error().Err(err).Msg("Failed to run OnConfigLoaded hooks")
		}

		if updatedGlobalConfig != nil {
			// Merge the config with the one loaded from the file (in memory).
			// The changes won't be persisted to disk.
			if err := globalConfig.Load(
				confmap.Provider(updatedGlobalConfig, "."), nil); err != nil {
				DefaultLogger.Fatal().Err(err).Msg("Failed to merge configuration")
			}
		}

		// Create a new logger from the config.
		loggerCfg := loggerConfig()
		logger := logging.NewLogger(loggerCfg)

		// Replace the default logger with the new one from the config.
		hooksConfig.Logger = logger

		// This is a notification hook, so we don't care about the result.
		data := map[string]interface{}{
			"timeFormat": loggerCfg.TimeFormat,
			"level":      loggerCfg.Level.String(),
			"noColor":    loggerCfg.NoColor,
		}
		// TODO: Use a context with a timeout
		_, err = hooksConfig.Run(
			context.Background(), data, plugin.OnNewLogger, hooksConfig.Verification)
		if err != nil {
			logger.Error().Err(err).Msg("Failed to run OnNewLogger hooks")
		}

		// Create and initialize a pool of connections.
		poolSize, clientConfig := poolConfig()
		pool := pool.NewPool(poolSize)

		// Add clients to the pool
		for i := 0; i < poolSize; i++ {
			client := network.NewClient(
				clientConfig.Network,
				clientConfig.Address,
				clientConfig.ReceiveBufferSize,
				clientConfig.ReceiveChunkSize,
				clientConfig.ReceiveDeadline,
				clientConfig.SendDeadline,
				clientConfig.TCPKeepAlive,
				clientConfig.TCPKeepAlivePeriod,
				logger,
			)

			if client != nil {
				clientCfg := map[string]interface{}{
					"id":                 client.ID,
					"network":            clientConfig.Network,
					"address":            clientConfig.Address,
					"receiveBufferSize":  clientConfig.ReceiveBufferSize,
					"receiveChunkSize":   clientConfig.ReceiveChunkSize,
					"receiveDeadline":    clientConfig.ReceiveDeadline.Seconds(),
					"sendDeadline":       clientConfig.SendDeadline.Seconds(),
					"tcpKeepAlive":       clientConfig.TCPKeepAlive,
					"tcpKeepAlivePeriod": clientConfig.TCPKeepAlivePeriod.Seconds(),
				}
				_, err := hooksConfig.Run(
					context.Background(),
					clientCfg,
					plugin.OnNewClient,
					hooksConfig.Verification)
				if err != nil {
					logger.Error().Err(err).Msg("Failed to run OnNewClient hooks")
				}

				err = pool.Put(client.ID, client)
				if err != nil {
					logger.Error().Err(err).Msg("Failed to add client to the pool")
				}
			}
		}

		// Verify that the pool is properly populated
		logger.Info().Str("count", fmt.Sprint(pool.Size())).Msg(
			"There are clients available in the pool")
		if pool.Size() != poolSize {
			logger.Error().Msg(
				"The pool size is incorrect, either because " +
					"the clients cannot connect due to no network connectivity " +
					"or the server is not running. exiting...")
			pluginRegistry.Shutdown()
			os.Exit(1)
		}

		_, err = hooksConfig.Run(
			context.Background(),
			map[string]interface{}{"size": poolSize},
			plugin.OnNewPool,
			hooksConfig.Verification)
		if err != nil {
			logger.Error().Err(err).Msg("Failed to run OnNewPool hooks")
		}

		// Create a prefork proxy with the pool of clients.
		elastic, reuseElasticClients, elasticClientConfig := proxyConfig()
		proxy := network.NewProxy(
			pool, hooksConfig, elastic, reuseElasticClients, elasticClientConfig, logger)

		proxyCfg := map[string]interface{}{
			"elastic":             elastic,
			"reuseElasticClients": reuseElasticClients,
			"clientConfig": map[string]interface{}{
				"network":            elasticClientConfig.Network,
				"address":            elasticClientConfig.Address,
				"receiveBufferSize":  elasticClientConfig.ReceiveBufferSize,
				"receiveChunkSize":   elasticClientConfig.ReceiveChunkSize,
				"receiveDeadline":    elasticClientConfig.ReceiveDeadline.Seconds(),
				"sendDeadline":       elasticClientConfig.SendDeadline.Seconds(),
				"tcpKeepAlive":       elasticClientConfig.TCPKeepAlive,
				"tcpKeepAlivePeriod": elasticClientConfig.TCPKeepAlivePeriod.Seconds(),
			},
		}
		_, err = hooksConfig.Run(
			context.Background(), proxyCfg, plugin.OnNewProxy, hooksConfig.Verification)
		if err != nil {
			logger.Error().Err(err).Msg("Failed to run OnNewProxy hooks")
		}

		// Create a server
		serverConfig := serverConfig()
		server := network.NewServer(
			serverConfig.Network,
			serverConfig.Address,
			serverConfig.SoftLimit,
			serverConfig.HardLimit,
			serverConfig.TickInterval,
			[]gnet.Option{
				// Scheduling options
				gnet.WithMulticore(serverConfig.MultiCore),
				gnet.WithLockOSThread(serverConfig.LockOSThread),
				// NumEventLoop overrides Multicore option.
				// gnet.WithNumEventLoop(1),

				// Can be used to send keepalive messages to the client.
				gnet.WithTicker(serverConfig.EnableTicker),

				// Internal event-loop load balancing options
				gnet.WithLoadBalancing(serverConfig.LoadBalancer),

				// Buffer options
				gnet.WithReadBufferCap(serverConfig.ReadBufferCap),
				gnet.WithWriteBufferCap(serverConfig.WriteBufferCap),
				gnet.WithSocketRecvBuffer(serverConfig.SocketRecvBuffer),
				gnet.WithSocketSendBuffer(serverConfig.SocketSendBuffer),

				// TCP options
				gnet.WithReuseAddr(serverConfig.ReuseAddress),
				gnet.WithReusePort(serverConfig.ReusePort),
				gnet.WithTCPKeepAlive(serverConfig.TCPKeepAlive),
				gnet.WithTCPNoDelay(serverConfig.TCPNoDelay),
			},
			proxy,
			logger,
			hooksConfig,
		)

		serverCfg := map[string]interface{}{
			"network":          serverConfig.Network,
			"address":          serverConfig.Address,
			"softLimit":        serverConfig.SoftLimit,
			"hardLimit":        serverConfig.HardLimit,
			"tickInterval":     serverConfig.TickInterval.Seconds(),
			"multiCore":        serverConfig.MultiCore,
			"lockOSThread":     serverConfig.LockOSThread,
			"enableTicker":     serverConfig.EnableTicker,
			"loadBalancer":     int(serverConfig.LoadBalancer),
			"readBufferCap":    serverConfig.ReadBufferCap,
			"writeBufferCap":   serverConfig.WriteBufferCap,
			"socketRecvBuffer": serverConfig.SocketRecvBuffer,
			"socketSendBuffer": serverConfig.SocketSendBuffer,
			"reuseAddress":     serverConfig.ReuseAddress,
			"reusePort":        serverConfig.ReusePort,
			"tcpKeepAlive":     serverConfig.TCPKeepAlive.Seconds(),
			"tcpNoDelay":       int(serverConfig.TCPNoDelay),
		}
		_, err = hooksConfig.Run(
			context.Background(), serverCfg, plugin.OnNewServer, hooksConfig.Verification)
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
		go func(hooksConfig *plugin.HookConfig) {
			for sig := range signalsCh {
				for _, s := range signals {
					if sig != s {
						// Notify the hooks that the server is shutting down.
						_, err := hooksConfig.Run(
							context.Background(),
							map[string]interface{}{"signal": sig.String()},
							plugin.OnSignal,
							hooksConfig.Verification,
						)
						if err != nil {
							logger.Error().Err(err).Msg("Failed to run OnSignal hooks")
						}

						server.Shutdown()
						pluginRegistry.Shutdown()
						os.Exit(0)
					}
				}
			}
		}(hooksConfig)

		// Run the server.
		if err := server.Run(); err != nil {
			logger.Error().Err(err).Msg("Failed to start server")
		}
	},
}

func init() {
	rootCmd.AddCommand(runCmd)

	runCmd.PersistentFlags().StringVarP(
		&globalConfigFile,
		"config", "c", "./gatewayd.yaml",
		"config file (default is ./gatewayd.yaml)")
	runCmd.PersistentFlags().StringVarP(
		&pluginConfigFile,
		"plugin-config", "p", "./gatewayd_plugins.yaml",
		"plugin config file (default is ./gatewayd_plugins.yaml)")
}
