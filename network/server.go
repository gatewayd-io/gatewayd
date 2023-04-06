package network

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"time"

	v1 "github.com/gatewayd-io/gatewayd-plugin-sdk/plugin/v1"
	"github.com/gatewayd-io/gatewayd/config"
	gerr "github.com/gatewayd-io/gatewayd/errors"
	"github.com/gatewayd-io/gatewayd/metrics"
	"github.com/gatewayd-io/gatewayd/plugin"
	"github.com/panjf2000/gnet/v2"
	"github.com/rs/zerolog"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
)

type Server struct {
	gnet.BuiltinEventEngine
	engine         gnet.Engine
	proxy          IProxy
	logger         zerolog.Logger
	pluginRegistry *plugin.Registry
	ctx            context.Context //nolint:containedctx
	pluginTimeout  time.Duration

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
	s.Status = config.Running

	// Run the OnBooted hooks.
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

	return gnet.None
}

// OnOpen is called when a new connection is opened. It calls the OnOpening and OnOpened hooks.
// It also checks if the server is at the soft or hard limit and closes the connection if it is.
func (s *Server) OnOpen(gconn gnet.Conn) ([]byte, gnet.Action) {
	_, span := otel.Tracer("gatewayd").Start(s.ctx, "OnOpen")
	defer span.End()

	s.logger.Debug().Str("from", gconn.RemoteAddr().String()).Msg(
		"GatewayD is opening a connection")

	pluginTimeoutCtx, cancel := context.WithTimeout(context.Background(), s.pluginTimeout)
	defer cancel()
	// Run the OnOpening hooks.
	onOpeningData := map[string]interface{}{
		"client": map[string]interface{}{
			"local":  gconn.LocalAddr().String(),
			"remote": gconn.RemoteAddr().String(),
		},
	}
	_, err := s.pluginRegistry.Run(
		pluginTimeoutCtx, onOpeningData, v1.HookName_HOOK_NAME_ON_OPENING)
	if err != nil {
		s.logger.Error().Err(err).Msg("Failed to run OnOpening hook")
		span.RecordError(err)
	}
	span.AddEvent("Ran the OnOpening hooks")

	// Check if the server is at the soft or hard limit.
	if uint64(s.engine.CountConnections()) >= s.SoftLimit {
		s.logger.Warn().Msg("Soft limit reached")
	}

	if uint64(s.engine.CountConnections()) >= s.HardLimit {
		s.logger.Error().Msg("Hard limit reached")
		_, err := gconn.Write([]byte("Hard limit reached\n"))
		if err != nil {
			s.logger.Error().Err(err).Msg("Failed to write to connection")
			span.RecordError(err)
		}
		return nil, gnet.Close
	}

	// Use the proxy to connect to the backend. Close the connection if the pool is exhausted.
	// This effectively get a connection from the pool and puts both the incoming and the server
	// connections in the pool of the busy connections.
	if err := s.proxy.Connect(gconn); err != nil {
		if errors.Is(err, gerr.ErrPoolExhausted) {
			span.RecordError(err)
			return nil, gnet.Close
		}

		// This should never happen.
		// TODO: Send error to client or retry connection
		s.logger.Error().Err(err).Msg("Failed to connect to proxy")
		span.RecordError(err)
		return nil, gnet.None
	}

	// Run the OnOpened hooks.
	onOpenedData := map[string]interface{}{
		"client": map[string]interface{}{
			"local":  gconn.LocalAddr().String(),
			"remote": gconn.RemoteAddr().String(),
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

	return nil, gnet.None
}

// OnClose is called when a connection is closed. It calls the OnClosing and OnClosed hooks.
// It also recycles the connection back to the available connection pool, unless the pool
// is elastic and reuse is disabled.
func (s *Server) OnClose(gconn gnet.Conn, err error) gnet.Action {
	_, span := otel.Tracer("gatewayd").Start(s.ctx, "OnClose")
	defer span.End()

	s.logger.Debug().Str("from", gconn.RemoteAddr().String()).Msg(
		"GatewayD is closing a connection")

	pluginTimeoutCtx, cancel := context.WithTimeout(context.Background(), s.pluginTimeout)
	defer cancel()
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
	_, gatewaydErr := s.pluginRegistry.Run(
		pluginTimeoutCtx, data, v1.HookName_HOOK_NAME_ON_CLOSING)
	if gatewaydErr != nil {
		s.logger.Error().Err(gatewaydErr).Msg("Failed to run OnClosing hook")
		span.RecordError(gatewaydErr)
	}
	span.AddEvent("Ran the OnClosing hooks")

	// Shutdown the server if there are no more connections and the server is stopped.
	// This is used to shutdown the server gracefully.
	if uint64(s.engine.CountConnections()) == 0 && s.Status == config.Stopped {
		span.AddEvent("Shutting down the server")
		return gnet.Shutdown
	}

	// Disconnect the connection from the proxy. This effectively removes the mapping between
	// the incoming and the server connections in the pool of the busy connections and either
	// recycles or disconnects the connections.
	if err := s.proxy.Disconnect(gconn); err != nil {
		s.logger.Error().Err(err).Msg("Failed to disconnect the server connection")
		span.RecordError(err)
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
	_, gatewaydErr = s.pluginRegistry.Run(
		pluginTimeoutCtx, data, v1.HookName_HOOK_NAME_ON_CLOSED)
	if gatewaydErr != nil {
		s.logger.Error().Err(gatewaydErr).Msg("Failed to run OnClosed hook")
		span.RecordError(gatewaydErr)
	}
	span.AddEvent("Ran the OnClosed hooks")

	metrics.ClientConnections.Dec()

	return gnet.Close
}

// OnTraffic is called when data is received from the client. It calls the OnTraffic hooks.
// It then passes the traffic to the proxied connection.
func (s *Server) OnTraffic(gconn gnet.Conn) gnet.Action {
	_, span := otel.Tracer("gatewayd").Start(s.ctx, "OnTraffic")
	defer span.End()

	pluginTimeoutCtx, cancel := context.WithTimeout(context.Background(), s.pluginTimeout)
	defer cancel()
	// Run the OnTraffic hooks.
	onTrafficData := map[string]interface{}{
		"client": map[string]interface{}{
			"local":  gconn.LocalAddr().String(),
			"remote": gconn.RemoteAddr().String(),
		},
	}
	_, err := s.pluginRegistry.Run(
		pluginTimeoutCtx, onTrafficData, v1.HookName_HOOK_NAME_ON_TRAFFIC)
	if err != nil {
		s.logger.Error().Err(err).Msg("Failed to run OnTraffic hook")
		span.RecordError(err)
	}
	span.AddEvent("Ran the OnTraffic hooks")

	// Pass the traffic from the client to server and vice versa.
	// If there is an error, log it and close the connection.
	if err := s.proxy.PassThrough(gconn); err != nil {
		s.logger.Trace().Err(err).Msg("Failed to pass through traffic")
		span.RecordError(err)
		switch {
		case errors.Is(err, gerr.ErrPoolExhausted),
			errors.Is(err, gerr.ErrCastFailed),
			errors.Is(err, gerr.ErrClientNotFound),
			errors.Is(err, gerr.ErrClientNotConnected),
			errors.Is(err, gerr.ErrClientSendFailed),
			errors.Is(err, gerr.ErrClientReceiveFailed),
			errors.Is(err, gerr.ErrHookTerminatedConnection),
			errors.Is(err.Unwrap(), io.EOF):
			return gnet.Close
		}
	}
	// Flush the connection to make sure all data is sent
	gconn.Flush()

	return gnet.None
}

// OnShutdown is called when the server is shutting down. It calls the OnShutdown hooks.
func (s *Server) OnShutdown(gnet.Engine) {
	_, span := otel.Tracer("gatewayd").Start(s.ctx, "OnShutdown")
	defer span.End()

	s.logger.Debug().Msg("GatewayD is shutting down...")

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
	s.Status = config.Stopped
}

// OnTick is called every TickInterval. It calls the OnTick hooks.
func (s *Server) OnTick() (time.Duration, gnet.Action) {
	_, span := otel.Tracer("gatewayd").Start(s.ctx, "OnTick")
	defer span.End()

	s.logger.Debug().Msg("GatewayD is ticking...")
	s.logger.Info().Str("count", fmt.Sprint(s.engine.CountConnections())).Msg(
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
	return s.TickInterval, gnet.None
}

// Run starts the server and blocks until the server is stopped. It calls the OnRun hooks.
func (s *Server) Run() error {
	_, span := otel.Tracer("gatewayd").Start(s.ctx, "Run")
	defer span.End()

	s.logger.Info().Str("pid", fmt.Sprint(os.Getpid())).Msg("GatewayD is running")

	// Try to resolve the address and log an error if it can't be resolved
	addr, err := Resolve(s.Network, s.Address, s.logger)
	if err != nil {
		s.logger.Error().Err(err).Msg("Failed to resolve address")
		span.RecordError(err)
	}

	pluginTimeoutCtx, cancel := context.WithTimeout(context.Background(), s.pluginTimeout)
	defer cancel()
	// Run the OnRun hooks.
	// Since gnet.Run is blocking, we need to run OnRun before it.
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

	// Start the server.
	origErr := gnet.Run(s, s.Network+"://"+addr, s.Options...)
	if origErr != nil {
		s.logger.Error().Err(origErr).Msg("Failed to start server")
		span.RecordError(origErr)
		return gerr.ErrFailedToStartServer.Wrap(origErr)
	}

	return nil
}

// Shutdown stops the server.
func (s *Server) Shutdown() {
	_, span := otel.Tracer("gatewayd").Start(s.ctx, "Shutdown")
	defer span.End()

	// Shutdown the proxy.
	s.proxy.Shutdown()

	// Set the server status to stopped. This is used to shutdown the server gracefully in OnClose.
	s.Status = config.Stopped

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

	return s.Status == config.Running
}

// NewServer creates a new server.
func NewServer(
	ctx context.Context,
	network, address string,
	softLimit, hardLimit uint64,
	tickInterval time.Duration,
	options []gnet.Option,
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

	return &server
}
