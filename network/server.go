package network

import (
	"context"
	"errors"
	"fmt"
	"net"
	"os"
	"strconv"
	"sync"
	"time"

	v1 "github.com/gatewayd-io/gatewayd-plugin-sdk/plugin/v1"
	"github.com/gatewayd-io/gatewayd/config"
	gerr "github.com/gatewayd-io/gatewayd/errors"
	"github.com/gatewayd-io/gatewayd/metrics"
	"github.com/gatewayd-io/gatewayd/plugin"
	"github.com/rs/zerolog"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
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

type Server struct {
	engine         Engine
	proxy          IProxy
	logger         zerolog.Logger
	pluginRegistry *plugin.Registry
	ctx            context.Context //nolint:containedctx
	pluginTimeout  time.Duration
	mu             *sync.RWMutex

	Network      string // tcp/udp/unix
	Address      string
	Options      Option
	Status       config.Status
	TickInterval time.Duration
}

// OnBoot is called when the server is booted. It calls the OnBooting and OnBooted hooks.
// It also sets the status to running, which is used to determine if the server should be running
// or shutdown.
func (s *Server) OnBoot(engine Engine) Action {
	_, span := otel.Tracer("gatewayd").Start(s.ctx, "OnBoot")
	defer span.End()

	s.logger.Debug().Msg("GatewayD is booting...")

	pluginTimeoutCtx, cancel := context.WithTimeout(context.Background(), s.pluginTimeout)
	defer cancel()
	// Run the OnBooting hooks.
	_, err := s.pluginRegistry.Run(
		pluginTimeoutCtx,
		map[string]interface{}{"status": fmt.Sprint(s.Status)},
		v1.HookName_HOOK_NAME_ON_BOOTING)
	if err != nil {
		s.logger.Error().Err(err).Msg("Failed to run OnBooting hook")
		span.RecordError(err)
	}
	span.AddEvent("Ran the OnBooting hooks")

	s.engine = engine

	// Set the server status to running.
	s.mu.Lock()
	s.Status = config.Running
	s.mu.Unlock()

	// Run the OnBooted hooks.
	pluginTimeoutCtx, cancel = context.WithTimeout(context.Background(), s.pluginTimeout)
	defer cancel()

	_, err = s.pluginRegistry.Run(
		pluginTimeoutCtx,
		map[string]interface{}{"status": fmt.Sprint(s.Status)},
		v1.HookName_HOOK_NAME_ON_BOOTED)
	if err != nil {
		s.logger.Error().Err(err).Msg("Failed to run OnBooted hook")
		span.RecordError(err)
	}
	span.AddEvent("Ran the OnBooted hooks")

	s.logger.Debug().Msg("GatewayD booted")

	return None
}

// OnOpen is called when a new connection is opened. It calls the OnOpening and OnOpened hooks.
// It also checks if the server is at the soft or hard limit and closes the connection if it is.
func (s *Server) OnOpen(conn *ConnWrapper) ([]byte, Action) {
	_, span := otel.Tracer("gatewayd").Start(s.ctx, "OnOpen")
	defer span.End()

	s.logger.Debug().Str("from", RemoteAddr(conn.Conn())).Msg(
		"GatewayD is opening a connection")

	pluginTimeoutCtx, cancel := context.WithTimeout(context.Background(), s.pluginTimeout)
	defer cancel()
	// Run the OnOpening hooks.
	onOpeningData := map[string]interface{}{
		"client": map[string]interface{}{
			"local":  LocalAddr(conn.Conn()),
			"remote": RemoteAddr(conn.Conn()),
		},
	}
	_, err := s.pluginRegistry.Run(
		pluginTimeoutCtx, onOpeningData, v1.HookName_HOOK_NAME_ON_OPENING)
	if err != nil {
		s.logger.Error().Err(err).Msg("Failed to run OnOpening hook")
		span.RecordError(err)
	}
	span.AddEvent("Ran the OnOpening hooks")

	// Use the proxy to connect to the backend. Close the connection if the pool is exhausted.
	// This effectively get a connection from the pool and puts both the incoming and the server
	// connections in the pool of the busy connections.
	if err := s.proxy.Connect(conn); err != nil {
		if errors.Is(err, gerr.ErrPoolExhausted) {
			span.RecordError(err)
			return nil, Close
		}

		// This should never happen.
		// TODO: Send error to client or retry connection
		s.logger.Error().Err(err).Msg("Failed to connect to proxy")
		span.RecordError(err)
		return nil, None
	}

	// Run the OnOpened hooks.
	pluginTimeoutCtx, cancel = context.WithTimeout(context.Background(), s.pluginTimeout)
	defer cancel()

	onOpenedData := map[string]interface{}{
		"client": map[string]interface{}{
			"local":  LocalAddr(conn.Conn()),
			"remote": RemoteAddr(conn.Conn()),
		},
	}
	_, err = s.pluginRegistry.Run(
		pluginTimeoutCtx, onOpenedData, v1.HookName_HOOK_NAME_ON_OPENED)
	if err != nil {
		s.logger.Error().Err(err).Msg("Failed to run OnOpened hook")
		span.RecordError(err)
	}
	span.AddEvent("Ran the OnOpened hooks")

	metrics.ClientConnections.Inc()

	return nil, None
}

// OnClose is called when a connection is closed. It calls the OnClosing and OnClosed hooks.
// It also recycles the connection back to the available connection pool, unless the pool
// is elastic and reuse is disabled.
func (s *Server) OnClose(conn *ConnWrapper, err error) Action {
	_, span := otel.Tracer("gatewayd").Start(s.ctx, "OnClose")
	defer span.End()

	s.logger.Debug().Str("from", RemoteAddr(conn.Conn())).Msg(
		"GatewayD is closing a connection")

	// Run the OnClosing hooks.
	pluginTimeoutCtx, cancel := context.WithTimeout(context.Background(), s.pluginTimeout)
	defer cancel()

	data := map[string]interface{}{
		"client": map[string]interface{}{
			"local":  LocalAddr(conn.Conn()),
			"remote": RemoteAddr(conn.Conn()),
		},
		"error": "",
	}
	if err != nil {
		data["error"] = err.Error()
	}
	_, gatewaydErr := s.pluginRegistry.Run(
		pluginTimeoutCtx, data, v1.HookName_HOOK_NAME_ON_CLOSING)
	if gatewaydErr != nil {
		s.logger.Error().Err(gatewaydErr).Msg("Failed to run OnClosing hook")
		span.RecordError(gatewaydErr)
	}
	span.AddEvent("Ran the OnClosing hooks")

	// Shutdown the server if there are no more connections and the server is stopped.
	// This is used to shut down the server gracefully.
	s.mu.Lock()
	if uint64(s.engine.CountConnections()) == 0 && s.Status == config.Stopped {
		span.AddEvent("Shutting down the server")
		s.mu.Unlock()
		return Shutdown
	}
	s.mu.Unlock()

	// Disconnect the connection from the proxy. This effectively removes the mapping between
	// the incoming and the server connections in the pool of the busy connections and either
	// recycles or disconnects the connections.
	if err := s.proxy.Disconnect(conn); err != nil {
		s.logger.Error().Err(err).Msg("Failed to disconnect the server connection")
		span.RecordError(err)
		return Close
	}

	// Close the incoming connection.
	if err := conn.Close(); err != nil {
		s.logger.Error().Err(err).Msg("Failed to close the incoming connection")
		span.RecordError(err)
		return Close
	}

	// Run the OnClosed hooks.
	pluginTimeoutCtx, cancel = context.WithTimeout(context.Background(), s.pluginTimeout)
	defer cancel()

	data = map[string]interface{}{
		"client": map[string]interface{}{
			"local":  LocalAddr(conn.Conn()),
			"remote": RemoteAddr(conn.Conn()),
		},
		"error": "",
	}
	if err != nil {
		data["error"] = err.Error()
	}
	_, gatewaydErr = s.pluginRegistry.Run(
		pluginTimeoutCtx, data, v1.HookName_HOOK_NAME_ON_CLOSED)
	if gatewaydErr != nil {
		s.logger.Error().Err(gatewaydErr).Msg("Failed to run OnClosed hook")
		span.RecordError(gatewaydErr)
	}
	span.AddEvent("Ran the OnClosed hooks")

	metrics.ClientConnections.Dec()

	return Close
}

// OnTraffic is called when data is received from the client. It calls the OnTraffic hooks.
// It then passes the traffic to the proxied connection.
func (s *Server) OnTraffic(conn *ConnWrapper, stopConnection chan struct{}) Action {
	_, span := otel.Tracer("gatewayd").Start(s.ctx, "OnTraffic")
	defer span.End()

	// Run the OnTraffic hooks.
	pluginTimeoutCtx, cancel := context.WithTimeout(context.Background(), s.pluginTimeout)
	defer cancel()

	onTrafficData := map[string]interface{}{
		"client": map[string]interface{}{
			"local":  LocalAddr(conn.Conn()),
			"remote": RemoteAddr(conn.Conn()),
		},
	}
	_, err := s.pluginRegistry.Run(
		pluginTimeoutCtx, onTrafficData, v1.HookName_HOOK_NAME_ON_TRAFFIC)
	if err != nil {
		s.logger.Error().Err(err).Msg("Failed to run OnTraffic hook")
		span.RecordError(err)
	}
	span.AddEvent("Ran the OnTraffic hooks")

	stack := NewStack()

	// Pass the traffic from the client to server.
	// If there is an error, log it and close the connection.
	go func(server *Server, conn *ConnWrapper, stopConnection chan struct{}, stack *Stack) {
		for {
			server.logger.Trace().Msg("Passing through traffic from client to server")
			if err := server.proxy.PassThroughToServer(conn, stack); err != nil {
				server.logger.Trace().Err(err).Msg("Failed to pass through traffic")
				span.RecordError(err)
				stopConnection <- struct{}{}
				break
			}
		}
	}(s, conn, stopConnection, stack)

	// Pass the traffic from the server to client.
	// If there is an error, log it and close the connection.
	go func(server *Server, conn *ConnWrapper, stopConnection chan struct{}, stack *Stack) {
		for {
			server.logger.Debug().Msg("Passing through traffic from server to client")
			if err := server.proxy.PassThroughToClient(conn, stack); err != nil {
				server.logger.Trace().Err(err).Msg("Failed to pass through traffic")
				span.RecordError(err)
				stopConnection <- struct{}{}
				break
			}
		}
	}(s, conn, stopConnection, stack)

	<-stopConnection
	stack.Clear()

	return Close
}

// OnShutdown is called when the server is shutting down. It calls the OnShutdown hooks.
func (s *Server) OnShutdown() {
	_, span := otel.Tracer("gatewayd").Start(s.ctx, "OnShutdown")
	defer span.End()

	s.logger.Debug().Msg("GatewayD is shutting down")

	pluginTimeoutCtx, cancel := context.WithTimeout(context.Background(), s.pluginTimeout)
	defer cancel()
	// Run the OnShutdown hooks.
	_, err := s.pluginRegistry.Run(
		pluginTimeoutCtx,
		map[string]interface{}{"connections": s.engine.CountConnections()},
		v1.HookName_HOOK_NAME_ON_SHUTDOWN)
	if err != nil {
		s.logger.Error().Err(err).Msg("Failed to run OnShutdown hook")
		span.RecordError(err)
	}
	span.AddEvent("Ran the OnShutdown hooks")

	// Shutdown the proxy.
	s.proxy.Shutdown()

	// Set the server status to stopped. This is used to shutdown the server gracefully in OnClose.
	s.mu.Lock()
	s.Status = config.Stopped
	s.mu.Unlock()
}

// OnTick is called every TickInterval. It calls the OnTick hooks.
func (s *Server) OnTick() (time.Duration, Action) {
	_, span := otel.Tracer("gatewayd").Start(s.ctx, "OnTick")
	defer span.End()

	s.logger.Debug().Msg("GatewayD is ticking...")
	s.logger.Info().Str("count", strconv.Itoa(s.engine.CountConnections())).Msg(
		"Active client connections")

	pluginTimeoutCtx, cancel := context.WithTimeout(context.Background(), s.pluginTimeout)
	defer cancel()
	// Run the OnTick hooks.
	_, err := s.pluginRegistry.Run(
		pluginTimeoutCtx,
		map[string]interface{}{"connections": s.engine.CountConnections()},
		v1.HookName_HOOK_NAME_ON_TICK)
	if err != nil {
		s.logger.Error().Err(err).Msg("Failed to run OnTick hook")
		span.RecordError(err)
	}
	span.AddEvent("Ran the OnTick hooks")

	// TODO: Investigate whether to move schedulers here or not

	metrics.ServerTicksFired.Inc()

	// TickInterval is the interval at which the OnTick hooks are called. It can be adjusted
	// in the configuration file.
	return s.TickInterval, None
}

// Run starts the server and blocks until the server is stopped. It calls the OnRun hooks.
func (s *Server) Run() *gerr.GatewayDError {
	_, span := otel.Tracer("gatewayd").Start(s.ctx, "Run")
	defer span.End()

	s.logger.Info().Str("pid", strconv.Itoa(os.Getpid())).Msg("GatewayD is running")

	// Try to resolve the address and log an error if it can't be resolved
	addr, err := Resolve(s.Network, s.Address, s.logger)
	if err != nil {
		s.logger.Error().Err(err).Msg("Failed to resolve address")
		span.RecordError(err)
	}

	pluginTimeoutCtx, cancel := context.WithTimeout(context.Background(), s.pluginTimeout)
	defer cancel()
	// Run the OnRun hooks.
	// Since Run is blocking, we need to run OnRun before it.
	onRunData := map[string]interface{}{"address": addr}
	if err != nil && err.Unwrap() != nil {
		onRunData["error"] = err.OriginalError.Error()
	}
	result, err := s.pluginRegistry.Run(
		pluginTimeoutCtx, onRunData, v1.HookName_HOOK_NAME_ON_RUN)
	if err != nil {
		s.logger.Error().Err(err).Msg("Failed to run the hook")
		span.RecordError(err)
	}
	span.AddEvent("Ran the OnRun hooks")

	if result != nil {
		if errMsg, ok := result["error"].(string); ok && errMsg != "" {
			s.logger.Error().Str("error", errMsg).Msg("Error in hook")
		}

		if address, ok := result["address"].(string); ok {
			addr = address
		}
	}

	if action := s.OnBoot(s.engine); action != None {
		return nil
	}

	listener, origErr := net.Listen(s.Network, addr)
	if origErr != nil {
		s.logger.Error().Err(origErr).Msg("Server failed to start listening")
		return gerr.ErrServerListenFailed.Wrap(origErr)
	}
	s.mu.Lock()
	s.engine.listener = listener
	s.mu.Unlock()
	defer s.engine.listener.Close()

	if s.engine.listener == nil {
		s.logger.Error().Msg("Server is not properly initialized")
		return nil
	}

	var port string
	s.engine.host, port, origErr = net.SplitHostPort(s.engine.listener.Addr().String())
	if origErr != nil {
		s.logger.Error().Err(origErr).Msg("Failed to split host and port")
		return gerr.ErrSplitHostPortFailed.Wrap(origErr)
	}

	if s.engine.port, origErr = strconv.Atoi(port); origErr != nil {
		s.logger.Error().Err(origErr).Msg("Failed to convert port to integer")
		return gerr.ErrCastFailed.Wrap(origErr)
	}

	go func(server *Server) {
		<-server.engine.stopServer
		server.OnShutdown()
		server.logger.Debug().Msg("Server stopped")
	}(s)

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
	}(s)

	s.engine.running.Store(true)

	for {
		select {
		case <-s.engine.stopServer:
			s.logger.Info().Msg("Server stopped")
			return nil
		default:
			netConn, err := s.engine.listener.Accept()
			if err != nil {
				if !s.engine.running.Load() {
					return nil
				}
				s.logger.Error().Err(err).Msg("Failed to accept connection")
				return gerr.ErrAcceptFailed.Wrap(err)
			}

			conn, err := NewConnWrapper(netConn, nil)

			if out, action := s.OnOpen(conn); action != None {
				if _, err := conn.Write(out); err != nil {
					s.logger.Error().Err(err).Msg("Failed to write to connection")
				}
				conn.Close()
				if action == Shutdown {
					s.OnShutdown()
					return nil
				}
			}
			s.engine.mu.Lock()
			s.engine.connections++
			s.engine.mu.Unlock()

			// For every new connection, a new unbuffered channel is created to help
			// stop the proxy, recycle the server connection and close stale connections.
			stopConnection := make(chan struct{})
			go func(server *Server, conn *ConnWrapper, stopConnection chan struct{}) {
				if action := server.OnTraffic(conn, stopConnection); action == Close {
					stopConnection <- struct{}{}
				}
			}(s, conn, stopConnection)

			go func(server *Server, conn *ConnWrapper, stopConnection chan struct{}) {
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
			}(s, conn, stopConnection)
		}
	}
}

