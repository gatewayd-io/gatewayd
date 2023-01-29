package network

import (
	"sync"

	"github.com/sirupsen/logrus"
)

type Pool interface {
	ForEach(callback func(client *Client) error)
	Pool() sync.Map
	ClientIDs() []string
	Put(client *Client) error
	Pop(ID string) *Client
	Size() int
	Close() error
	Shutdown()
}

type PoolImpl struct {
	pool sync.Map
}

var _ Pool = &PoolImpl{}

func (p *PoolImpl) ForEach(callback func(client *Client) error) {
	p.pool.Range(func(key, value interface{}) bool {
		err := callback(value.(*Client))
		if err != nil {
			logrus.Errorf("an error occurred running the callback: %v", err)
		}
		return true
	})
}

func (p *PoolImpl) Pool() sync.Map {
	return p.pool
}

func (p *PoolImpl) ClientIDs() []string {
	var ids []string
	p.pool.Range(func(key, _ interface{}) bool {
		ids = append(ids, key.(string))
		return true
	})
	return ids
}

func (p *PoolImpl) Put(client *Client) error {
	p.pool.Store(client.ID, client)
	logrus.Debugf("Client %s has been put on the pool", client.ID)

	return nil
}

func (p *PoolImpl) Pop(ID string) *Client {
	if client, ok := p.pool.Load(ID); ok {
		p.pool.Delete(ID)
		logrus.Debugf("Client %s has been popped from the pool", ID)
		return client.(*Client)
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
		client := value.(*Client)
		client.Close()
		p.pool.Delete(key)
		return true
	})
}

func NewPool() *PoolImpl {
	return &PoolImpl{pool: sync.Map{}}
}
