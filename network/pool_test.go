package network

import (
	"testing"

	embeddedpostgres "github.com/fergusstrange/embedded-postgres"
	"github.com/stretchr/testify/assert"
)

func TestNewPool(t *testing.T) {
	pool := NewPool()
	defer pool.Close()
	assert.NotNil(t, pool)
	assert.NotNil(t, pool.Pool())
	assert.Equal(t, 0, pool.Size())
}

func TestPool_Put(t *testing.T) {
	postgres := embeddedpostgres.NewDatabase()
	if err := postgres.Start(); err != nil {
		t.Fatal(err)
	}

	defer func() {
		if err := postgres.Stop(); err != nil {
			t.Fatal(err)
		}
	}()

	pool := NewPool()
	defer pool.Close()
	assert.NotNil(t, pool)
	assert.NotNil(t, pool.Pool())
	assert.Equal(t, 0, pool.Size())
	assert.NoError(t, pool.Put(NewClient("tcp", "localhost:5432", DefaultBufferSize)))
	assert.Equal(t, 1, pool.Size())
	assert.NoError(t, pool.Put(NewClient("tcp", "localhost:5432", DefaultBufferSize)))
	assert.Equal(t, 2, pool.Size())
}

func TestPool_Pop(t *testing.T) {
	postgres := embeddedpostgres.NewDatabase()
	if err := postgres.Start(); err != nil {
		t.Fatal(err)
	}

	defer func() {
		if err := postgres.Stop(); err != nil {
			t.Fatal(err)
		}
	}()

	pool := NewPool()
	defer pool.Close()
	assert.NotNil(t, pool)
	assert.NotNil(t, pool.Pool())
	assert.Equal(t, 0, pool.Size())
	client1 := NewClient("tcp", "localhost:5432", DefaultBufferSize)
	assert.NoError(t, pool.Put(client1))
	assert.Equal(t, 1, pool.Size())
	client2 := NewClient("tcp", "localhost:5432", DefaultBufferSize)
	assert.NoError(t, pool.Put(client2))
	assert.Equal(t, 2, pool.Size())
	client := pool.Pop(client1.ID)
	assert.Equal(t, client1.ID, client.ID)
	assert.Equal(t, 1, pool.Size())
	client = pool.Pop(client2.ID)
	assert.Equal(t, client2.ID, client.ID)
	assert.Equal(t, 0, pool.Size())
}

func TestPool_Close(t *testing.T) {
	postgres := embeddedpostgres.NewDatabase()
	if err := postgres.Start(); err != nil {
		t.Fatal(err)
	}

	defer func() {
		if err := postgres.Stop(); err != nil {
			t.Fatal(err)
		}
	}()

	pool := NewPool()
	assert.NotNil(t, pool)
	assert.NotNil(t, pool.Pool())
	assert.Equal(t, 0, pool.Size())
	client1 := NewClient("tcp", "localhost:5432", DefaultBufferSize)
	assert.NoError(t, pool.Put(client1))
	assert.Equal(t, 1, pool.Size())
	client2 := NewClient("tcp", "localhost:5432", DefaultBufferSize)
	assert.NoError(t, pool.Put(client2))
	assert.Equal(t, 2, pool.Size())
	err := pool.Close()
	assert.Nil(t, err)
	assert.Equal(t, 2, pool.Size())
}

func TestPool_Shutdown(t *testing.T) {
	postgres := embeddedpostgres.NewDatabase()
	if err := postgres.Start(); err != nil {
		t.Fatal(err)
	}

	defer func() {
		if err := postgres.Stop(); err != nil {
			t.Fatal(err)
		}
	}()

	pool := NewPool()
	defer pool.Close()
	assert.NotNil(t, pool)
	assert.NotNil(t, pool.Pool())
	assert.Equal(t, 0, pool.Size())
	client1 := NewClient("tcp", "localhost:5432", DefaultBufferSize)
	assert.NoError(t, pool.Put(client1))
	assert.Equal(t, 1, pool.Size())
	client2 := NewClient("tcp", "localhost:5432", DefaultBufferSize)
	assert.NoError(t, pool.Put(client2))
	assert.Equal(t, 2, pool.Size())
	pool.Shutdown()
	assert.Equal(t, 0, pool.Size())
}

func TestPool_ForEach(t *testing.T) {
	postgres := embeddedpostgres.NewDatabase()
	if err := postgres.Start(); err != nil {
		t.Fatal(err)
	}

	defer func() {
		if err := postgres.Stop(); err != nil {
			t.Fatal(err)
		}
	}()

	pool := NewPool()
	defer pool.Close()
	assert.NotNil(t, pool)
	assert.NotNil(t, pool.Pool())
	assert.Equal(t, 0, pool.Size())
	client1 := NewClient("tcp", "localhost:5432", DefaultBufferSize)
	assert.NoError(t, pool.Put(client1))
	assert.Equal(t, 1, pool.Size())
	client2 := NewClient("tcp", "localhost:5432", DefaultBufferSize)
	assert.NoError(t, pool.Put(client2))
	assert.Equal(t, 2, pool.Size())
	pool.ForEach(func(client *Client) error {
		assert.NotNil(t, client)
		return nil
	})
}

func TestPool_ClientIDs(t *testing.T) {
	postgres := embeddedpostgres.NewDatabase()
	if err := postgres.Start(); err != nil {
		t.Fatal(err)
	}

	defer func() {
		if err := postgres.Stop(); err != nil {
			t.Fatal(err)
		}
	}()

	pool := NewPool()
	defer pool.Close()
	assert.NotNil(t, pool)
	assert.NotNil(t, pool.Pool())
	assert.Equal(t, 0, pool.Size())
	client1 := NewClient("tcp", "localhost:5432", DefaultBufferSize)
	assert.NoError(t, pool.Put(client1))
	assert.Equal(t, 1, pool.Size())
	client2 := NewClient("tcp", "localhost:5432", DefaultBufferSize)
	assert.NoError(t, pool.Put(client2))
	assert.Equal(t, 2, pool.Size())
	ids := pool.ClientIDs()
	assert.Equal(t, 2, len(ids))
}
