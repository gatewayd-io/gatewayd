package network

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/gatewayd-io/gatewayd/plugin"
	"github.com/panjf2000/gnet/v2"
	"github.com/rs/zerolog"
	"google.golang.org/protobuf/types/known/structpb"
)

type Status string

const (
	Running Status = "running"
	Stopped Status = "stopped"

	DefaultTickInterval = 5 * time.Second
	DefaultPoolSize     = 10
	MinimumPoolSize     = 2
	DefaultBufferSize   = 4096
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

	onBootingData, err := structpb.NewStruct(map[string]interface{}{
		"status": string(s.Status),
	})
	if err != nil {
		s.logger.Error().Err(err).Msg("Failed to create structpb")
	} else {
		_, err := s.hooksConfig.Run(
			context.Background(), onBootingData, plugin.OnBooting, s.hooksConfig.Verification)
		if err != nil {
			s.logger.Error().Err(err).Msg("Failed to run OnBooting hook")
		}
	}

	s.engine = engine

	// Set the status to running
	s.Status = Running

	onBootedData, err := structpb.NewStruct(map[string]interface{}{
		"status": string(s.Status),
	})
	if err != nil {
		s.logger.Error().Err(err).Msg("Failed to create structpb")
	} else {
		_, err := s.hooksConfig.Run(
			context.Background(), onBootedData, plugin.OnBooted, s.hooksConfig.Verification)
		if err != nil {
			s.logger.Error().Err(err).Msg("Failed to run OnBooted hook")
		}
	}

	s.logger.Debug().Msg("GatewayD booted")

	return gnet.None
}

