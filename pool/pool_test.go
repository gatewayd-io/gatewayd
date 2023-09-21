package pool

import (
	"context"
	"testing"

	"github.com/gatewayd-io/gatewayd/config"
	"github.com/stretchr/testify/assert"
)

// TestNewPool tests the NewPool function.
func TestNewPool(t *testing.T) {
	pool := NewPool(context.Background(), config.EmptyPoolCapacity)
	defer pool.Clear()
	assert.NotNil(t, pool)
	assert.NotNil(t, pool.Pool())
	assert.Equal(t, 0, pool.Size())
}

// TestPool_Put tests the Put function.
func TestPool_Put(t *testing.T) {
	pool := NewPool(context.Background(), config.EmptyPoolCapacity)
	defer pool.Clear()
	assert.NotNil(t, pool)
	assert.NotNil(t, pool.Pool())
	assert.Equal(t, 0, pool.Size())
	err := pool.Put("client1.ID", "client1")
	assert.Nil(t, err)
	assert.Equal(t, 1, pool.Size())
	err = pool.Put("client2.ID", "client2")
	assert.Equal(t, 2, pool.Size())
	assert.Nil(t, err)
}

// TestPool_Pop tests the Pop function.
//
//nolint:dupl
func TestPool_Pop(t *testing.T) {
	pool := NewPool(context.Background(), config.EmptyPoolCapacity)
	defer pool.Clear()
	assert.NotNil(t, pool)
	assert.NotNil(t, pool.Pool())
	assert.Equal(t, 0, pool.Size())
	err := pool.Put("client1.ID", "client1")
	assert.Nil(t, err)
	assert.Equal(t, 1, pool.Size())
	err = pool.Put("client2.ID", "client2")
	assert.Nil(t, err)
	assert.Equal(t, 2, pool.Size())
	if c1, ok := pool.Pop("client1.ID").(string); !ok {
		assert.Equal(t, c1, "client1")
	} else {
		assert.Equal(t, "client1", c1)
		assert.Equal(t, 1, pool.Size())
	}
	if c2, ok := pool.Pop("client2.ID").(string); !ok {
		assert.Equal(t, c2, "client2")
	} else {
		assert.Equal(t, "client2", c2)
		assert.Equal(t, 0, pool.Size())
	}
}

// TestPool_Clear tests the Clear function.
func TestPool_Clear(t *testing.T) {
	pool := NewPool(context.Background(), config.EmptyPoolCapacity)
	defer pool.Clear()
	assert.NotNil(t, pool)
	assert.NotNil(t, pool.Pool())
	assert.Equal(t, 0, pool.Size())
	err := pool.Put("client1.ID", "client1")
	assert.Nil(t, err)
	assert.Equal(t, 1, pool.Size())
	err = pool.Put("client2.ID", "client2")
	assert.Nil(t, err)
	assert.Equal(t, 2, pool.Size())
	pool.Clear()
	assert.Equal(t, 0, pool.Size())
}

// TestPool_ForEach tests the ForEach function.
func TestPool_ForEach(t *testing.T) {
	pool := NewPool(context.Background(), config.EmptyPoolCapacity)
	defer pool.Clear()
	assert.NotNil(t, pool)
	assert.NotNil(t, pool.Pool())
	assert.Equal(t, 0, pool.Size())
	err := pool.Put("client1.ID", "client1")
	assert.Nil(t, err)
	assert.Equal(t, 1, pool.Size())
	err = pool.Put("client2.ID", "client2")
	assert.Nil(t, err)
	assert.Equal(t, 2, pool.Size())
	pool.ForEach(func(key, value interface{}) bool {
		if c, ok := value.(string); ok {
			assert.NotEmpty(t, c)
		}
		return true
	})
}

// TestPool_Get tests the Get function.
//
//nolint:dupl
func TestPool_Get(t *testing.T) {
	pool := NewPool(context.Background(), config.EmptyPoolCapacity)
	defer pool.Clear()
	assert.NotNil(t, pool)
	assert.NotNil(t, pool.Pool())
	assert.Equal(t, 0, pool.Size())
	err := pool.Put("client1.ID", "client1")
	assert.Nil(t, err)
	assert.Equal(t, 1, pool.Size())
	err = pool.Put("client2.ID", "client2")
	assert.Nil(t, err)
	assert.Equal(t, 2, pool.Size())
	if c1, ok := pool.Get("client1.ID").(string); !ok {
		assert.Equal(t, c1, "client1")
	} else {
		assert.Equal(t, "client1", c1)
		assert.Equal(t, 2, pool.Size())
	}
	if c2, ok := pool.Get("client2.ID").(string); !ok {
		assert.Equal(t, c2, "client2")
	} else {
		assert.Equal(t, "client2", c2)
		assert.Equal(t, 2, pool.Size())
	}
}

// TestPool_GetOrPut tests the GetOrPut function.
func TestPool_GetOrPut(t *testing.T) {
	pool := NewPool(context.Background(), config.EmptyPoolCapacity)
	defer pool.Clear()
	assert.NotNil(t, pool)
	assert.NotNil(t, pool.Pool())
	assert.Equal(t, 0, pool.Size())
	err := pool.Put("client1.ID", "client1")
	assert.Nil(t, err)
	assert.Equal(t, 1, pool.Size())
	err = pool.Put("client2.ID", "client2")
	assert.Nil(t, err)
	assert.Equal(t, 2, pool.Size())
	c1, loaded, err := pool.GetOrPut("client1.ID", "client1")
	assert.True(t, loaded)
	if c1, ok := c1.(string); !ok {
		assert.Equal(t, c1, "client1")
	} else {
		assert.Equal(t, "client1", c1)
		assert.Equal(t, 2, pool.Size())
	}
	assert.Nil(t, err)
	c2, loaded, err := pool.GetOrPut("client2.ID", "client2")
	assert.True(t, loaded)
	if c2, ok := c2.(string); !ok {
		assert.Equal(t, c2, "client2")
	} else {
		assert.Equal(t, "client2", c2)
		assert.Equal(t, 2, pool.Size())
	}
	assert.Nil(t, err)
}

