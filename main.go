package main

import (
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gatewayd-io/gatewayd/logging"
	"github.com/gatewayd-io/gatewayd/network"
	"github.com/panjf2000/gnet/v2"
	"github.com/rs/zerolog"
)

const (
	DefaultTCPKeepAlive = 3 * time.Second
)

//nolint:funlen
func main() {
	// Create a logger
	logger := logging.NewLogger(nil, zerolog.TimeFormatUnix, zerolog.InfoLevel, true)

	// Create a pool
	pool := network.NewPool(logger)

	// Add a client to the pool
	for i := 0; i < network.DefaultPoolSize; i++ {
		client := network.NewClient("tcp", "localhost:5432", network.DefaultBufferSize, logger)
		if client != nil {
			if err := pool.Put(client); err != nil {
				logger.Panic().Err(err).Msg("Failed to add client to pool")
			}
		}
	}

	// Verify that the pool is properly populated
	logger.Debug().Msgf("There are %d clients in the pool", len(pool.ClientIDs()))
	if len(pool.ClientIDs()) != network.DefaultPoolSize {
		logger.Error().Msg(
			"The pool size is incorrect, either because " +
				"the clients are cannot connect (no network connectivity) " +
				"or the server is not running. Exiting...")
		os.Exit(1)
	}

	// Create a prefork proxy with the pool of clients
	proxy := network.NewProxy(pool, false, false, &network.Client{
		Network:           "tcp",
		Address:           "localhost:5432",
		ReceiveBufferSize: network.DefaultBufferSize,
	}, logger)

	// Create a server
	server := network.NewServer(
		"tcp",
		"0.0.0.0:15432",
		0,
		0,
		network.DefaultTickInterval,
		[]gnet.Option{
			// Scheduling options
			gnet.WithMulticore(true),
			gnet.WithLockOSThread(false),
			// NumEventLoop overrides Multicore option.
			// gnet.WithNumEventLoop(1),

			// Can be used to send keepalive messages to the client.
			gnet.WithTicker(false),

			// Internal event-loop load balancing options
			gnet.WithLoadBalancing(gnet.RoundRobin),

			// Logger options
			// TODO: This is a temporary solution and will be replaced.
			// gnet.WithLogger(logrus.New()),
			// gnet.WithLogPath("./gnet.log"),
			// gnet.WithLogLevel(zapcore.DebugLevel),

			// Buffer options
			// TODO: This should be configurable and optimized.
			gnet.WithReadBufferCap(network.DefaultBufferSize),
			gnet.WithWriteBufferCap(network.DefaultBufferSize),
			gnet.WithSocketRecvBuffer(network.DefaultBufferSize),
			gnet.WithSocketSendBuffer(network.DefaultBufferSize),

			// TCP options
			gnet.WithReuseAddr(true),
			gnet.WithReusePort(true),
			gnet.WithTCPKeepAlive(DefaultTCPKeepAlive),
			gnet.WithTCPNoDelay(gnet.TCPNoDelay),
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
}
