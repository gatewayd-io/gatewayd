package network

import (
	"context"
	"testing"
	"time"

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
		ConsoleTimeFormat: time.RFC3339,
		Level:             zerolog.WarnLevel,
		NoColor:           true,
	})

	// Create a connection newPool
	newPool := pool.NewPool(context.Background(), config.EmptyPoolCapacity)

	client := NewClient(
		context.Background(),
		&config.Client{
			Network:            "tcp",
			Address:            "localhost:5432",
			ReceiveChunkSize:   config.DefaultChunkSize,
			ReceiveDeadline:    config.DefaultReceiveDeadline,
			SendDeadline:       config.DefaultSendDeadline,
			TCPKeepAlive:       false,
			TCPKeepAlivePeriod: config.DefaultTCPKeepAlivePeriod,
		},
		logger)
	err := newPool.Put(client.ID, client)
	assert.Nil(t, err)

	// Create a proxy with a fixed buffer newPool
	proxy := NewProxy(
		context.Background(),
		newPool,
		plugin.NewRegistry(
			context.Background(),
			config.Loose,
			config.PassDown,
			config.Accept,
			config.Stop,
			logger,
			false,
		),
		false,
		false,
		config.DefaultHealthCheckPeriod,
		nil,
		logger,
		config.DefaultPluginTimeout)
	defer proxy.Shutdown()

	assert.NotNil(t, proxy)
	assert.Equal(t, 0, proxy.busyConnections.Size(), "Proxy should have no connected clients")
	assert.Equal(t, 1, proxy.availableConnections.Size())
	if c, ok := proxy.availableConnections.Pop(client.ID).(*Client); ok {
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
		ConsoleTimeFormat: time.RFC3339,
		Level:             zerolog.WarnLevel,
		NoColor:           true,
	})

	// Create a connection newPool
	newPool := pool.NewPool(context.Background(), config.EmptyPoolCapacity)

	// Create a proxy with an elastic buffer newPool
	proxy := NewProxy(
		context.Background(),
		newPool,
		plugin.NewRegistry(
			context.Background(),
			config.Loose,
			config.PassDown,
			config.Accept,
			config.Stop,
			logger,
			false,
		),
		true,
		false,
		config.DefaultHealthCheckPeriod,
		&config.Client{
			Network:            "tcp",
			Address:            "localhost:5432",
			ReceiveChunkSize:   config.DefaultChunkSize,
			ReceiveDeadline:    config.DefaultReceiveDeadline,
			SendDeadline:       config.DefaultSendDeadline,
			TCPKeepAlive:       false,
			TCPKeepAlivePeriod: config.DefaultTCPKeepAlivePeriod,
		},
		logger,
		config.DefaultPluginTimeout)
	defer proxy.Shutdown()

	assert.NotNil(t, proxy)
	assert.Equal(t, 0, proxy.busyConnections.Size())
	assert.Equal(t, 0, proxy.availableConnections.Size())
	assert.Equal(t, true, proxy.Elastic)
	assert.Equal(t, false, proxy.ReuseElasticClients)
	assert.Equal(t, "tcp", proxy.ClientConfig.Network)
	assert.Equal(t, "localhost:5432", proxy.ClientConfig.Address)
}

func BenchmarkNewProxy(b *testing.B) {
	logger := logging.NewLogger(context.Background(), logging.LoggerConfig{
		Output:            []config.LogOutput{config.Console},
		TimeFormat:        zerolog.TimeFormatUnix,
		ConsoleTimeFormat: time.RFC3339,
		Level:             zerolog.WarnLevel,
		NoColor:           true,
	})

	// Create a connection newPool
	newPool := pool.NewPool(context.Background(), config.EmptyPoolCapacity)

	// Create a proxy with a fixed buffer newPool
	for i := 0; i < b.N; i++ {
		proxy := NewProxy(
			context.Background(),
			newPool,
			plugin.NewRegistry(
				context.Background(),
				config.Loose,
				config.PassDown,
				config.Accept,
				config.Stop,
				logger,
				false,
			),
			false,
			false,
			config.DefaultHealthCheckPeriod,
			nil,
			logger,
			config.DefaultPluginTimeout)
		proxy.Shutdown()
	}
}

