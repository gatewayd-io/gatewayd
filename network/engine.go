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
	logger      zerolog.Logger
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
	if engine.listener != nil {
		if err := engine.listener.Close(); err != nil {
			engine.logger.Error().Err(err).Msg("Failed to close listener")
		}
	} else {
		engine.logger.Error().Msg("Listener is not initialized")
	}
	engine.stopServer <- struct{}{}
	close(engine.stopServer)
	return nil
}