// Shutdown stops the server.
func (s *Server) Shutdown() {
	_, span := otel.Tracer("gatewayd").Start(s.ctx, "Shutdown")
	defer span.End()

	// Shutdown the proxy.
	s.proxy.Shutdown()

	// Set the server status to stopped. This is used to shutdown the server gracefully in OnClose.
	s.mu.Lock()
	s.Status = config.Stopped
	s.mu.Unlock()

	// Shutdown the server.
	if err := s.engine.Stop(context.Background()); err != nil {
		s.logger.Error().Err(err).Msg("Failed to shutdown server")
		span.RecordError(err)
	}
}

// IsRunning returns true if the server is running.
func (s *Server) IsRunning() bool {
	_, span := otel.Tracer("gatewayd").Start(s.ctx, "IsRunning")
	defer span.End()
	span.SetAttributes(attribute.Bool("status", s.Status == config.Running))

	s.mu.Lock()
	defer s.mu.Unlock()
	return s.Status == config.Running
}

// NewServer creates a new server.
func NewServer(
	ctx context.Context,
	network, address string,
	tickInterval time.Duration,
	options Option,
	proxy IProxy,
	logger zerolog.Logger,
	pluginRegistry *plugin.Registry,
	pluginTimeout time.Duration,
) *Server {
	serverCtx, span := otel.Tracer(config.TracerName).Start(ctx, "NewServer")
	defer span.End()

	// Create the server.
	server := Server{
		ctx:            serverCtx,
		Network:        network,
		Address:        address,
		Options:        options,
		TickInterval:   tickInterval,
		Status:         config.Stopped,
		proxy:          proxy,
		logger:         logger,
		pluginRegistry: pluginRegistry,
		pluginTimeout:  pluginTimeout,
		mu:             &sync.RWMutex{},
		engine:         NewEngine(logger),
	}

	// Try to resolve the address and log an error if it can't be resolved.
	addr, err := Resolve(server.Network, server.Address, logger)
	if err != nil {
		logger.Error().Err(err).Msg("Failed to resolve address")
		span.AddEvent(err.Error())
	}

	if addr != "" {
		server.Address = addr
		logger.Debug().Str("address", addr).Msg("Resolved address")
		logger.Info().Str("address", addr).Msg("GatewayD is listening")
	} else {
		logger.Error().Msg("Failed to resolve address")
		logger.Warn().Str("address", server.Address).Msg(
			"GatewayD is listening on an unresolved address")
	}

	return &server
}
