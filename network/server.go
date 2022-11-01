package network

import (
	"os"
	"time"

	"github.com/panjf2000/gnet/v2"
	"github.com/sirupsen/logrus"
)

type Status string

const (
	Running Status = "running"
	Stopped Status = "stopped"
)

type Server struct {
	gnet.BuiltinEventEngine
	engine gnet.Engine
	proxy  Proxy

	Network   string // tcp/udp/unix
	Address   string
	Options   []gnet.Option
	SoftLimit int
	HardLimit int
	Status    Status
}

func (s *Server) OnBoot(engine gnet.Engine) gnet.Action {
	s.engine = engine

	// Get the current limits
	limits := GetRLimit()

	// Set the soft and hard limits if they are not set
	if s.SoftLimit == 0 {
		s.SoftLimit = int(limits.Cur)
		logrus.Debugf("Soft limit is not set, using the current system soft limit")
	}

	if s.HardLimit == 0 {
		s.HardLimit = int(limits.Max)
		logrus.Debugf("Hard limit is not set, using the current system hard limit")
	}

	// Create a proxy with a fixed/elastic buffer pool
	s.proxy = NewProxy(10, false, false)

	// Try to resolve the address and log an error if it can't be resolved
	addr, err := Resolve(s.Network, s.Address)
	if err != nil {
		logrus.Error(err)
	}

	if addr != "" {
		logrus.Infof("GatewayD is listening on %s", addr)
	} else {
		logrus.Warnf("GatewayD is listening on %s (cannot resolve address)", addr)
	}

	s.Status = Running

	return gnet.None
}

func (s *Server) OnOpen(gconn gnet.Conn) ([]byte, gnet.Action) {
	logrus.Debugf("GatewayD is opening a connection from %s", gconn.RemoteAddr().String())
	if s.engine.CountConnections() >= s.SoftLimit {
		logrus.Warn("Soft limit reached")
	}
	if s.engine.CountConnections() >= s.HardLimit {
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

	return gnet.Close
}

func (s *Server) OnTraffic(gconn gnet.Conn) gnet.Action {
	if err := s.proxy.PassThrough(gconn, func(buf []byte, err error) error {
		// TODO: Implement the traffic handler
		logrus.Infof("GatewayD is passing traffic from %s to %s", gconn.RemoteAddr().String(), gconn.LocalAddr().String())
		return nil
	}, func(buf []byte, err error) error {
		// TODO: Implement the traffic handler
		logrus.Infof("GatewayD is passing traffic from %s to %s", gconn.LocalAddr().String(), gconn.RemoteAddr().String())
		return nil
	}); err != nil {
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
	return time.Second * 5, gnet.None
}

func (s *Server) Run() {
	logrus.Infof("GatewayD is running with PID %d", os.Getpid())

	// Try to resolve the address and log an error if it can't be resolved
	addr, err := Resolve(s.Network, s.Address)
	if err != nil {
		logrus.Error(err)
	}

	err = gnet.Run(s, s.Network+"://"+addr, s.Options...)
	if err != nil {
		logrus.Error(err)
	}
}

func (s *Server) Shutdown() {
	s.proxy.Shutdown()
}
