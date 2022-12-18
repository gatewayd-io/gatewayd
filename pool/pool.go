package pool

import (
	"sync"

	gerr "github.com/gatewayd-io/gatewayd/errors"
)

const (
	EmptyPoolCapacity = 0
)

type Callback func(key, value interface{}) bool

type Pool interface {
	ForEach(Callback)
	Pool() *sync.Map
	Put(key, value interface{}) error
	Get(key interface{}) interface{}
	GetOrPut(key, value interface{}) (interface{}, bool, error)
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

func (p *Impl) ForEach(cb Callback) {
	p.pool.Range(cb)
}

func (p *Impl) Pool() *sync.Map {
	return &p.pool
}

func (p *Impl) Put(key, value interface{}) error {
	if p.cap > 0 && p.Size() >= p.cap {
		return gerr.ErrPoolExhausted
	}
	p.pool.Store(key, value)
	return nil
}

func (p *Impl) Get(key interface{}) interface{} {
	if value, ok := p.pool.Load(key); ok {
		return value
	}
	return nil
}

func (p *Impl) GetOrPut(key, value interface{}) (interface{}, bool, error) {
	if p.cap > 0 && p.Size() >= p.cap {
		return nil, false, gerr.ErrPoolExhausted
	}
	val, loaded := p.pool.LoadOrStore(key, value)
	return val, loaded, nil
}

func (p *Impl) Pop(key interface{}) interface{} {
	if p.Size() == 0 {
		return nil
	}
	if value, ok := p.pool.LoadAndDelete(key); ok {
		return value
	}
	return nil
}

func (p *Impl) Remove(key interface{}) {
	if p.Size() == 0 {
		return
	}
	if _, ok := p.pool.Load(key); ok {
		p.pool.Delete(key)
	}
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

func (p *Impl) Cap() int {
	return p.cap
}

func NewPool(cap int) *Impl {
	return &Impl{pool: sync.Map{}, cap: cap}
}
