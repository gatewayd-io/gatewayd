package network

import (
	"time"

	"github.com/panjf2000/gnet/v2"
	"github.com/sirupsen/logrus"
)

type PostgreSQLServer struct {
	gnet.BuiltinEventEngine

	Address string
	engine  gnet.Engine
	Options []gnet.Option
}

func (p *PostgreSQLServer) OnBoot(engine gnet.Engine) gnet.Action {
	p.engine = engine
	logrus.Printf("PostgreSQL server is listening on %s\n", p.Address)
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
	logrus.Printf("PostgreSQL server is opening a connection from %s", c.RemoteAddr().String())
	return []byte("connected\n"), gnet.None
}

func (p *PostgreSQLServer) OnClose(c gnet.Conn, err error) (action gnet.Action) {
	logrus.Printf("PostgreSQL server is closing a connection from %s", c.RemoteAddr().String())
	return gnet.None
}

func (p *PostgreSQLServer) OnTick() (delay time.Duration, action gnet.Action) {
	logrus.Println("PostgreSQL server is ticking...")
	logrus.Printf("Active connections: %d", p.engine.CountConnections())
	return time.Second * 5, gnet.None
}

func (p *PostgreSQLServer) Run() {
	err := gnet.Run(p, p.Address, p.Options...)
	if err != nil {
		logrus.Error(err)
	}
}
