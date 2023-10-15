package network

import (
	"context"
	"net"
	"strconv"
	"sync"
	"sync/atomic"
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
	running     *atomic.Bool
	stopServer  chan struct{}
	mu          *sync.RWMutex
}

func (engine *Engine) CountConnections() int {
	engine.mu.RLock()
	defer engine.mu.RUnlock()
	return int(engine.connections)
}

func (engine *Engine) Stop(ctx context.Context) error {
	_, cancel := context.WithDeadline(ctx, time.Now().Add(config.DefaultEngineStopTimeout))
	defer cancel()

	engine.running.Store(false)
	if err := engine.listener.Close(); err != nil {
		engine.stopServer <- struct{}{}
		close(engine.stopServer)
		return gerr.ErrCloseListenerFailed.Wrap(err)
	}
	engine.stopServer <- struct{}{}
	close(engine.stopServer)
	return nil
}

// Run starts a server and connects all the handlers.
func Run(network, address string, server *Server) *gerr.GatewayDError {
	server.engine = Engine{
		connections: 0,
		stopServer:  make(chan struct{}),
		mu:          &sync.RWMutex{},
		running:     &atomic.Bool{},
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
		server.OnShutdown()
		server.logger.Debug().Msg("Server stopped")
	}(server)

	go func(server *Server) {
		if !server.Options.EnableTicker {
			return
		}

		for {
			select {
			case <-server.engine.stopServer:
				return
			default:
				interval, action := server.OnTick()
				if action == Shutdown {
					server.OnShutdown()
					return
				}
				if interval == time.Duration(0) {
					return
				}
				time.Sleep(interval)
			}
		}
	}(server)

	server.engine.running.Store(true)

	for {
		select {
		case <-server.engine.stopServer:
			server.logger.Info().Msg("Server stopped")
			return nil
		default:
			conn, err := server.engine.listener.Accept()
			if err != nil {
				if !server.engine.running.Load() {
					return nil
				}
				server.logger.Error().Err(err).Msg("Failed to accept connection")
				return gerr.ErrAcceptFailed.Wrap(err)
			}

			if out, action := server.OnOpen(conn); action != None {
				if _, err := conn.Write(out); err != nil {
					server.logger.Error().Err(err).Msg("Failed to write to connection")
				}
				conn.Close()
				if action == Shutdown {
					server.OnShutdown()
					return nil
				}
			}
			server.engine.mu.Lock()
			server.engine.connections++
			server.engine.mu.Unlock()

			// For every new connection, a new unbuffered channel is created to help
			// stop the proxy, recycle the server connection and close stale connections.
			stopConnection := make(chan struct{})
			go func(server *Server, conn net.Conn, stopConnection chan struct{}) {
				if action := server.OnTraffic(conn, stopConnection); action == Close {
					stopConnection <- struct{}{}
				}
			}(server, conn, stopConnection)

			go func(server *Server, conn net.Conn, stopConnection chan struct{}) {
				for {
					select {
					case <-stopConnection:
						server.engine.mu.Lock()
						server.engine.connections--
						server.engine.mu.Unlock()
						server.OnClose(conn, err)
						return
					case <-server.engine.stopServer:
						return
					}
				}
			}(server, conn, stopConnection)
		}
	}
}
