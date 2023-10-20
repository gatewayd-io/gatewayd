package pool

import (
	"context"
	"sync"

	"github.com/gatewayd-io/gatewayd/config"
	gerr "github.com/gatewayd-io/gatewayd/errors"
	"go.opentelemetry.io/otel"
)

type Callback func(key, value interface{}) bool

type IPool interface {
	ForEach(cb Callback)
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
	ctx  context.Context //nolint:containedctx
}

var _ IPool = &Pool{}

// ForEach iterates over the pool and calls the callback function for each key/value pair.
func (p *Pool) ForEach(cb Callback) {
	_, span := otel.Tracer(config.TracerName).Start(p.ctx, "ForEach")
	p.pool.Range(func(key, value any) bool {
		span.AddEvent("Callback")
		return cb(key, value)
	})
	span.End()
}

// Pool returns the underlying sync.Map.
func (p *Pool) Pool() *sync.Map {
	_, span := otel.Tracer(config.TracerName).Start(p.ctx, "Pool")
	defer span.End()
	return &p.pool
}

// Put adds a new key/value pair to the pool.
func (p *Pool) Put(key, value interface{}) *gerr.GatewayDError {
	_, span := otel.Tracer(config.TracerName).Start(p.ctx, "Put")
	defer span.End()
	if p.cap > 0 && p.Size() >= p.cap {
		span.RecordError(gerr.ErrPoolExhausted)
		return gerr.ErrPoolExhausted
	}

	if value == nil {
		span.RecordError(gerr.ErrNilPointer)
		return gerr.ErrNilPointer
	}

	p.pool.Store(key, value)
	return nil
}

// Get returns the value for the given key.
func (p *Pool) Get(key interface{}) interface{} {
	_, span := otel.Tracer(config.TracerName).Start(p.ctx, "Get")
	defer span.End()
	if value, ok := p.pool.Load(key); ok {
		return value
	}
	return nil
}

// GetOrPut returns the value for the given key if it exists, otherwise it adds
// the key/value pair to the pool.
func (p *Pool) GetOrPut(key, value interface{}) (interface{}, bool, *gerr.GatewayDError) {
	_, span := otel.Tracer(config.TracerName).Start(p.ctx, "GetOrPut")
	defer span.End()
	if p.cap > 0 && p.Size() >= p.cap {
		span.RecordError(gerr.ErrPoolExhausted)
		return nil, false, gerr.ErrPoolExhausted
	}

	if value == nil {
		span.RecordError(gerr.ErrNilPointer)
		return nil, false, gerr.ErrNilPointer
	}

	val, loaded := p.pool.LoadOrStore(key, value)
	return val, loaded, nil
}

// Pop removes the key/value pair from the pool and returns the value.
func (p *Pool) Pop(key interface{}) interface{} {
	_, span := otel.Tracer(config.TracerName).Start(p.ctx, "Pop")
	defer span.End()
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
	_, span := otel.Tracer(config.TracerName).Start(p.ctx, "Remove")
	defer span.End()
	if p.Size() == 0 {
		return
	}
	if _, ok := p.pool.Load(key); ok {
		p.pool.Delete(key)
	}
}

// Size returns the number of key/value pairs in the pool.
func (p *Pool) Size() int {
	_, span := otel.Tracer(config.TracerName).Start(p.ctx, "Size")
	defer span.End()
	var size int
	p.pool.Range(func(_, _ interface{}) bool {
		size++
		return true
	})

	return size
}

// Clear removes all key/value pairs from the pool.
func (p *Pool) Clear() {
	_, span := otel.Tracer(config.TracerName).Start(p.ctx, "Clear")
	defer span.End()
	p.pool = sync.Map{}
}

// Cap returns the capacity of the pool.
func (p *Pool) Cap() int {
	_, span := otel.Tracer(config.TracerName).Start(p.ctx, "Cap")
	defer span.End()
	return p.cap
}

// NewPool creates a new pool with the given capacity.
//
//nolint:predeclared
func NewPool(ctx context.Context, cap int) *Pool {
	poolCtx, span := otel.Tracer(config.TracerName).Start(ctx, "NewPool")
	defer span.End()

	return &Pool{
		pool: sync.Map{},
		cap:  cap,
		ctx:  poolCtx,
	}
}
