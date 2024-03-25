package network

import (
	"context"
	"testing"

	"github.com/gatewayd-io/gatewayd/config"
	"github.com/gatewayd-io/gatewayd/logging"
	"github.com/gatewayd-io/gatewayd/plugin"
	"github.com/gatewayd-io/gatewayd/pool"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
)

// TestNewProxy tests the creation of a new proxy with a fixed connection pool.
func TestNewProxy(t *testing.T) {
	logger := logging.NewLogger(context.Background(), logging.LoggerConfig{
		Output:            []config.LogOutput{config.Console},
		TimeFormat:        zerolog.TimeFormatUnix,
		ConsoleTimeFormat: config.DefaultConsoleTimeFormat,
		Level:             zerolog.DebugLevel,
		NoColor:           true,
	})

	// Create a connection pool
	pool := pool.NewPool(context.Background(), config.EmptyPoolCapacity)

	client := NewClient(
		context.Background(),
		&Client{
			Network:            "tcp",
			Address:            "localhost:5432",
			ReceiveChunkSize:   config.DefaultChunkSize,
			ReceiveDeadline:    config.DefaultReceiveDeadline,
			SendDeadline:       config.DefaultSendDeadline,
			TCPKeepAlive:       false,
			TCPKeepAlivePeriod: config.DefaultTCPKeepAlivePeriod,
		})
	err := pool.Put(client.ID, client)
	assert.Nil(t, err)

	// Create a proxy with a fixed buffer pool
	pluginRegistry := plugin.NewRegistry(
		context.Background(),
		config.Loose,
		config.PassDown,
		config.Accept,
		config.Stop,
		logger,
		false,
	)
	proxy := NewProxy(
		context.Background(),
		Proxy{
			AvailableConnections: pool,
			PluginRegistry:       pluginRegistry,
			HealthCheckPeriod:    config.DefaultHealthCheckPeriod,
			Logger:               logger,
			PluginTimeout:        config.DefaultPluginTimeout,
		})
	defer proxy.Shutdown()

	assert.NotNil(t, proxy)
	assert.Equal(t, 0, proxy.busyConnections.Size(), "Proxy should have no connected clients")
	assert.Equal(t, 1, proxy.AvailableConnections.Size())
	if c, ok := proxy.AvailableConnections.Pop(client.ID).(*Client); ok {
		assert.NotEqual(t, "", c.ID)
	}
	assert.Equal(t, false, proxy.Elastic)
	assert.Equal(t, false, proxy.ReuseElasticClients)
	assert.Equal(t, false, proxy.IsExhausted())
	c, err := proxy.IsHealty(client)
	assert.Nil(t, err)
	assert.Equal(t, client, c)
}

// TestNewProxyElastic tests the creation of a new proxy with an elastic connection pool.
func TestNewProxyElastic(t *testing.T) {
	logger := logging.NewLogger(context.Background(), logging.LoggerConfig{
		Output:            []config.LogOutput{config.Console},
		TimeFormat:        zerolog.TimeFormatUnix,
		ConsoleTimeFormat: config.DefaultConsoleTimeFormat,
		Level:             zerolog.DebugLevel,
		NoColor:           true,
	})

	// Create a connection pool
	pool := pool.NewPool(context.Background(), config.EmptyPoolCapacity)

	// Create a proxy with an elastic buffer pool
	pluginRegistry := plugin.NewRegistry(
		context.Background(),
		config.Loose,
		config.PassDown,
		config.Accept,
		config.Stop,
		logger,
		false,
	)
	proxy := NewProxy(
		context.Background(),
		Proxy{
			AvailableConnections: pool,
			Elastic:              true,
			HealthCheckPeriod:    config.DefaultHealthCheckPeriod,
			Client: &Client{
				Network:            "tcp",
				Address:            "localhost:5432",
				ReceiveChunkSize:   config.DefaultChunkSize,
				ReceiveDeadline:    config.DefaultReceiveDeadline,
				SendDeadline:       config.DefaultSendDeadline,
				TCPKeepAlive:       false,
				TCPKeepAlivePeriod: config.DefaultTCPKeepAlivePeriod,
			},
			Logger:         logger,
			PluginTimeout:  config.DefaultPluginTimeout,
			PluginRegistry: pluginRegistry,
		})
	defer proxy.Shutdown()

	assert.NotNil(t, proxy)
	assert.Equal(t, 0, proxy.busyConnections.Size())
	assert.Equal(t, 0, proxy.AvailableConnections.Size())
	assert.Equal(t, true, proxy.Elastic)
	assert.Equal(t, false, proxy.ReuseElasticClients)
	assert.Equal(t, "tcp", proxy.Client.Network)
	assert.Equal(t, "localhost:5432", proxy.Client.Address)
}
