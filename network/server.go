package network

import (
	"context"
	"errors"
	"io"
	"os"
	"time"

	gerr "github.com/gatewayd-io/gatewayd/errors"
	"github.com/gatewayd-io/gatewayd/plugin"
	"github.com/panjf2000/gnet/v2"
	"github.com/rs/zerolog"
)

type Status string

const (
	Running Status = "running"
	Stopped Status = "stopped"

	DefaultTickInterval = 5 * time.Second
	DefaultPoolSize     = 10
	MinimumPoolSize     = 2
	DefaultBufferSize   = 1 << 24 // 16777216 bytes
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
	Status       Status
	TickInterval time.Duration
}

func (s *Server) OnBoot(engine gnet.Engine) gnet.Action {
	s.logger.Debug().Msg("GatewayD is booting...")

	_, err := s.hooksConfig.Run(
		context.Background(),
		map[string]interface{}{"status": string(s.Status)},
		plugin.OnBooting,
		s.hooksConfig.Verification)
	if err != nil {
		s.logger.Error().Err(err).Msg("Failed to run OnBooting hook")
	}

	s.engine = engine

	// Set the status to running
	s.Status = Running

	_, err = s.hooksConfig.Run(
		context.Background(),
		map[string]interface{}{"status": string(s.Status)},
		plugin.OnBooted,
		s.hooksConfig.Verification)
	if err != nil {
		s.logger.Error().Err(err).Msg("Failed to run OnBooted hook")
	}

	s.logger.Debug().Msg("GatewayD booted")

	return gnet.None
}

func (s *Server) OnOpen(gconn gnet.Conn) ([]byte, gnet.Action) {
	s.logger.Debug().Msgf("GatewayD is opening a connection from %s", gconn.RemoteAddr().String())

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

	if err := s.proxy.Connect(gconn); err != nil {
		if errors.Is(err, gerr.ErrPoolExhausted) {
			return nil, gnet.Close
		}

		// This should never happen
		// TODO: Send error to client or retry connection
		s.logger.Error().Err(err).Msg("Failed to connect to proxy")
		return nil, gnet.None
	}

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

func (s *Server) OnClose(gconn gnet.Conn, err error) gnet.Action {
	s.logger.Debug().Msgf("GatewayD is closing a connection from %s", gconn.RemoteAddr().String())

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

	// Shutdown the server if there are no more connections and the server is stopped
	if uint64(s.engine.CountConnections()) == 0 && s.Status == Stopped {
		return gnet.Shutdown
	}

	if err := s.proxy.Disconnect(gconn); err != nil {
		s.logger.Error().Err(err).Msg("Failed to disconnect the server connection")
		return gnet.Close
	}

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

func (s *Server) OnTraffic(gconn gnet.Conn) gnet.Action {
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

	if err := s.proxy.PassThrough(gconn); err != nil {
		s.logger.Error().Err(err).Msg("Failed to pass through traffic")
		switch {
		case errors.Is(err, gerr.ErrPoolExhausted):
		case errors.Is(err, gerr.ErrCastFailed):
		case errors.Is(err, gerr.ErrClientNotFound):
		case errors.Is(err, gerr.ErrClientNotConnected):
		case errors.Is(err, gerr.ErrClientSendFailed):
		case errors.Is(err, gerr.ErrClientReceiveFailed):
		case errors.Is(err.Unwrap(), io.EOF):
			return gnet.Close
		}
	}
	// Flush the connection to make sure all data is sent
	gconn.Flush()

	return gnet.None
}

func (s *Server) OnShutdown(engine gnet.Engine) {
	s.logger.Debug().Msg("GatewayD is shutting down...")

	_, err := s.hooksConfig.Run(
		context.Background(),
		map[string]interface{}{"connections": s.engine.CountConnections()},
		plugin.OnShutdown,
		s.hooksConfig.Verification)
	if err != nil {
		s.logger.Error().Err(err).Msg("Failed to run OnShutdown hook")
	}

	s.proxy.Shutdown()
	s.Status = Stopped
}

func (s *Server) OnTick() (time.Duration, gnet.Action) {
	s.logger.Debug().Msg("GatewayD is ticking...")
	s.logger.Info().Msgf("Active connections: %d", s.engine.CountConnections())

	_, err := s.hooksConfig.Run(
		context.Background(),
		map[string]interface{}{"connections": s.engine.CountConnections()},
		plugin.OnTick,
		s.hooksConfig.Verification)
	if err != nil {
		s.logger.Error().Err(err).Msg("Failed to run OnTick hook")
	}

	return s.TickInterval, gnet.None
}

func (s *Server) Run() error {
	s.logger.Info().Msgf("GatewayD is running with PID %d", os.Getpid())

	// Try to resolve the address and log an error if it can't be resolved
	addr, err := Resolve(s.Network, s.Address, s.logger)
	if err != nil {
		s.logger.Error().Err(err).Msg("Failed to resolve address")
	}

	// Since gnet.Run is blocking, we need to run OnRun before it
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
			s.logger.Error().Msgf("Error in hook: %s", errMsg)
		}

		if address, ok := result["address"].(string); ok {
			addr = address
		}
	}

	origErr := gnet.Run(s, s.Network+"://"+addr, s.Options...)
	if origErr != nil {
		s.logger.Error().Err(origErr).Msg("Failed to start server")
		return gerr.ErrFailedToStartServer.Wrap(origErr)
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
	tickInterval time.Duration,
	options []gnet.Option,
	proxy Proxy,
	logger zerolog.Logger,
	hooksConfig *plugin.HookConfig,
) *Server {
	server := Server{
		Network:      network,
		Address:      address,
		Options:      options,
		TickInterval: tickInterval,
		Status:       Stopped,
	}

	// Try to resolve the address and log an error if it can't be resolved
	addr, err := Resolve(server.Network, server.Address, logger)
	if err != nil {
		logger.Error().Err(err).Msg("Failed to resolve address")
	}

	if addr != "" {
		server.Address = addr
		logger.Debug().Msgf("Resolved address to %s", addr)
		logger.Info().Msgf("GatewayD is listening on %s", addr)
	} else {
		logger.Error().Msg("Failed to resolve address")
		logger.Warn().Msgf("GatewayD is listening on %s (unresolved address)", server.Address)
	}

	// Get the current limits
	limits := GetRLimit(logger)

	// Set the soft and hard limits if they are not set
	if softLimit == 0 {
		server.SoftLimit = limits.Cur
		logger.Debug().Msg("Soft limit is not set, using the current system soft limit")
	} else {
		server.SoftLimit = softLimit
		logger.Debug().Msgf("Soft limit is set to %d", softLimit)
	}

	if hardLimit == 0 {
		server.HardLimit = limits.Max
		logger.Debug().Msg("Hard limit is not set, using the current system hard limit")
	} else {
		server.HardLimit = hardLimit
		logger.Debug().Msgf("Hard limit is set to %d", hardLimit)
	}

	if tickInterval == 0 {
		server.TickInterval = DefaultTickInterval
		logger.Debug().Msg("Tick interval is not set, using the default value")
	} else {
		server.TickInterval = tickInterval
	}

	server.proxy = proxy
	server.logger = logger
	server.hooksConfig = hooksConfig

	return &server
}