// TestPool_Remove tests the Remove function.
func TestPool_Remove(t *testing.T) {
	pool := NewPool(context.Background(), config.EmptyPoolCapacity)
	defer pool.Clear()
	assert.NotNil(t, pool)
	assert.NotNil(t, pool.Pool())
	assert.Equal(t, 0, pool.Size())
	err := pool.Put("client1.ID", "client1")
	assert.Nil(t, err)
	assert.Equal(t, 1, pool.Size())
	err = pool.Put("client2.ID", "client2")
	assert.Nil(t, err)
	assert.Equal(t, 2, pool.Size())
	pool.Remove("client1.ID")
	assert.Equal(t, 1, pool.Size())
	pool.Remove("client2.ID")
	assert.Equal(t, 0, pool.Size())
}

// TestPool_GetClientIDs tests the GetClientIDs function.
func TestPool_GetClientIDs(t *testing.T) {
	pool := NewPool(context.Background(), config.EmptyPoolCapacity)
	defer pool.Clear()
	assert.NotNil(t, pool)
	assert.NotNil(t, pool.Pool())
	assert.Equal(t, 0, pool.Size())
	err := pool.Put("client1.ID", "client1")
	assert.Nil(t, err)
	assert.Equal(t, 1, pool.Size())
	err = pool.Put("client2.ID", "client2")
	assert.Nil(t, err)
	assert.Equal(t, 2, pool.Size())

	var ids []string
	pool.ForEach(func(key, value interface{}) bool {
		if id, ok := key.(string); ok {
			ids = append(ids, id)
		}
		return true
	})
	assert.Equal(t, 2, len(ids))
	assert.Contains(t, ids, "client1.ID")
	assert.Contains(t, ids, "client2.ID")
	pool.Clear()
}

func TestPool_Cap(t *testing.T) {
	pool := NewPool(context.Background(), 1)
	assert.NotNil(t, pool)
	assert.NotNil(t, pool.Pool())
	assert.Equal(t, 0, pool.Size())
	assert.Equal(t, 1, pool.Cap())
	err := pool.Put("client1.ID", "client1")
	assert.Nil(t, err)
	assert.Equal(t, 1, pool.Size())
	err = pool.Put("client2.ID", "client2")
	assert.NotNil(t, err)
	assert.Equal(t, 1, pool.Size())
	assert.Equal(t, 1, pool.Cap())
	pool.Clear()
	assert.Equal(t, 0, pool.Size())
	assert.Equal(t, 1, pool.Cap())
}

func BenchmarkNewPool(b *testing.B) {
	for i := 0; i < b.N; i++ {
		NewPool(context.Background(), config.EmptyPoolCapacity)
	}
}

func BenchmarkPool_PutPop(b *testing.B) {
	pool := NewPool(context.Background(), config.EmptyPoolCapacity)
	defer pool.Clear()
	for i := 0; i < b.N; i++ {
		pool.Put("client1.ID", "client1") //nolint:errcheck
		pool.Pop("client1.ID")
	}
}

func BenchmarkPool_Clear(b *testing.B) {
	pool := NewPool(context.Background(), config.EmptyPoolCapacity)
	for i := 0; i < b.N; i++ {
		pool.Clear()
	}
}

func BenchmarkPool_ForEach(b *testing.B) {
	pool := NewPool(context.Background(), config.EmptyPoolCapacity)
	defer pool.Clear()
	pool.Put("client1.ID", "client1") //nolint:errcheck
	pool.Put("client2.ID", "client2") //nolint:errcheck
	for i := 0; i < b.N; i++ {
		pool.ForEach(func(key, value interface{}) bool {
			return true
		})
	}
}

func BenchmarkPool_Get(b *testing.B) {
	pool := NewPool(context.Background(), config.EmptyPoolCapacity)
	defer pool.Clear()
	pool.Put("client1.ID", "client1") //nolint:errcheck
	pool.Put("client2.ID", "client2") //nolint:errcheck
	for i := 0; i < b.N; i++ {
		pool.Get("client1.ID")
		pool.Get("client2.ID")
	}
}

func BenchmarkPool_GetOrPut(b *testing.B) {
	pool := NewPool(context.Background(), config.EmptyPoolCapacity)
	defer pool.Clear()
	pool.Put("client1.ID", "client1") //nolint:errcheck
	pool.Put("client2.ID", "client2") //nolint:errcheck
	for i := 0; i < b.N; i++ {
		pool.GetOrPut("client1.ID", "client1") //nolint:errcheck
		pool.GetOrPut("client2.ID", "client2") //nolint:errcheck
	}
}

func BenchmarkPool_Remove(b *testing.B) {
	pool := NewPool(context.Background(), config.EmptyPoolCapacity)
	defer pool.Clear()
	for i := 0; i < b.N; i++ {
		pool.Put("client1.ID", "client1") //nolint:errcheck
		pool.Remove("client1.ID")
	}
}

func BenchmarkPool_Size(b *testing.B) {
	pool := NewPool(context.Background(), config.EmptyPoolCapacity)
	defer pool.Clear()
	for i := 0; i < b.N; i++ {
		pool.Size()
	}
}

func BenchmarkPool_Cap(b *testing.B) {
	pool := NewPool(context.Background(), config.EmptyPoolCapacity)
	defer pool.Clear()
	for i := 0; i < b.N; i++ {
		pool.Cap()
	}
}