func BenchmarkNewProxyElastic(b *testing.B) {
	logger := logging.NewLogger(context.Background(), logging.LoggerConfig{
		Output:            []config.LogOutput{config.Console},
		TimeFormat:        zerolog.TimeFormatUnix,
		ConsoleTimeFormat: time.RFC3339,
		Level:             zerolog.WarnLevel,
		NoColor:           true,
	})

	// Create a connection newPool
	newPool := pool.NewPool(context.Background(), config.EmptyPoolCapacity)

	// Create a proxy with an elastic buffer newPool
	for i := 0; i < b.N; i++ {
		proxy := NewProxy(
			context.Background(),
			newPool,
			plugin.NewRegistry(
				context.Background(),
				config.Loose,
				config.PassDown,
				config.Accept,
				config.Stop,
				logger,
				false,
			),
			true,
			false,
			config.DefaultHealthCheckPeriod,
			&config.Client{
				Network:            "tcp",
				Address:            "localhost:5432",
				ReceiveChunkSize:   config.DefaultChunkSize,
				ReceiveDeadline:    config.DefaultReceiveDeadline,
				SendDeadline:       config.DefaultSendDeadline,
				TCPKeepAlive:       false,
				TCPKeepAlivePeriod: config.DefaultTCPKeepAlivePeriod,
			},
			logger,
			config.DefaultPluginTimeout)
		proxy.Shutdown()
	}
}

func BenchmarkProxyConnectDisconnect(b *testing.B) {
	logger := logging.NewLogger(context.Background(), logging.LoggerConfig{
		Output:            []config.LogOutput{config.Console},
		TimeFormat:        zerolog.TimeFormatUnix,
		ConsoleTimeFormat: time.RFC3339,
		Level:             zerolog.PanicLevel,
		NoColor:           true,
	})

	// Create a connection newPool
	newPool := pool.NewPool(context.Background(), 1)

	clientConfig := config.Client{
		Network:            "tcp",
		Address:            "localhost:5432",
		ReceiveChunkSize:   config.DefaultChunkSize,
		ReceiveDeadline:    config.DefaultReceiveDeadline,
		ReceiveTimeout:     config.DefaultReceiveTimeout,
		SendDeadline:       config.DefaultSendDeadline,
		TCPKeepAlive:       false,
		TCPKeepAlivePeriod: config.DefaultTCPKeepAlivePeriod,
	}
	newPool.Put("client", NewClient(context.Background(), &clientConfig, logger)) //nolint:errcheck

	// Create a proxy with a fixed buffer newPool
	proxy := NewProxy(
		context.Background(),
		newPool,
		plugin.NewRegistry(
			context.Background(),
			config.Loose,
			config.PassDown,
			config.Accept,
			config.Stop,
			logger,
			false,
		),
		false,
		false,
		config.DefaultHealthCheckPeriod,
		&clientConfig,
		logger,
		config.DefaultPluginTimeout)
	defer proxy.Shutdown()

	conn := testConnection{}

	// Connect to the proxy
	for i := 0; i < b.N; i++ {
		proxy.Connect(conn.Conn) //nolint:errcheck
		proxy.Disconnect(&conn)  //nolint:errcheck
	}
}

func BenchmarkProxyPassThrough(b *testing.B) {
	logger := logging.NewLogger(context.Background(), logging.LoggerConfig{
		Output:            []config.LogOutput{config.Console},
		TimeFormat:        zerolog.TimeFormatUnix,
		ConsoleTimeFormat: time.RFC3339,
		Level:             zerolog.PanicLevel,
		NoColor:           true,
	})

	// Create a connection newPool
	newPool := pool.NewPool(context.Background(), 1)

	clientConfig := config.Client{
		Network:            "tcp",
		Address:            "localhost:5432",
		ReceiveChunkSize:   config.DefaultChunkSize,
		ReceiveDeadline:    config.DefaultReceiveDeadline,
		ReceiveTimeout:     config.DefaultReceiveTimeout,
		SendDeadline:       config.DefaultSendDeadline,
		TCPKeepAlive:       false,
		TCPKeepAlivePeriod: config.DefaultTCPKeepAlivePeriod,
	}
	newPool.Put("client", NewClient(context.Background(), &clientConfig, logger)) //nolint:errcheck

	// Create a proxy with a fixed buffer newPool
	proxy := NewProxy(
		context.Background(),
		newPool,
		plugin.NewRegistry(
			context.Background(),
			config.Loose,
			config.PassDown,
			config.Accept,
			config.Stop,
			logger,
			false,
		),
		false,
		false,
		config.DefaultHealthCheckPeriod,
		&clientConfig,
		logger,
		config.DefaultPluginTimeout)
	defer proxy.Shutdown()

	conn := testConnection{}
	proxy.Connect(conn.Conn)      //nolint:errcheck
	defer proxy.Disconnect(&conn) //nolint:errcheck

	// Connect to the proxy
	for i := 0; i < b.N; i++ {
		proxy.PassThroughToClient(&conn) //nolint:errcheck
		proxy.PassThroughToServer(&conn) //nolint:errcheck
	}
}

