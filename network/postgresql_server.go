package network

import (
	"syscall"
	"time"

	"github.com/panjf2000/gnet/v2"
	"github.com/sirupsen/logrus"
)

type PostgreSQLServer struct {
	gnet.BuiltinEventEngine

	Address   string
	engine    gnet.Engine
	Options   []gnet.Option
	SoftLimit int
	HardLimit int
}

func GetRLimit() syscall.Rlimit {
	var limits syscall.Rlimit
	if err := syscall.Getrlimit(syscall.RLIMIT_NOFILE, &limits); err != nil {
		logrus.Error(err)
	}
	logrus.Infof("Current system soft limit: %d", limits.Cur)
	logrus.Infof("Current system hard limit: %d", limits.Max)
	return limits
}

func (p *PostgreSQLServer) OnBoot(engine gnet.Engine) gnet.Action {
	p.engine = engine

	// Get the current limits
	limits := GetRLimit()

	// Set the soft and hard limits if they are not set
	if p.SoftLimit == 0 {
		p.SoftLimit = int(limits.Cur)
		logrus.Info("Soft limit is not set, using the current system soft limit")
	}

	if p.HardLimit == 0 {
		p.HardLimit = int(limits.Max)
		logrus.Info("Hard limit is not set, using the current system hard limit")
	}

	logrus.Infof("PostgreSQL server is listening on %s\n", p.Address)
	return gnet.None
}

func (p *PostgreSQLServer) OnTraffic(c gnet.Conn) gnet.Action {
	buf, _ := c.Next(-1)
	// TODO: parse the buffer and send the response or error
	// The buffer is a PostgreSQL packet
	c.Write([]byte("OK\n"))
	logrus.Infof("Received data: %s", string(buf))
	return gnet.None
}

func (p *PostgreSQLServer) OnShutdown(engine gnet.Engine) {
	logrus.Println("PostgreSQL server is shutting down...")
}

func (p *PostgreSQLServer) OnOpen(c gnet.Conn) (out []byte, action gnet.Action) {
	if p.engine.CountConnections() >= p.SoftLimit {
		logrus.Warn("Soft limit reached")
	}
	if p.engine.CountConnections() >= p.HardLimit {
		logrus.Error("Hard limit reached")
		c.Write([]byte("Hard limit reached\n"))
		c.Close()
		return nil, gnet.Close
	}
	logrus.Infof("PostgreSQL server is opening a connection from %s", c.RemoteAddr().String())
	return []byte("connected\n"), gnet.None
}

func (p *PostgreSQLServer) OnClose(c gnet.Conn, err error) (action gnet.Action) {
	logrus.Infof("PostgreSQL server is closing a connection from %s", c.RemoteAddr().String())
	return gnet.None
}

func (p *PostgreSQLServer) OnTick() (delay time.Duration, action gnet.Action) {
	logrus.Println("PostgreSQL server is ticking...")
	logrus.Infof("Active connections: %d", p.engine.CountConnections())
	return time.Second * 5, gnet.None
}

func (p *PostgreSQLServer) Run() {
	err := gnet.Run(p, p.Address, p.Options...)
	if err != nil {
		logrus.Error(err)
	}
}
