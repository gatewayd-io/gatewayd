package network

import (
	"sync"

	"github.com/rs/zerolog"
)

type Pool interface {
	ForEach(callback func(client *Client) error)
	Pool() *sync.Map
	ClientIDs() []string
	Put(client *Client) error
	Pop(ID string) *Client
	Size() int
	Close() error
	Shutdown()
}

type PoolImpl struct {
	pool   sync.Map
	logger zerolog.Logger
}

var _ Pool = &PoolImpl{}

func (p *PoolImpl) ForEach(callback func(client *Client) error) {
	p.pool.Range(func(key, value interface{}) bool {
		if c, ok := value.(*Client); ok {
			err := callback(c)
			if err != nil {
				p.logger.Debug().Err(err).Msg("an error occurred running the callback")
			}
			return true
		}

		return false
	})
}

func (p *PoolImpl) Pool() *sync.Map {
	return &p.pool
}

func (p *PoolImpl) ClientIDs() []string {
	var ids []string
	p.pool.Range(func(key, _ interface{}) bool {
		if id, ok := key.(string); ok {
			ids = append(ids, id)
			return true
		}
		return false
	})
	return ids
}

func (p *PoolImpl) Put(client *Client) error {
	p.pool.Store(client.ID, client)
	p.logger.Debug().Msgf("Client %s has been put on the pool", client.ID)

	return nil
}

func (p *PoolImpl) Pop(id string) *Client {
	if client, ok := p.pool.Load(id); ok {
		p.pool.Delete(id)
		p.logger.Debug().Msgf("Client %s has been popped from the pool", id)
		if c, ok := client.(*Client); ok {
			return c
		}
		return nil
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

func (p *PoolImpl) Close() error {
	p.ForEach(func(client *Client) error {
		client.Close()
		return nil
	})

	return nil
}

func (p *PoolImpl) Shutdown() {
	p.pool.Range(func(key, value interface{}) bool {
		if cl, ok := value.(*Client); ok {
			cl.Close()
		}
		p.pool.Delete(key)
		return true
	})

	p.pool = sync.Map{}
}

func NewPool(logger zerolog.Logger) *PoolImpl {
	return &PoolImpl{pool: sync.Map{}, logger: logger}
}
