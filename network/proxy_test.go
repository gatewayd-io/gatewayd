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

	proxy := NewProxy(1, false, false)
	assert.NotNil(t, proxy)
	assert.Equal(t, 0, proxy.Size(), "Proxy should have no connected clients")
	assert.Equal(t, 1, len(proxy.pool.ClientIDs()))
	assert.NotEqual(t, "", proxy.pool.ClientIDs()[0])
	assert.Equal(t, 1, proxy.PoolSize)
	assert.Equal(t, false, proxy.Elastic)
	assert.Equal(t, false, proxy.ReuseElasticClients)

	proxy.pool.Close()
}

func TestNewProxyElastic(t *testing.T) {
	proxy := NewProxy(1, true, false)
	assert.NotNil(t, proxy)
	assert.Equal(t, 0, proxy.Size())
	assert.Equal(t, 0, len(proxy.pool.ClientIDs()))
	assert.Equal(t, 1, proxy.PoolSize)
	assert.Equal(t, true, proxy.Elastic)
	assert.Equal(t, false, proxy.ReuseElasticClients)

	proxy.pool.Close()
}

func TestNewProxyElasticReuse(t *testing.T) {
	proxy := NewProxy(1, true, true)
	assert.NotNil(t, proxy)
	assert.Equal(t, 0, proxy.Size())
	assert.Equal(t, 0, len(proxy.pool.ClientIDs()))
	assert.Equal(t, 1, proxy.PoolSize)
	assert.Equal(t, true, proxy.Elastic)
	assert.Equal(t, true, proxy.ReuseElasticClients)

	proxy.pool.Close()
}
