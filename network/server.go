package network

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/gatewayd-io/gatewayd/config"
	gerr "github.com/gatewayd-io/gatewayd/errors"
	"github.com/gatewayd-io/gatewayd/plugin"
	"github.com/panjf2000/gnet/v2"
	"github.com/rs/zerolog"
)

type Server struct {
	gnet.BuiltinEventEngine
	engine      gnet.Engine
	proxy       Proxy
	logger      zerolog.Logger
	hooksConfig *plugin.HookConfig

	Network      string // tcp/udp/unix
	Address      string
	Options      []gnet.Option
	SoftLimit    uint64
	HardLimit    uint64
	Status       config.Status
	TickInterval time.Duration
}

// OnBoot is called when the server is booted. It calls the OnBooting and OnBooted hooks.
// It also sets the status to running, which is used to determine if the server should be running
// or shutdown.
func (s *Server) OnBoot(engine gnet.Engine) gnet.Action {
	s.logger.Debug().Msg("GatewayD is booting...")

	// Run the OnBooting hooks.
	_, err := s.hooksConfig.Run(
		context.Background(),
		map[string]interface{}{"status": s.Status},
		plugin.OnBooting,
		s.hooksConfig.Verification)
	if err != nil {
		s.logger.Error().Err(err).Msg("Failed to run OnBooting hook")
	}

	s.engine = engine

	// Set the server status to running.
	s.Status = config.Running

	// Run the OnBooted hooks.
	_, err = s.hooksConfig.Run(
		context.Background(),
		map[string]interface{}{"status": s.Status},
		plugin.OnBooted,
		s.hooksConfig.Verification)
	if err != nil {
		s.logger.Error().Err(err).Msg("Failed to run OnBooted hook")
	}

	s.logger.Debug().Msg("GatewayD booted")

	return gnet.None
}

// OnOpen is called when a new connection is opened. It calls the OnOpening and OnOpened hooks.
// It also checks if the server is at the soft or hard limit and closes the connection if it is.
func (s *Server) OnOpen(gconn gnet.Conn) ([]byte, gnet.Action) {
	s.logger.Debug().Str("from", gconn.RemoteAddr().String()).Msg(
		"GatewayD is opening a connection")

	// Run the OnOpening hooks.
	onOpeningData := map[string]interface{}{
		"client": map[string]interface{}{
			"local":  gconn.LocalAddr().String(),
			"remote": gconn.RemoteAddr().String(),
		},
	}
	_, err := s.hooksConfig.Run(
		context.Background(), onOpeningData, plugin.OnOpening, s.hooksConfig.Verification)
	if err != nil {
		s.logger.Error().Err(err).Msg("Failed to run OnOpening hook")
	}

	// Check if the server is at the soft or hard limit.
	if uint64(s.engine.CountConnections()) >= s.SoftLimit {
		s.logger.Warn().Msg("Soft limit reached")
	}

	if uint64(s.engine.CountConnections()) >= s.HardLimit {
		s.logger.Error().Msg("Hard limit reached")
		_, err := gconn.Write([]byte("Hard limit reached\n"))
		if err != nil {
			s.logger.Error().Err(err).Msg("Failed to write to connection")
		}
		return nil, gnet.Close
	}

	// Use the proxy to connect to the backend. Close the connection if the pool is exhausted.
	// This effectively get a connection from the pool and puts both the incoming and the server
	// connections in the pool of the busy connections.
	if err := s.proxy.Connect(gconn); err != nil {
		if errors.Is(err, gerr.ErrPoolExhausted) {
			return nil, gnet.Close
		}

		// This should never happen.
		// TODO: Send error to client or retry connection
		s.logger.Error().Err(err).Msg("Failed to connect to proxy")
		return nil, gnet.None
	}

	// Run the OnOpened hooks.
	onOpenedData := map[string]interface{}{
		"client": map[string]interface{}{
			"local":  gconn.LocalAddr().String(),
			"remote": gconn.RemoteAddr().String(),
		},
	}
	_, err = s.hooksConfig.Run(
		context.Background(), onOpenedData, plugin.OnOpened, s.hooksConfig.Verification)
	if err != nil {
		s.logger.Error().Err(err).Msg("Failed to run OnOpened hook")
	}

	return nil, gnet.None
}

