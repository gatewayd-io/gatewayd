package network

import (
	"fmt"
	"os"
	"time"

	"github.com/panjf2000/gnet/v2"
	"github.com/rs/zerolog"
)

type Status string

const (
	Running Status = "running"
	Stopped Status = "stopped"

	DefaultTickInterval = 5
	DefaultPoolSize     = 10
	DefaultBufferSize   = 4096
)

type Server struct {
	gnet.BuiltinEventEngine
	engine gnet.Engine
	proxy  Proxy
	logger zerolog.Logger

	Network      string // tcp/udp/unix
	Address      string
	Options      []gnet.Option
	SoftLimit    uint64
	HardLimit    uint64
	Status       Status
	TickInterval int
	// PoolSize            int
	// ElasticPool         bool
	// ReuseElasticClients bool
	// BufferSize        int
	OnIncomingTraffic Traffic
	OnOutgoingTraffic Traffic
}

func (s *Server) OnBoot(engine gnet.Engine) gnet.Action {
	s.engine = engine

	// Set the status to running
	s.Status = Running

	return gnet.None
}

func (s *Server) OnOpen(gconn gnet.Conn) ([]byte, gnet.Action) {
	s.logger.Debug().Msgf("GatewayD is opening a connection from %s", gconn.RemoteAddr().String())

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

	return nil, gnet.None
}

func (s *Server) OnClose(gconn gnet.Conn, err error) gnet.Action {
	s.logger.Debug().Msgf("GatewayD is closing a connection from %s", gconn.RemoteAddr().String())

	if err := s.proxy.Disconnect(gconn); err != nil {
		s.logger.Error().Err(err).Msg("Failed to disconnect from the client")
	}

	if uint64(s.engine.CountConnections()) == 0 && s.Status == Stopped {
		return gnet.Shutdown
	}

	return gnet.Close
}

func (s *Server) OnTraffic(gconn gnet.Conn) gnet.Action {
	if err := s.proxy.PassThrough(gconn, s.OnIncomingTraffic, s.OnOutgoingTraffic); err != nil {
		s.logger.Error().Err(err).Msg("Failed to pass through traffic")
		// TODO: Close the connection *gracefully*
		return gnet.Close
	}

	return gnet.None
}

func (s *Server) OnShutdown(engine gnet.Engine) {
	s.logger.Debug().Msg("GatewayD is shutting down...")
	s.proxy.Shutdown()
	s.Status = Stopped
}

func (s *Server) OnTick() (time.Duration, gnet.Action) {
	s.logger.Debug().Msg("GatewayD is ticking...")
	s.logger.Info().Msgf("Active connections: %d", s.engine.CountConnections())
	return time.Duration(s.TickInterval * int(time.Second)), gnet.None
}

func (s *Server) Run() error {
	s.logger.Info().Msgf("GatewayD is running with PID %d", os.Getpid())

	// Try to resolve the address and log an error if it can't be resolved
	addr, err := Resolve(s.Network, s.Address, s.logger)
	if err != nil {
		s.logger.Error().Err(err).Msg("Failed to resolve address")
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
	tickInterval int,
	options []gnet.Option,
	onIncomingTraffic, onOutgoingTraffic Traffic,
	proxy Proxy,
	logger zerolog.Logger,
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
		server.TickInterval = int(DefaultTickInterval)
		logger.Debug().Msgf("Tick interval is not set, using the default value")
	} else {
		server.TickInterval = tickInterval
	}

	if onIncomingTraffic == nil {
		server.OnIncomingTraffic = func(gconn gnet.Conn, cl *Client, buf []byte, err error) error {
			// TODO: Implement the traffic handler
			logger.Info().Msgf(
				"GatewayD is passing traffic from %s to %s",
				gconn.RemoteAddr().String(),
				gconn.LocalAddr().String())
			return nil
		}
	} else {
		server.OnIncomingTraffic = onIncomingTraffic
	}

	if onOutgoingTraffic == nil {
		server.OnOutgoingTraffic = func(gconn gnet.Conn, cl *Client, buf []byte, err error) error {
			// TODO: Implement the traffic handler
			logger.Info().Msgf(
				"GatewayD is passing traffic from %s to %s",
				gconn.LocalAddr().String(),
				gconn.RemoteAddr().String())
			return nil
		}
	} else {
		server.OnOutgoingTraffic = onOutgoingTraffic
	}

	server.proxy = proxy
	server.logger = logger

	return &server
}
