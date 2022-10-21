package main

import (
	"time"

	"github.com/gatewayd-io/gatewayd/network"
	"github.com/panjf2000/gnet/v2"
	"github.com/sirupsen/logrus"
	"go.uber.org/zap/zapcore"
)

func main() {
	// Create a server
	server := &network.Server{
		Network: "unix",
		Address: ":15432",
		Options: []gnet.Option{
			// Scheduling options
			gnet.WithMulticore(true),
			gnet.WithLockOSThread(true),
			// NumEventLoop overrides Multicore option.
			// gnet.WithNumEventLoop(1),

			// Can be used to send keepalive messages to the client.
			gnet.WithTicker(false),

			// Internal event-loop load balancing options
			gnet.WithLoadBalancing(gnet.RoundRobin),

			// Logger options
			// TODO: This is a temporary solution and will be replaced.
			gnet.WithLogger(logrus.New()),
			gnet.WithLogPath("./gnet.log"),
			gnet.WithLogLevel(zapcore.DebugLevel),

			// Buffer options
			// TODO: This should be configurable and optimized.
			gnet.WithReadBufferCap(4096),
			gnet.WithWriteBufferCap(4096),
			gnet.WithSocketRecvBuffer(4096),
			gnet.WithSocketSendBuffer(4096),

			// TCP options
			gnet.WithReuseAddr(true),
			gnet.WithReusePort(true),
			gnet.WithTCPKeepAlive(time.Second * 3),
			gnet.WithTCPNoDelay(gnet.TCPNoDelay),
		},
	}
	server.Run()
}
