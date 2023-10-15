package network

import (
	"context"
	"net"
	"sync"
	"sync/atomic"
	"time"

	"github.com/gatewayd-io/gatewayd/config"
	"github.com/rs/zerolog"
)

// Engine is the network engine.
// TODO: Move this to the Server struct.
type Engine struct {
	listener    net.Listener
	host        string
	port        int
	connections uint32
	logger      zerolog.Logger
	running     *atomic.Bool
	stopServer  chan struct{}
	mu          *sync.RWMutex
}

// CountConnections returns the current number of connections.
func (engine *Engine) CountConnections() int {
	engine.mu.RLock()
	defer engine.mu.RUnlock()
	return int(engine.connections)
}

// Stop stops the engine.
func (engine *Engine) Stop(ctx context.Context) error {
	_, cancel := context.WithDeadline(ctx, time.Now().Add(config.DefaultEngineStopTimeout))
	defer cancel()

	var err error
	engine.running.Store(false)
	if engine.listener != nil {
		if err = engine.listener.Close(); err != nil {
			engine.logger.Error().Err(err).Msg("Failed to close listener")
		}
	} else {
		engine.logger.Error().Msg("Listener is not initialized")
	}
	engine.stopServer <- struct{}{}
	close(engine.stopServer)
	return err
}

// NewEngine creates a new engine.
func NewEngine(logger zerolog.Logger) Engine {
	return Engine{
		connections: 0,
		logger:      logger,
		running:     &atomic.Bool{},
		stopServer:  make(chan struct{}),
		mu:          &sync.RWMutex{},
	}
}
