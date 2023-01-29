package pool

import (
	"sync"

	gerr "github.com/gatewayd-io/gatewayd/errors"
)

type Callback func(key, value interface{}) bool

type Pool interface {
	ForEach(Callback)
	Pool() *sync.Map
	Put(key, value interface{}) *gerr.GatewayDError
	Get(key interface{}) interface{}
	GetOrPut(key, value interface{}) (interface{}, bool, *gerr.GatewayDError)
	Pop(key interface{}) interface{}
	Remove(key interface{})
	Size() int
	Clear()
	Cap() int
}

type Impl struct {
	pool sync.Map
	cap  int
}

var _ Pool = &Impl{}

// ForEach iterates over the pool and calls the callback function for each key/value pair.
func (p *Impl) ForEach(cb Callback) {
	p.pool.Range(cb)
}

// Pool returns the underlying sync.Map.
func (p *Impl) Pool() *sync.Map {
	return &p.pool
}

// Put adds a new key/value pair to the pool.
func (p *Impl) Put(key, value interface{}) *gerr.GatewayDError {
	if p.cap > 0 && p.Size() >= p.cap {
		return gerr.ErrPoolExhausted
	}
	p.pool.Store(key, value)
	return nil
}

// Get returns the value for the given key.
func (p *Impl) Get(key interface{}) interface{} {
	if value, ok := p.pool.Load(key); ok {
		return value
	}
	return nil
}

// GetOrPut returns the value for the given key if it exists, otherwise it adds
// the key/value pair to the pool.
func (p *Impl) GetOrPut(key, value interface{}) (interface{}, bool, *gerr.GatewayDError) {
	if p.cap > 0 && p.Size() >= p.cap {
		return nil, false, gerr.ErrPoolExhausted
	}
	val, loaded := p.pool.LoadOrStore(key, value)
	return val, loaded, nil
}

// Pop removes the key/value pair from the pool and returns the value.
func (p *Impl) Pop(key interface{}) interface{} {
	if p.Size() == 0 {
		return nil
	}
	if value, ok := p.pool.LoadAndDelete(key); ok {
		return value
	}
	return nil
}

// Remove removes the key/value pair from the pool.
func (p *Impl) Remove(key interface{}) {
	if p.Size() == 0 {
		return
	}
	if _, ok := p.pool.Load(key); ok {
		p.pool.Delete(key)
	}
}

// Size returns the number of key/value pairs in the pool.
func (p *Impl) Size() int {
	var size int
	p.pool.Range(func(_, _ interface{}) bool {
		size++
		return true
	})

	return size
}

// Clear removes all key/value pairs from the pool.
func (p *Impl) Clear() {
	p.pool = sync.Map{}
}

// Cap returns the capacity of the pool.
func (p *Impl) Cap() int {
	return p.cap
}

// NewPool creates a new pool with the given capacity.
//
//nolint:predeclared
func NewPool(cap int) *Impl {
	return &Impl{pool: sync.Map{}, cap: cap}
}
