package pool

import (
	"sync"

	gerr "github.com/gatewayd-io/gatewayd/errors"
)

type Callback func(key, value interface{}) bool

type IPool interface {
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

type Pool struct {
	pool sync.Map
	cap  int
}

var _ IPool = &Pool{}

// ForEach iterates over the pool and calls the callback function for each key/value pair.
func (p *Pool) ForEach(cb Callback) {
	p.pool.Range(cb)
}

// Pool returns the underlying sync.Map.
func (p *Pool) Pool() *sync.Map {
	return &p.pool
}

// Put adds a new key/value pair to the pool.
func (p *Pool) Put(key, value interface{}) *gerr.GatewayDError {
	if p.cap > 0 && p.Size() >= p.cap {
		return gerr.ErrPoolExhausted
	}
	p.pool.Store(key, value)
	return nil
}

// Get returns the value for the given key.
func (p *Pool) Get(key interface{}) interface{} {
	if value, ok := p.pool.Load(key); ok {
		return value
	}
	return nil
}

// GetOrPut returns the value for the given key if it exists, otherwise it adds
// the key/value pair to the pool.
func (p *Pool) GetOrPut(key, value interface{}) (interface{}, bool, *gerr.GatewayDError) {
	if p.cap > 0 && p.Size() >= p.cap {
		return nil, false, gerr.ErrPoolExhausted
	}
	val, loaded := p.pool.LoadOrStore(key, value)
	return val, loaded, nil
}

// Pop removes the key/value pair from the pool and returns the value.
func (p *Pool) Pop(key interface{}) interface{} {
	if p.Size() == 0 {
		return nil
	}
	if value, ok := p.pool.LoadAndDelete(key); ok {
		return value
	}
	return nil
}

// Remove removes the key/value pair from the pool.
func (p *Pool) Remove(key interface{}) {
	if p.Size() == 0 {
		return
	}
	if _, ok := p.pool.Load(key); ok {
		p.pool.Delete(key)
	}
}

// Size returns the number of key/value pairs in the pool.
func (p *Pool) Size() int {
	var size int
	p.pool.Range(func(_, _ interface{}) bool {
		size++
		return true
	})

	return size
}

// Clear removes all key/value pairs from the pool.
func (p *Pool) Clear() {
	p.pool = sync.Map{}
}

// Cap returns the capacity of the pool.
func (p *Pool) Cap() int {
	return p.cap
}

// NewPool creates a new pool with the given capacity.
//
//nolint:predeclared
func NewPool(cap int) *Pool {
	return &Pool{pool: sync.Map{}, cap: cap}
}
