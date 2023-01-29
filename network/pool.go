package network

import (
	"os"
	"sync"

	"github.com/rs/zerolog"
)

type Callback func(key, value interface{}) bool

type Pool interface {
	ForEach(Callback)
	Pool() *sync.Map
	Put(key, value interface{})
	Get(key interface{}) interface{}
	Pop(key interface{}) interface{}
	Size() int
	Clear()
}

type PoolImpl struct {
	pool   sync.Map
	logger zerolog.Logger
}

var _ Pool = &PoolImpl{}

func (p *PoolImpl) ForEach(cb Callback) {
	p.pool.Range(cb)
}

func (p *PoolImpl) Pool() *sync.Map {
	return &p.pool
}

func (p *PoolImpl) Put(key, value interface{}) {
	p.pool.Store(key, value)
	p.logger.Debug().Msg("Item has been put on the pool")
}

func (p *PoolImpl) Get(key interface{}) interface{} {
	if value, ok := p.pool.Load(key); ok {
		return value
	}

	return nil
}

func (p *PoolImpl) Pop(key interface{}) interface{} {
	if value, ok := p.pool.LoadAndDelete(key); ok {
		p.logger.Debug().Msg("Item has been popped from the pool")
		return value
	}

	return nil
}

func (p *PoolImpl) Size() int {
	var size int
	p.pool.Range(func(_, _ interface{}) bool {
		size++
		return true
	})

	return size
}

func (p *PoolImpl) Clear() {
	p.pool = sync.Map{}
}

func NewEmptyPool(logger zerolog.Logger) Pool {
	return &PoolImpl{
		logger: logger,
	}
}

func NewPool(
	logger zerolog.Logger,
	poolSize int,
	clientConfig *Client,
	onNewClient map[Prio]HookDef,
) *PoolImpl {
	pool := PoolImpl{
		pool:   sync.Map{},
		logger: logger,
	}

	// Add a client to the pool
	for i := 0; i < poolSize; i++ {
		client := NewClient(
			clientConfig.Network,
			clientConfig.Address,
			clientConfig.ReceiveBufferSize,
			logger,
		)

		for _, hook := range onNewClient {
			hook(Signature{"client": client})
		}

		if client != nil {
			pool.Put(client.ID, client)
		}
	}

	// Verify that the pool is properly populated
	logger.Info().Msgf("There are %d clients in the pool", pool.Size())
	if pool.Size() != poolSize {
		logger.Error().Msg(
			"The pool size is incorrect, either because " +
				"the clients cannot connect due to no network connectivity " +
				"or the server is not running. exiting...")
		os.Exit(1)
	}

	return &pool
}
