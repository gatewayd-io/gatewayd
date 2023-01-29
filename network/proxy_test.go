package network

import (
	"testing"

	embeddedpostgres "github.com/fergusstrange/embedded-postgres"
	"github.com/stretchr/testify/assert"
)

func TestNewProxy(t *testing.T) {
	postgres := embeddedpostgres.NewDatabase()
	if err := postgres.Start(); err != nil {
		t.Fatal(err)
	}

	defer func() {
		if err := postgres.Stop(); err != nil {
			t.Fatal(err)
		}
	}()

	// Create a connection pool
	pool := NewPool()
	assert.NoError(t, pool.Put(NewClient("tcp", "localhost:5432", DefaultBufferSize)))

	// Create a proxy with a fixed buffer pool
	proxy := NewProxy(pool, false, false, nil)

	assert.NotNil(t, proxy)
	assert.Equal(t, 0, proxy.Size(), "Proxy should have no connected clients")
	assert.Equal(t, 1, len(proxy.pool.ClientIDs()))
	assert.NotEqual(t, "", proxy.pool.ClientIDs()[0])
	assert.Equal(t, false, proxy.Elastic)
	assert.Equal(t, false, proxy.ReuseElasticClients)

	proxy.pool.Close()
}

func TestNewProxyElastic(t *testing.T) {
	// Create a connection pool
	pool := NewPool()

	// Create a proxy with an elastic buffer pool
	proxy := NewProxy(pool, true, false, &Client{
		Network:           "tcp",
		Address:           "localhost:5432",
		ReceiveBufferSize: DefaultBufferSize,
	})

	assert.NotNil(t, proxy)
	assert.Equal(t, 0, proxy.Size())
	assert.Equal(t, 0, len(proxy.pool.ClientIDs()))
	assert.Equal(t, true, proxy.Elastic)
	assert.Equal(t, false, proxy.ReuseElasticClients)
	assert.Equal(t, "tcp", proxy.ClientConfig.Network)
	assert.Equal(t, "localhost:5432", proxy.ClientConfig.Address)
	assert.Equal(t, DefaultBufferSize, proxy.ClientConfig.ReceiveBufferSize)

	proxy.pool.Close()
}

func TestNewProxyElasticReuse(t *testing.T) {
	// Create a connection pool
	pool := NewPool()

	// Create a proxy with an elastic buffer pool
	proxy := NewProxy(pool, true, true, &Client{
		Network:           "tcp",
		Address:           "localhost:5432",
		ReceiveBufferSize: DefaultBufferSize,
	})

	assert.NotNil(t, proxy)
	assert.Equal(t, 0, proxy.Size())
	assert.Equal(t, 0, len(proxy.pool.ClientIDs()))
	assert.Equal(t, true, proxy.Elastic)
	assert.Equal(t, true, proxy.ReuseElasticClients)
	assert.Equal(t, "tcp", proxy.ClientConfig.Network)
	assert.Equal(t, "localhost:5432", proxy.ClientConfig.Address)
	assert.Equal(t, DefaultBufferSize, proxy.ClientConfig.ReceiveBufferSize)

	proxy.pool.Close()
}
