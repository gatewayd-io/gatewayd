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

type Impl struct {
	pool sync.Map
}

var _ Pool = &Impl{}

func (p *Impl) ForEach(cb Callback) {
	p.pool.Range(cb)
}

func (p *Impl) Pool() *sync.Map {
	return &p.pool
}

func (p *Impl) Put(key, value interface{}) {
	p.pool.Store(key, value)
}

func (p *Impl) Get(key interface{}) interface{} {
	if value, ok := p.pool.Load(key); ok {
		return value
	}

	return nil
}

func (p *Impl) GetOrPut(key, value interface{}) (interface{}, bool) {
	return p.pool.LoadOrStore(key, value)
}

func (p *Impl) Pop(key interface{}) interface{} {
	if value, ok := p.pool.LoadAndDelete(key); ok {
		return value
	}

	return nil
}

func (p *Impl) Remove(key interface{}) {
	p.pool.Delete(key)
}

func (p *Impl) Size() int {
	var size int
	p.pool.Range(func(_, _ interface{}) bool {
		size++
		return true
	})

	return size
}

func (p *Impl) Clear() {
	p.pool = sync.Map{}
}

func NewPool() *Impl {
	return &Impl{pool: sync.Map{}}
}
