package pool

import (
	"sync"
)

type Callback func(key, value interface{}) bool

type Pool interface {
	ForEach(Callback)
	Pool() *sync.Map
	Put(key, value interface{})
	Get(key interface{}) interface{}
	GetOrPut(key, value interface{}) (interface{}, bool)
	Pop(key interface{}) interface{}
	Remove(key interface{})
	Size() int
	Clear()
}

type PoolImpl struct {
	pool sync.Map
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
}

func (p *PoolImpl) Get(key interface{}) interface{} {
	if value, ok := p.pool.Load(key); ok {
		return value
	}

	return nil
}

func (p *PoolImpl) GetOrPut(key, value interface{}) (interface{}, bool) {
	return p.pool.LoadOrStore(key, value)
}

func (p *PoolImpl) Pop(key interface{}) interface{} {
	if value, ok := p.pool.LoadAndDelete(key); ok {
		return value
	}

	return nil
}

func (p *PoolImpl) Remove(key interface{}) {
	p.pool.Delete(key)
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

func NewPool() *PoolImpl {
	return &PoolImpl{pool: sync.Map{}}
}
