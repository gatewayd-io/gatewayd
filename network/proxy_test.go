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

	pr := NewProxy(1, false, false)
	assert.NotNil(t, pr)
	assert.Equal(t, 0, pr.Size(), "Proxy should have no connected clients")
	assert.Equal(t, 1, len(pr.pool.ClientIDs()))
	assert.NotEqual(t, "", pr.pool.ClientIDs()[0])
	assert.Equal(t, 1, pr.PoolSize)
	assert.Equal(t, false, pr.Elastic)
	assert.Equal(t, false, pr.ReuseElasticClients)

	pr.pool.Close()
}

func TestNewProxyElastic(t *testing.T) {
	pr := NewProxy(1, true, false)
	assert.NotNil(t, pr)
	assert.Equal(t, 0, pr.Size())
	assert.Equal(t, 0, len(pr.pool.ClientIDs()))
	assert.Equal(t, 1, pr.PoolSize)
	assert.Equal(t, true, pr.Elastic)
	assert.Equal(t, false, pr.ReuseElasticClients)

	pr.pool.Close()
}

func TestNewProxyElasticReuse(t *testing.T) {
	pr := NewProxy(1, true, true)
	assert.NotNil(t, pr)
	assert.Equal(t, 0, pr.Size())
	assert.Equal(t, 0, len(pr.pool.ClientIDs()))
	assert.Equal(t, 1, pr.PoolSize)
	assert.Equal(t, true, pr.Elastic)
	assert.Equal(t, true, pr.ReuseElasticClients)

	pr.pool.Close()
}
