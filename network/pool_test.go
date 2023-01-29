package network

import (
	"testing"

	embeddedpostgres "github.com/fergusstrange/embedded-postgres"
	"github.com/stretchr/testify/assert"
)

func TestNewPool(t *testing.T) {
	p := NewPool()
	defer p.Close()
	assert.NotNil(t, p)
	assert.NotNil(t, p.Pool())
	assert.Equal(t, 0, p.Size())
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

	p := NewPool()
	defer p.Close()
	assert.NotNil(t, p)
	assert.NotNil(t, p.Pool())
	assert.Equal(t, 0, p.Size())
	p.Put(NewClient("tcp", "localhost:5432", 4096))
	assert.Equal(t, 1, p.Size())
	p.Put(NewClient("tcp", "localhost:5432", 4096))
	assert.Equal(t, 2, p.Size())
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

	p := NewPool()
	defer p.Close()
	assert.NotNil(t, p)
	assert.NotNil(t, p.Pool())
	assert.Equal(t, 0, p.Size())
	c1 := NewClient("tcp", "localhost:5432", 4096)
	p.Put(c1)
	assert.Equal(t, 1, p.Size())
	c2 := NewClient("tcp", "localhost:5432", 4096)
	p.Put(c2)
	assert.Equal(t, 2, p.Size())
	client := p.Pop(c1.ID)
	assert.Equal(t, c1.ID, client.ID)
	assert.Equal(t, 1, p.Size())
	client = p.Pop(c2.ID)
	assert.Equal(t, c2.ID, client.ID)
	assert.Equal(t, 0, p.Size())
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

	p := NewPool()
	assert.NotNil(t, p)
	assert.NotNil(t, p.Pool())
	assert.Equal(t, 0, p.Size())
	c1 := NewClient("tcp", "localhost:5432", 4096)
	p.Put(c1)
	assert.Equal(t, 1, p.Size())
	c2 := NewClient("tcp", "localhost:5432", 4096)
	p.Put(c2)
	assert.Equal(t, 2, p.Size())
	err := p.Close()
	assert.Nil(t, err)
	assert.Equal(t, 2, p.Size())
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

	p := NewPool()
	defer p.Close()
	assert.NotNil(t, p)
	assert.NotNil(t, p.Pool())
	assert.Equal(t, 0, p.Size())
	c1 := NewClient("tcp", "localhost:5432", 4096)
	p.Put(c1)
	assert.Equal(t, 1, p.Size())
	c2 := NewClient("tcp", "localhost:5432", 4096)
	p.Put(c2)
	assert.Equal(t, 2, p.Size())
	p.Shutdown()
	assert.Equal(t, 0, p.Size())
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

	p := NewPool()
	defer p.Close()
	assert.NotNil(t, p)
	assert.NotNil(t, p.Pool())
	assert.Equal(t, 0, p.Size())
	c1 := NewClient("tcp", "localhost:5432", 4096)
	p.Put(c1)
	assert.Equal(t, 1, p.Size())
	c2 := NewClient("tcp", "localhost:5432", 4096)
	p.Put(c2)
	assert.Equal(t, 2, p.Size())
	p.ForEach(func(client *Client) error {
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

	p := NewPool()
	defer p.Close()
	assert.NotNil(t, p)
	assert.NotNil(t, p.Pool())
	assert.Equal(t, 0, p.Size())
	c1 := NewClient("tcp", "localhost:5432", 4096)
	p.Put(c1)
	assert.Equal(t, 1, p.Size())
	c2 := NewClient("tcp", "localhost:5432", 4096)
	p.Put(c2)
	assert.Equal(t, 2, p.Size())
	ids := p.ClientIDs()
	assert.Equal(t, 2, len(ids))
}