// OnClose is called when a connection is closed. It calls the OnClosing and OnClosed hooks.
// It also recycles the connection back to the available connection pool, unless the pool
// is elastic and reuse is disabled.
func (s *Server) OnClose(gconn gnet.Conn, err error) gnet.Action {
	s.logger.Debug().Str("from", gconn.RemoteAddr().String()).Msg(
		"GatewayD is closing a connection")

	// Run the OnClosing hooks.
	data := map[string]interface{}{
		"client": map[string]interface{}{
			"local":  gconn.LocalAddr().String(),
			"remote": gconn.RemoteAddr().String(),
		},
		"error": "",
	}
	if err != nil {
		data["error"] = err.Error()
	}
	_, gatewaydErr := s.hooksConfig.Run(
		context.Background(), data, plugin.OnClosing, s.hooksConfig.Verification)
	if gatewaydErr != nil {
		s.logger.Error().Err(gatewaydErr).Msg("Failed to run OnClosing hook")
	}

	// Shutdown the server if there are no more connections and the server is stopped.
	// This is used to shutdown the server gracefully.
	if uint64(s.engine.CountConnections()) == 0 && s.Status == config.Stopped {
		return gnet.Shutdown
	}

	// Disconnect the connection from the proxy. This effectively removes the mapping between
	// the incoming and the server connections in the pool of the busy connections and either
	// recycles or disconnects the connections.
	if err := s.proxy.Disconnect(gconn); err != nil {
		s.logger.Error().Err(err).Msg("Failed to disconnect the server connection")
		return gnet.Close
	}

	// Run the OnClosed hooks.
	data = map[string]interface{}{
		"client": map[string]interface{}{
			"local":  gconn.LocalAddr().String(),
			"remote": gconn.RemoteAddr().String(),
		},
		"error": "",
	}
	if err != nil {
		data["error"] = err.Error()
	}
	_, gatewaydErr = s.hooksConfig.Run(
		context.Background(), data, plugin.OnClosed, s.hooksConfig.Verification)
	if gatewaydErr != nil {
		s.logger.Error().Err(gatewaydErr).Msg("Failed to run OnClosed hook")
	}

	return gnet.Close
}

// OnTraffic is called when data is received from the client. It calls the OnTraffic hooks.
// It then passes the traffic to the proxied connection.
func (s *Server) OnTraffic(gconn gnet.Conn) gnet.Action {
	// Run the OnTraffic hooks.
	onTrafficData := map[string]interface{}{
		"client": map[string]interface{}{
			"local":  gconn.LocalAddr().String(),
			"remote": gconn.RemoteAddr().String(),
		},
	}
	_, err := s.hooksConfig.Run(
		context.Background(), onTrafficData, plugin.OnTraffic, s.hooksConfig.Verification)
	if err != nil {
		s.logger.Error().Err(err).Msg("Failed to run OnTraffic hook")
	}

	// Pass the traffic from the client to server and vice versa.
	// If there is an error, log it and close the connection.
	if err := s.proxy.PassThrough(gconn); err != nil {
		s.logger.Warn().Err(err).Msg("Failed to pass through traffic")
		switch {
		case errors.Is(err, gerr.ErrPoolExhausted),
			errors.Is(err, gerr.ErrCastFailed),
			errors.Is(err, gerr.ErrClientNotFound),
			errors.Is(err, gerr.ErrClientNotConnected),
			errors.Is(err, gerr.ErrClientSendFailed),
			errors.Is(err, gerr.ErrClientReceiveFailed),
			errors.Is(err.Unwrap(), io.EOF):
			return gnet.Close
		}
	}
	// Flush the connection to make sure all data is sent
	gconn.Flush()

	return gnet.None
}

// OnShutdown is called when the server is shutting down. It calls the OnShutdown hooks.
func (s *Server) OnShutdown(engine gnet.Engine) {
	s.logger.Debug().Msg("GatewayD is shutting down...")

	// Run the OnShutdown hooks.
	_, err := s.hooksConfig.Run(
		context.Background(),
		map[string]interface{}{"connections": s.engine.CountConnections()},
		plugin.OnShutdown,
		s.hooksConfig.Verification)
	if err != nil {
		s.logger.Error().Err(err).Msg("Failed to run OnShutdown hook")
	}

	// Shutdown the proxy.
	s.proxy.Shutdown()

	// Set the server status to stopped. This is used to shutdown the server gracefully in OnClose.
	s.Status = config.Stopped
}

