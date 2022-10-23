package network

import (
	"time"

	"github.com/panjf2000/gnet/v2"
	"github.com/sirupsen/logrus"
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
}

func (s *Server) OnBoot(engine gnet.Engine) gnet.Action {
	s.engine = engine

	// Get the current limits
	limits := GetRLimit()

	// Set the soft and hard limits if they are not set
	if s.SoftLimit == 0 {
		s.SoftLimit = int(limits.Cur)
		logrus.Info("Soft limit is not set, using the current system soft limit")
	}

	if s.HardLimit == 0 {
		s.HardLimit = int(limits.Max)
		logrus.Info("Hard limit is not set, using the current system hard limit")
	}

	// Create a proxy with an elastic buffer pool
	s.proxy = NewProxy(10, true, false)

	logrus.Infof("PostgreSQL server is listening on %s\n", s.Address)
	return gnet.None
}

func (s *Server) OnOpen(c gnet.Conn) (out []byte, action gnet.Action) {
	logrus.Infof("PostgreSQL server is opening a connection from %s", c.RemoteAddr().String())
	if s.engine.CountConnections() >= s.SoftLimit {
		logrus.Warn("Soft limit reached")
	}
	if s.engine.CountConnections() >= s.HardLimit {
		logrus.Error("Hard limit reached")
		c.Write([]byte("Hard limit reached\n"))
		c.Close()
		return nil, gnet.Close
	}

	s.proxy.Connect(c)

	return nil, gnet.None
}

func (s *Server) OnClose(c gnet.Conn, err error) (action gnet.Action) {
	logrus.Infof("PostgreSQL server is closing a connection from %s", c.RemoteAddr().String())

	s.proxy.Disconnect(c)

	return gnet.Close
}

func (s *Server) OnTraffic(c gnet.Conn) gnet.Action {
	if err := s.proxy.PassThrough(c); err != nil {
		logrus.Error(err)
	}

	// logrus.Infof("Received data: %s", string(buf))
	return gnet.None
}

func (s *Server) OnShutdown(engine gnet.Engine) {
	logrus.Println("PostgreSQL server is shutting down...")
	s.proxy.Shutdown()
}

func (s *Server) OnTick() (delay time.Duration, action gnet.Action) {
	logrus.Println("PostgreSQL server is ticking...")
	logrus.Infof("Active connections: %d", s.engine.CountConnections())
	return time.Second * 5, gnet.None
}

func (s *Server) Run() {
	err := gnet.Run(s, s.Network+"://"+s.Address, s.Options...)
	if err != nil {
		logrus.Error(err)
	}
}
