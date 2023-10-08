package network

import (
	"context"
	"net"
	"strconv"
	"time"

	"github.com/gatewayd-io/gatewayd/config"
	gerr "github.com/gatewayd-io/gatewayd/errors"
)

type Option struct {
	EnableTicker bool
}

type Action int

const (
	None Action = iota
	Close
	Shutdown
)

type TCPSocketOpt int

const (
	TCPNoDelay TCPSocketOpt = iota
	TCPDelay
)

type Engine struct {
	listener    net.Listener
	host        string
	port        int
	connections uint32
	stopServer  chan struct{}
}

func (engine *Engine) CountConnections() int {
	return int(engine.connections)
}

func (engine *Engine) Stop(ctx context.Context) error {
	_, cancel := context.WithDeadline(ctx, time.Now().Add(config.DefaultEngineStopTimeout))
	defer cancel()

	engine.stopServer <- struct{}{}
	return nil
}

// Run starts a server and connects all the handlers.
func Run(network, address string, server *Server) *gerr.GatewayDError {
	server.engine = Engine{
		connections: 0,
		stopServer:  make(chan struct{}),
	}

	if action := server.OnBoot(server.engine); action != None {
		return nil
	}

	var err error
	server.engine.listener, err = net.Listen(network, address)
	if err != nil {
		server.logger.Error().Err(err).Msg("Server failed to start listening")
		return gerr.ErrServerListenFailed.Wrap(err)
	}
	defer server.engine.listener.Close()

	if server.engine.listener == nil {
		server.logger.Error().Msg("Server is not properly initialized")
		return nil
	}

	var port string
	server.engine.host, port, err = net.SplitHostPort(server.engine.listener.Addr().String())
	if err != nil {
		server.logger.Error().Err(err).Msg("Failed to split host and port")
		return gerr.ErrSplitHostPortFailed.Wrap(err)
	}

	if server.engine.port, err = strconv.Atoi(port); err != nil {
		server.logger.Error().Err(err).Msg("Failed to convert port to integer")
		return gerr.ErrCastFailed.Wrap(err)
	}

	go func(server *Server) {
		<-server.engine.stopServer
		server.OnShutdown(server.engine)
		server.logger.Debug().Msg("Server stopped")
	}(server)

	go func(server *Server) {
		if !server.Options.EnableTicker {
			return
		}

		for {
			interval, action := server.OnTick()
			if action == Shutdown {
				server.OnShutdown(server.engine)
				return
			}
			if interval == time.Duration(0) {
				return
			}
			time.Sleep(interval)
		}
	}(server)

	for {
		conn, err := server.engine.listener.Accept()
		if err != nil {
			server.logger.Error().Err(err).Msg("Failed to accept connection")
			return gerr.ErrAcceptFailed.Wrap(err)
		}

		if out, action := server.OnOpen(conn); action != None {
			if _, err := conn.Write(out); err != nil {
				server.logger.Error().Err(err).Msg("Failed to write to connection")
			}
			conn.Close()
			if action == Shutdown {
				server.OnShutdown(server.engine)
				return nil
			}
		}
		server.engine.connections++

		// For every new connection, a new unbuffered channel is created to help
		// stop the proxy, recycle the server connection and close stale connections.
		stopConnection := make(chan struct{})
		go func(server *Server, conn net.Conn, stopConnection chan struct{}) {
			if action := server.OnTraffic(conn, stopConnection); action == Close {
				return
			}
		}(server, conn, stopConnection)

		go func(server *Server, conn net.Conn, stopConnection chan struct{}) {
			<-stopConnection
			server.engine.connections--
			if action := server.OnClose(conn, err); action == Close {
				return
			}
		}(server, conn, stopConnection)
	}
}