// OnTick is called every TickInterval. It calls the OnTick hooks.
func (s *Server) OnTick() (time.Duration, gnet.Action) {
	s.logger.Debug().Msg("GatewayD is ticking...")
	s.logger.Info().Str("count", fmt.Sprint(s.engine.CountConnections())).Msg(
		"Active client connections")

	// Run the OnTick hooks.
	_, err := s.hooksConfig.Run(
		context.Background(),
		map[string]interface{}{"connections": s.engine.CountConnections()},
		plugin.OnTick,
		s.hooksConfig.Verification)
	if err != nil {
		s.logger.Error().Err(err).Msg("Failed to run OnTick hook")
	}

	// TickInterval is the interval at which the OnTick hooks are called. It can be adjusted
	// in the configuration file.
	return s.TickInterval, gnet.None
}

// Run starts the server and blocks until the server is stopped. It calls the OnRun hooks.
func (s *Server) Run() error {
	s.logger.Info().Str("pid", fmt.Sprint(os.Getpid())).Msg("GatewayD is running")

	// Try to resolve the address and log an error if it can't be resolved
	addr, err := Resolve(s.Network, s.Address, s.logger)
	if err != nil {
		s.logger.Error().Err(err).Msg("Failed to resolve address")
	}

	// Run the OnRun hooks.
	// Since gnet.Run is blocking, we need to run OnRun before it.
	onRunData := map[string]interface{}{"address": addr}
	if err != nil && err.Unwrap() != nil {
		onRunData["error"] = err.OriginalError.Error()
	}
	result, err := s.hooksConfig.Run(
		context.Background(), onRunData, plugin.OnRun, s.hooksConfig.Verification)
	if err != nil {
		s.logger.Error().Err(err).Msg("Failed to run the hook")
	}

	if result != nil {
		if errMsg, ok := result["error"].(string); ok && errMsg != "" {
			s.logger.Error().Str("error", errMsg).Msg("Error in hook")
		}

		if address, ok := result["address"].(string); ok {
			addr = address
		}
	}

	// Start the server.
	origErr := gnet.Run(s, s.Network+"://"+addr, s.Options...)
	if origErr != nil {
		s.logger.Error().Err(origErr).Msg("Failed to start server")
		return gerr.ErrFailedToStartServer.Wrap(origErr)
	}

	return nil
}

// Shutdown stops the server.
func (s *Server) Shutdown() {
	// Shutdown the proxy.
	s.proxy.Shutdown()

	// Set the server status to stopped. This is used to shutdown the server gracefully in OnClose.
	s.Status = config.Stopped
}

// IsRunning returns true if the server is running.
func (s *Server) IsRunning() bool {
	return s.Status == config.Running
}

// NewServer creates a new server.
//
//nolint:funlen
func NewServer(
	network, address string,
	softLimit, hardLimit uint64,
	tickInterval time.Duration,
	options []gnet.Option,
	proxy Proxy,
	logger zerolog.Logger,
	hooksConfig *plugin.HookConfig,
) *Server {
	// Create the server.
	server := Server{
		Network:      network,
		Address:      address,
		Options:      options,
		TickInterval: tickInterval,
		Status:       config.Stopped,
	}

	// Try to resolve the address and log an error if it can't be resolved.
	addr, err := Resolve(server.Network, server.Address, logger)
	if err != nil {
		logger.Error().Err(err).Msg("Failed to resolve address")
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

	// Get the current limits.
	limits := GetRLimit(logger)

	// Set the soft and hard limits if they are not set.
	if softLimit == 0 {
		server.SoftLimit = limits.Cur
		logger.Debug().Msg("Soft limit is not set, using the current system soft limit")
	} else {
		server.SoftLimit = softLimit
		logger.Debug().Str("value", fmt.Sprint(softLimit)).Msg("Set soft limit")
	}

	if hardLimit == 0 {
		server.HardLimit = limits.Max
		logger.Debug().Msg("Hard limit is not set, using the current system hard limit")
	} else {
		server.HardLimit = hardLimit
		logger.Debug().Str("value", fmt.Sprint(hardLimit)).Msg("Set hard limit")
	}

	if tickInterval == 0 {
		server.TickInterval = config.DefaultTickInterval
		logger.Debug().Msg("Tick interval is not set, using the default value")
	} else {
		server.TickInterval = tickInterval
	}

	server.proxy = proxy
	server.logger = logger
	server.hooksConfig = hooksConfig

	return &server
}