func (s *Server) OnOpen(gconn gnet.Conn) ([]byte, gnet.Action) {
	s.logger.Debug().Msgf("GatewayD is opening a connection from %s", gconn.RemoteAddr().String())

	onOpeningData, err := structpb.NewStruct(map[string]interface{}{
		"client": map[string]interface{}{
			"local":  gconn.LocalAddr().String(),
			"remote": gconn.RemoteAddr().String(),
		},
	})
	if err != nil {
		s.logger.Error().Err(err).Msg("Failed to create structpb")
	} else {
		_, err := s.hooksConfig.Run(
			context.Background(), onOpeningData, plugin.OnOpening, s.hooksConfig.Verification)
		if err != nil {
			s.logger.Error().Err(err).Msg("Failed to run OnOpening hook")
		}
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
		gconn.Close()
		return nil, gnet.Close
	}

	if err := s.proxy.Connect(gconn); err != nil {
		return nil, gnet.Close
	}

	onOpenedData, err := structpb.NewStruct(map[string]interface{}{
		"client": map[string]interface{}{
			"local":  gconn.LocalAddr().String(),
			"remote": gconn.RemoteAddr().String(),
		},
	})
	if err != nil {
		s.logger.Error().Err(err).Msg("Failed to create structpb")
	} else {
		_, err := s.hooksConfig.Run(
			context.Background(), onOpenedData, plugin.OnOpened, s.hooksConfig.Verification)
		if err != nil {
			s.logger.Error().Err(err).Msg("Failed to run OnOpened hook")
		}
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

	onClosingData, intErr := structpb.NewStruct(data)
	if intErr != nil {
		s.logger.Error().Err(intErr).Msg("Failed to create structpb")
	} else {
		_, err := s.hooksConfig.Run(
			context.Background(), onClosingData, plugin.OnClosing, s.hooksConfig.Verification)
		if err != nil {
			s.logger.Error().Err(err).Msg("Failed to run OnClosing hook")
		}
	}

	if err := s.proxy.Disconnect(gconn); err != nil {
		s.logger.Error().Err(err).Msg("Failed to disconnect from the client")
	}

	if uint64(s.engine.CountConnections()) == 0 && s.Status == Stopped {
		return gnet.Shutdown
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

	onClosedData, intErr := structpb.NewStruct(data)
	if intErr != nil {
		s.logger.Error().Err(intErr).Msg("Failed to create structpb")
	} else {
		_, err := s.hooksConfig.Run(
			context.Background(), onClosedData, plugin.OnClosed, s.hooksConfig.Verification)
		if err != nil {
			s.logger.Error().Err(err).Msg("Failed to run OnClosed hook")
		}
	}

	return gnet.Close
}

func (s *Server) OnTraffic(gconn gnet.Conn) gnet.Action {
	onTrafficData, err := structpb.NewStruct(map[string]interface{}{
		"client": map[string]interface{}{
			"local":  gconn.LocalAddr().String(),
			"remote": gconn.RemoteAddr().String(),
		},
	})
	if err != nil {
		s.logger.Error().Err(err).Msg("Failed to create structpb")
	} else {
		_, err := s.hooksConfig.Run(
			context.Background(), onTrafficData, plugin.OnTraffic, s.hooksConfig.Verification)
		if err != nil {
			s.logger.Error().Err(err).Msg("Failed to run OnTraffic hook")
		}
	}

	if err := s.proxy.PassThrough(gconn); err != nil {
		s.logger.Error().Err(err).Msg("Failed to pass through traffic")
		// TODO: Close the connection *gracefully*
		return gnet.Close
	}

	return gnet.None
}

func (s *Server) OnShutdown(engine gnet.Engine) {
	s.logger.Debug().Msg("GatewayD is shutting down...")

	onShutdownData, err := structpb.NewStruct(map[string]interface{}{
		"connections": s.engine.CountConnections(),
	})
	if err != nil {
		s.logger.Error().Err(err).Msg("Failed to create structpb")
	} else {
		_, err := s.hooksConfig.Run(
			context.Background(), onShutdownData, plugin.OnShutdown, s.hooksConfig.Verification)
		if err != nil {
			s.logger.Error().Err(err).Msg("Failed to run OnShutdown hook")
		}
	}

	s.proxy.Shutdown()
	s.Status = Stopped
}

func (s *Server) OnTick() (time.Duration, gnet.Action) {
	s.logger.Debug().Msg("GatewayD is ticking...")
	s.logger.Info().Msgf("Active connections: %d", s.engine.CountConnections())

	onTickData, err := structpb.NewStruct(map[string]interface{}{
		"connections": s.engine.CountConnections(),
	})
	if err != nil {
		s.logger.Error().Err(err).Msg("Failed to create structpb")
	} else {
		_, err := s.hooksConfig.Run(
			context.Background(), onTickData, plugin.OnTick, s.hooksConfig.Verification)
		if err != nil {
			s.logger.Error().Err(err).Msg("Failed to run OnTick hook")
		}
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
	//nolint:nestif
	if onRunData, err := structpb.NewStruct(map[string]interface{}{
		"address": addr,
		"error":   err,
	}); err != nil {
		s.logger.Error().Err(err).Msg("Failed to create structpb")
	} else {
		result, err := s.hooksConfig.Run(
			context.Background(), onRunData, plugin.OnRun, s.hooksConfig.Verification)
		if err != nil {
			s.logger.Error().Err(err).Msg("Failed to run the hook")
		}

		if result != nil {
			if err, ok := result.AsMap()["error"].(error); ok && err != nil {
				s.logger.Err(err).Msg("The hook returned an error")
			}

			if address, ok := result.AsMap()["address"].(string); ok {
				addr = address
			}
		}
	}

	err = gnet.Run(s, s.Network+"://"+addr, s.Options...)
	if err != nil {
		s.logger.Error().Err(err).Msg("Failed to start server")
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
		logger.Debug().Msgf("Hard limit is not set, using the current system hard limit")
	} else {
		server.HardLimit = hardLimit
		logger.Debug().Msgf("Hard limit is set to %d", hardLimit)
	}

	if tickInterval == 0 {
		server.TickInterval = DefaultTickInterval
		logger.Debug().Msgf("Tick interval is not set, using the default value")
	} else {
		server.TickInterval = tickInterval
	}

	server.proxy = proxy
	server.logger = logger
	server.hooksConfig = hooksConfig

	return &server
}
