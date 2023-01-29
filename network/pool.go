package network

import (
	"sync"

	"github.com/sirupsen/logrus"
)

type Pool interface {
	ForEach(callback func(client Client) error) error

	GetPool() map[string]Client
	GetClientIDs() []string
	Put(client Client) error
	Pop(ID string) Client
	Size() int
	Close() error
}

type PoolImpl struct {
	mutex sync.Mutex
	pool  map[string]Client
}

var _ Pool = &PoolImpl{}

func (p *PoolImpl) ForEach(callback func(client Client) error) error {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	for _, client := range p.pool {
		if err := callback(client); err != nil {
			return err
		}
	}

	return nil
}

func (p *PoolImpl) GetPool() map[string]Client {
	return p.pool
}

func (p *PoolImpl) GetClientIDs() []string {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	ids := make([]string, 0, len(p.pool))
	for id := range p.pool {
		ids = append(ids, id)
	}
	return ids
}

func (p *PoolImpl) Put(client Client) error {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	p.pool[client.ID] = client
	logrus.Infof("Client %s has been put on the pool", client.ID)

	return nil
}

func (p *PoolImpl) Pop(ID string) Client {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	c := p.pool[ID]
	delete(p.pool, ID)
	logrus.Infof("Client %s has been popped from the pool", ID)

	return c
}

func (p *PoolImpl) Close() error {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	for _, client := range p.pool {
		go func(client Client) {
			client.Close()
		}(client)
	}

	return nil
}

func (p *PoolImpl) Size() int {
	return len(p.pool)
}

func NewPool() *PoolImpl {
	return &PoolImpl{
		pool:  make(map[string]Client),
		mutex: sync.Mutex{},
	}
}
