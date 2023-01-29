package cmd

import (
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gatewayd-io/gatewayd/logging"
	"github.com/gatewayd-io/gatewayd/network"
	"github.com/knadh/koanf/parsers/yaml"
	"github.com/knadh/koanf/providers/file"
	"github.com/panjf2000/gnet/v2"
	"github.com/spf13/cobra"
)

const (
	DefaultTCPKeepAlive = 3 * time.Second
)

var (
	configFile string
)

// runCmd represents the run command.
var runCmd = &cobra.Command{
	Use:   "run",
	Short: "Run a gatewayd instance",
	Run: func(cmd *cobra.Command, args []string) {
		if f, err := cmd.Flags().GetString("config"); err == nil {
			if err := konfig.Load(file.Provider(f), yaml.Parser()); err != nil {
				panic(err)
			}
		}

		// Create a new logger from the config
		logger := logging.NewLogger(loggerConfig())

		// Create and initialize a pool of connections
		poolSize, poolClientConfig := poolConfig()
		pool := network.NewPool(logger, poolSize, poolClientConfig)

		// Create a prefork proxy with the pool of clients
		elastic, reuseElasticClients, elasticClientConfig := proxyConfig()
		proxy := network.NewProxy(pool, elastic, reuseElasticClients, elasticClientConfig, logger)

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

				// Logger options
				// TODO: This is a temporary solution and will be replaced.
				// gnet.WithLogger(logrus.New()),
				// gnet.WithLogPath("./gnet.log"),
				// gnet.WithLogLevel(zapcore.DebugLevel),

				// Buffer options
				// TODO: This should be configurable and optimized.
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
			nil,
			nil,
			proxy,
			logger,
		)

		// Shutdown the server gracefully
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
		go func() {
			for sig := range signalsCh {
				for _, s := range signals {
					if sig != s {
						server.Shutdown()
						os.Exit(0)
					}
				}
			}
		}()

		// Run the server
		if err := server.Run(); err != nil {
			logger.Error().Err(err).Msg("Failed to start server")
		}
	},
}

func init() {
	rootCmd.AddCommand(runCmd)

	runCmd.PersistentFlags().StringVarP(
		&configFile, "config", "c", "./gatewayd.yaml", "config file (default is ./gatewayd.yaml)")
}
