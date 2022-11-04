package main

import (
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gatewayd-io/gatewayd/network"
	"github.com/panjf2000/gnet/v2"
	"github.com/sirupsen/logrus"
)

const (
	DefaultTCPKeepAlive = 3 * time.Second
)

//nolint:funlen
func main() {
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
		network.NewProxy(network.DefaultPoolSize, network.DefaultBufferSize, false, false),
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
		logrus.Error(err)
	}
}
