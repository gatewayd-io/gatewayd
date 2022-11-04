package network

import (
	"fmt"
	"os"
	"time"

	"github.com/panjf2000/gnet/v2"
	"github.com/sirupsen/logrus"
)

type Status string

const (
	Running Status = "running"
	Stopped Status = "stopped"

	DefaultTickInterval = 5
	DefaultPoolSize     = 10
	DefaultBufferSize   = 4096
)

type Server struct {
	gnet.BuiltinEventEngine
	engine gnet.Engine
	proxy  Proxy

	Network      string // tcp/udp/unix
	Address      string
	Options      []gnet.Option
	SoftLimit    uint64
	HardLimit    uint64
	Status       Status
	TickInterval int
	// PoolSize            int
	// ElasticPool         bool
	// ReuseElasticClients bool
	// BufferSize        int
	OnIncomingTraffic Traffic
	OnOutgoingTraffic Traffic
}

func (s *Server) OnBoot(engine gnet.Engine) gnet.Action {
	s.engine = engine

	// Create a proxy with a fixed/elastic buffer pool
	// s.proxy = NewProxy(s.PoolSize, s.BufferSize, s.ElasticPool, s.ReuseElasticClients)

	// Set the status to running
	s.Status = Running

	return gnet.None
}

func (s *Server) OnOpen(gconn gnet.Conn) ([]byte, gnet.Action) {
	logrus.Debugf("GatewayD is opening a connection from %s", gconn.RemoteAddr().String())
	if uint64(s.engine.CountConnections()) >= s.SoftLimit {
		logrus.Warn("Soft limit reached")
	}
	if uint64(s.engine.CountConnections()) >= s.HardLimit {
		logrus.Error("Hard limit reached")
		_, err := gconn.Write([]byte("Hard limit reached\n"))
		if err != nil {
			logrus.Error(err)
		}
		gconn.Close()
		return nil, gnet.Close
	}

	if err := s.proxy.Connect(gconn); err != nil {
		return nil, gnet.Close
	}

	return nil, gnet.None
}

func (s *Server) OnClose(gconn gnet.Conn, err error) gnet.Action {
	logrus.Debugf("GatewayD is closing a connection from %s", gconn.RemoteAddr().String())

	if err := s.proxy.Disconnect(gconn); err != nil {
		logrus.Error(err)
	}

	if uint64(s.engine.CountConnections()) == 0 && s.Status == Stopped {
		return gnet.Shutdown
	}

	return gnet.Close
}

func (s *Server) OnTraffic(gconn gnet.Conn) gnet.Action {
	if err := s.proxy.PassThrough(gconn, s.OnIncomingTraffic, s.OnOutgoingTraffic); err != nil {
		logrus.Error(err)
		// TODO: Close the connection *gracefully*
		return gnet.Close
	}

	return gnet.None
}

func (s *Server) OnShutdown(engine gnet.Engine) {
	logrus.Println("GatewayD is shutting down...")
	s.proxy.Shutdown()
	s.Status = Stopped
}

func (s *Server) OnTick() (time.Duration, gnet.Action) {
	logrus.Println("GatewayD is ticking...")
	logrus.Infof("Active connections: %d", s.engine.CountConnections())
	return time.Duration(s.TickInterval * int(time.Second)), gnet.None
}

func (s *Server) Run() error {
	logrus.Infof("GatewayD is running with PID %d", os.Getpid())

	// Try to resolve the address and log an error if it can't be resolved
	addr, err := Resolve(s.Network, s.Address)
	if err != nil {
		logrus.Error(err)
	}

	err = gnet.Run(s, s.Network+"://"+addr, s.Options...)
	if err != nil {
		logrus.Error(err)
		return fmt.Errorf("failed to start server: %w", err)
	}

	return nil
}

func (s *Server) Shutdown() {
	s.proxy.Shutdown()
	s.Status = Stopped
}

func (s *Server) IsRunning() bool {
	return s.Status == Running
}

//nolint:funlen
func NewServer(
	network, address string,
	softLimit, hardLimit uint64,
	tickInterval int,
	options []gnet.Option,
	onIncomingTraffic, onOutgoingTraffic Traffic,
	proxy Proxy,
) *Server {
	server := Server{
		Network:      network,
		Address:      address,
		Options:      options,
		TickInterval: tickInterval,
		Status:       Stopped,
	}

	// Try to resolve the address and log an error if it can't be resolved
	addr, err := Resolve(server.Network, server.Address)
	if err != nil {
		logrus.Error(err)
	}

	if addr != "" {
		server.Address = addr
		logrus.Infof("GatewayD is listening on %s", addr)
	} else {
		logrus.Warnf("GatewayD is listening on %s (cannot resolve address)", server.Address)
	}

	// Get the current limits
	limits := GetRLimit()

	// Set the soft and hard limits if they are not set
	if softLimit == 0 {
		server.SoftLimit = limits.Cur
		logrus.Debugf("Soft limit is not set, using the current system soft limit")
	} else {
		server.SoftLimit = softLimit
		logrus.Debugf("Soft limit is set to %d", softLimit)
	}

	if hardLimit == 0 {
		server.HardLimit = limits.Max
		logrus.Debugf("Hard limit is not set, using the current system hard limit")
	} else {
		server.HardLimit = hardLimit
		logrus.Debugf("Hard limit is set to %d", hardLimit)
	}

	if tickInterval == 0 {
		server.TickInterval = int(DefaultTickInterval)
		logrus.Debugf("Tick interval is not set, using the default value")
	} else {
		server.TickInterval = tickInterval
	}

	if onIncomingTraffic == nil {
		server.OnIncomingTraffic = func(gconn gnet.Conn, cl *Client, buf []byte, err error) error {
			// TODO: Implement the traffic handler
			logrus.Infof("GatewayD is passing traffic from %s to %s", gconn.RemoteAddr().String(), gconn.LocalAddr().String())
			return nil
		}
	} else {
		server.OnIncomingTraffic = onIncomingTraffic
	}

	if onOutgoingTraffic == nil {
		server.OnOutgoingTraffic = func(gconn gnet.Conn, cl *Client, buf []byte, err error) error {
			// TODO: Implement the traffic handler
			logrus.Infof("GatewayD is passing traffic from %s to %s", gconn.LocalAddr().String(), gconn.RemoteAddr().String())
			return nil
		}
	} else {
		server.OnOutgoingTraffic = onOutgoingTraffic
	}

	server.proxy = proxy

	return &server
}
