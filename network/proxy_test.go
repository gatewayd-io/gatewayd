package network

import (
	"testing"
	"time"

	embeddedpostgres "github.com/fergusstrange/embedded-postgres"
	"github.com/gatewayd-io/gatewayd/config"
	"github.com/gatewayd-io/gatewayd/logging"
	"github.com/gatewayd-io/gatewayd/plugin/hook"
	"github.com/gatewayd-io/gatewayd/pool"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestNewProxy tests the creation of a new proxy with a fixed connection pool.
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

	cfg := logging.LoggerConfig{
		Output:     config.Console,
		TimeFormat: zerolog.TimeFormatUnix,
		Level:      zerolog.DebugLevel,
		NoColor:    true,
	}

	logger := logging.NewLogger(cfg)

	// Create a connection pool
	pool := pool.NewPool(config.EmptyPoolCapacity)

	keepAlive, err := time.ParseDuration(config.DefaultTCPKeepAlivePeriod)
	require.NoError(t, err)

	client := NewClient(
		&config.Client{
			Network:            "tcp",
			Address:            "localhost:5432",
			ReceiveBufferSize:  config.DefaultBufferSize,
			ReceiveChunkSize:   config.DefaultChunkSize,
			ReceiveDeadline:    config.DefaultReceiveDeadline,
			SendDeadline:       config.DefaultSendDeadline,
			TCPKeepAlive:       false,
			TCPKeepAlivePeriod: keepAlive,
		},
		logger)
	err = pool.Put(client.ID, client)
	assert.Nil(t, err)

	healthCheck, err := time.ParseDuration(config.DefaultHealthCheckPeriod)
	require.NoError(t, err)

	// Create a proxy with a fixed buffer pool
	proxy := NewProxy(pool, hook.NewHookConfig(), false, false, healthCheck, nil, logger)

	assert.NotNil(t, proxy)
	assert.Equal(t, 0, proxy.busyConnections.Size(), "Proxy should have no connected clients")
	assert.Equal(t, 1, proxy.availableConnections.Size())
	if c, ok := proxy.availableConnections.Pop(client.ID).(*Client); ok {
		assert.NotEqual(t, "", c.ID)
	}
	assert.Equal(t, false, proxy.Elastic)
	assert.Equal(t, false, proxy.ReuseElasticClients)

	proxy.availableConnections.Clear()
}

// TestNewProxyElastic tests the creation of a new proxy with an elastic connection pool.
func TestNewProxyElastic(t *testing.T) {
	cfg := logging.LoggerConfig{
		Output:     config.Console,
		TimeFormat: zerolog.TimeFormatUnix,
		Level:      zerolog.DebugLevel,
		NoColor:    true,
	}

	logger := logging.NewLogger(cfg)

	// Create a connection pool
	pool := pool.NewPool(config.EmptyPoolCapacity)

	healthCheck, err := time.ParseDuration(config.DefaultHealthCheckPeriod)
	require.NoError(t, err)

	keepAlive, err := time.ParseDuration(config.DefaultTCPKeepAlivePeriod)
	require.NoError(t, err)

	// Create a proxy with an elastic buffer pool
	proxy := NewProxy(pool, hook.NewHookConfig(), true, false, healthCheck,
		&config.Client{
			Network:            "tcp",
			Address:            "localhost:5432",
			ReceiveBufferSize:  config.DefaultBufferSize,
			ReceiveChunkSize:   config.DefaultChunkSize,
			ReceiveDeadline:    config.DefaultReceiveDeadline,
			SendDeadline:       config.DefaultSendDeadline,
			TCPKeepAlive:       false,
			TCPKeepAlivePeriod: keepAlive,
		}, logger)

	assert.NotNil(t, proxy)
	assert.Equal(t, 0, proxy.busyConnections.Size())
	assert.Equal(t, 0, proxy.availableConnections.Size())
	assert.Equal(t, true, proxy.Elastic)
	assert.Equal(t, false, proxy.ReuseElasticClients)
	assert.Equal(t, "tcp", proxy.ClientConfig.Network)
	assert.Equal(t, "localhost:5432", proxy.ClientConfig.Address)
	assert.Equal(t, config.DefaultBufferSize, proxy.ClientConfig.ReceiveBufferSize)

	proxy.availableConnections.Clear()
}