func BenchmarkProxyIsHealthyAndIsExhausted(b *testing.B) {
	logger := logging.NewLogger(context.Background(), logging.LoggerConfig{
		Output:            []config.LogOutput{config.Console},
		TimeFormat:        zerolog.TimeFormatUnix,
		ConsoleTimeFormat: time.RFC3339,
		Level:             zerolog.PanicLevel,
		NoColor:           true,
	})

	// Create a connection newPool
	newPool := pool.NewPool(context.Background(), 1)

	clientConfig := config.Client{
		Network:            "tcp",
		Address:            "localhost:5432",
		ReceiveChunkSize:   config.DefaultChunkSize,
		ReceiveDeadline:    config.DefaultReceiveDeadline,
		ReceiveTimeout:     config.DefaultReceiveTimeout,
		SendDeadline:       config.DefaultSendDeadline,
		TCPKeepAlive:       false,
		TCPKeepAlivePeriod: config.DefaultTCPKeepAlivePeriod,
	}
	client := NewClient(context.Background(), &clientConfig, logger)
	newPool.Put("client", client) //nolint:errcheck

	// Create a proxy with a fixed buffer newPool
	proxy := NewProxy(
		context.Background(),
		newPool,
		plugin.NewRegistry(
			context.Background(),
			config.Loose,
			config.PassDown,
			config.Accept,
			config.Stop,
			logger,
			false,
		),
		false,
		false,
		config.DefaultHealthCheckPeriod,
		&clientConfig,
		logger,
		config.DefaultPluginTimeout)
	defer proxy.Shutdown()

	conn := testConnection{}
	proxy.Connect(conn.Conn)      //nolint:errcheck
	defer proxy.Disconnect(&conn) //nolint:errcheck

	// Connect to the proxy
	for i := 0; i < b.N; i++ {
		proxy.IsHealty(client) //nolint:errcheck
		proxy.IsExhausted()
	}
}

func BenchmarkProxyAvailableAndBusyConnections(b *testing.B) {
	logger := logging.NewLogger(context.Background(), logging.LoggerConfig{
		Output:            []config.LogOutput{config.Console},
		TimeFormat:        zerolog.TimeFormatUnix,
		ConsoleTimeFormat: time.RFC3339,
		Level:             zerolog.PanicLevel,
		NoColor:           true,
	})

	// Create a connection newPool
	newPool := pool.NewPool(context.Background(), 1)

	clientConfig := config.Client{
		Network:            "tcp",
		Address:            "localhost:5432",
		ReceiveChunkSize:   config.DefaultChunkSize,
		ReceiveDeadline:    config.DefaultReceiveDeadline,
		ReceiveTimeout:     config.DefaultReceiveTimeout,
		SendDeadline:       config.DefaultSendDeadline,
		TCPKeepAlive:       false,
		TCPKeepAlivePeriod: config.DefaultTCPKeepAlivePeriod,
	}
	client := NewClient(context.Background(), &clientConfig, logger)
	newPool.Put("client", client) //nolint:errcheck

	// Create a proxy with a fixed buffer newPool
	proxy := NewProxy(
		context.Background(),
		newPool,
		plugin.NewRegistry(
			context.Background(),
			config.Loose,
			config.PassDown,
			config.Accept,
			config.Stop,
			logger,
			false,
		),
		false,
		false,
		config.DefaultHealthCheckPeriod,
		&clientConfig,
		logger,
		config.DefaultPluginTimeout)
	defer proxy.Shutdown()

	conn := testConnection{}
	proxy.Connect(conn.Conn)      //nolint:errcheck
	defer proxy.Disconnect(&conn) //nolint:errcheck

	// Connect to the proxy
	for i := 0; i < b.N; i++ {
		proxy.AvailableConnections()
		proxy.BusyConnections()
	}
}
