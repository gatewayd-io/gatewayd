package network

import (
	"context"
	"testing"
	"time"

	"github.com/gatewayd-io/gatewayd/act"
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
			DialTimeout:        config.DefaultDialTimeout,
			TCPKeepAlive:       false,
			TCPKeepAlivePeriod: config.DefaultTCPKeepAlivePeriod,
		},
		logger,
		nil)
	err := newPool.Put(client.ID, client)
	assert.Nil(t, err)

	// Create a new Act registry
	actRegistry := act.NewRegistry(
		act.BuiltinSignals(), act.BuiltinPolicies(), act.BuiltinActions(),
		config.DefaultPolicy, config.DefaultPolicyTimeout, logger)

	// Create a proxy with a fixed buffer newPool
	proxy := NewProxy(
		context.Background(),
		newPool,
		plugin.NewRegistry(
			context.Background(),
			actRegistry,
			config.Loose,
			config.Stop,
			logger,
			false,
		),
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
	assert.False(t, proxy.IsExhausted())
	c, err := proxy.IsHealthy(client)
	assert.Nil(t, err)
	assert.Equal(t, client, c)
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

	// Create a new Act registry
	actRegistry := act.NewRegistry(
		act.BuiltinSignals(), act.BuiltinPolicies(), act.BuiltinActions(),
		config.DefaultPolicy, config.DefaultPolicyTimeout, logger)

	// Create a proxy with a fixed buffer newPool
	for i := 0; i < b.N; i++ {
		proxy := NewProxy(
			context.Background(),
			newPool,
			plugin.NewRegistry(
				context.Background(),
				actRegistry,
				config.Loose,
				config.Stop,
				logger,
				false,
			),
			config.DefaultHealthCheckPeriod,
			nil,
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
	newPool.Put("client", NewClient(context.Background(), &clientConfig, logger, nil)) //nolint:errcheck

	// Create a new Act registry
	actRegistry := act.NewRegistry(
		act.BuiltinSignals(), act.BuiltinPolicies(), act.BuiltinActions(),
		config.DefaultPolicy, config.DefaultPolicyTimeout, logger)

	// Create a proxy with a fixed buffer newPool
	proxy := NewProxy(
		context.Background(),
		newPool,
		plugin.NewRegistry(
			context.Background(),
			actRegistry,
			config.Loose,
			config.Stop,
			logger,
			false,
		),
		config.DefaultHealthCheckPeriod,
		&clientConfig,
		logger,
		config.DefaultPluginTimeout)
	defer proxy.Shutdown()

	conn := testConnection{}

	// Connect to the proxy
	for i := 0; i < b.N; i++ {
		proxy.Connect(conn.ConnWrapper)    //nolint:errcheck
		proxy.Disconnect(conn.ConnWrapper) //nolint:errcheck
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
	newPool.Put("client", NewClient(context.Background(), &clientConfig, logger, nil)) //nolint:errcheck

	// Create a new Act registry
	actRegistry := act.NewRegistry(
		act.BuiltinSignals(), act.BuiltinPolicies(), act.BuiltinActions(),
		config.DefaultPolicy, config.DefaultPolicyTimeout, logger)

	// Create a proxy with a fixed buffer newPool
	proxy := NewProxy(
		context.Background(),
		newPool,
		plugin.NewRegistry(
			context.Background(),
			actRegistry,
			config.Loose,
			config.Stop,
			logger,
			false,
		),
		config.DefaultHealthCheckPeriod,
		&clientConfig,
		logger,
		config.DefaultPluginTimeout)
	defer proxy.Shutdown()

	conn := testConnection{}
	proxy.Connect(conn.ConnWrapper)          //nolint:errcheck
	defer proxy.Disconnect(conn.ConnWrapper) //nolint:errcheck

	stack := NewStack()

	// Connect to the proxy
	for i := 0; i < b.N; i++ {
		proxy.PassThroughToClient(conn.ConnWrapper, stack) //nolint:errcheck
		proxy.PassThroughToServer(conn.ConnWrapper, stack) //nolint:errcheck
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
	client := NewClient(context.Background(), &clientConfig, logger, nil)
	newPool.Put("client", client) //nolint:errcheck

	// Create a new Act registry
	actRegistry := act.NewRegistry(
		act.BuiltinSignals(), act.BuiltinPolicies(), act.BuiltinActions(),
		config.DefaultPolicy, config.DefaultPolicyTimeout, logger)

	// Create a proxy with a fixed buffer newPool
	proxy := NewProxy(
		context.Background(),
		newPool,
		plugin.NewRegistry(
			context.Background(),
			actRegistry,
			config.Loose,
			config.Stop,
			logger,
			false,
		),
		config.DefaultHealthCheckPeriod,
		&clientConfig,
		logger,
		config.DefaultPluginTimeout)
	defer proxy.Shutdown()

	conn := testConnection{}
	proxy.Connect(conn.ConnWrapper)          //nolint:errcheck
	defer proxy.Disconnect(conn.ConnWrapper) //nolint:errcheck

	// Connect to the proxy
	for i := 0; i < b.N; i++ {
		proxy.IsHealthy(client) //nolint:errcheck
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
	client := NewClient(context.Background(), &clientConfig, logger, nil)
	newPool.Put("client", client) //nolint:errcheck

	// Create a new Act registry
	actRegistry := act.NewRegistry(
		act.BuiltinSignals(), act.BuiltinPolicies(), act.BuiltinActions(),
		config.DefaultPolicy, config.DefaultPolicyTimeout, logger)

	// Create a proxy with a fixed buffer newPool
	proxy := NewProxy(
		context.Background(),
		newPool,
		plugin.NewRegistry(
			context.Background(),
			actRegistry,
			config.Loose,
			config.Stop,
			logger,
			false,
		),
		config.DefaultHealthCheckPeriod,
		&clientConfig,
		logger,
		config.DefaultPluginTimeout)
	defer proxy.Shutdown()

	conn := testConnection{}
	proxy.Connect(conn.ConnWrapper)          //nolint:errcheck
	defer proxy.Disconnect(conn.ConnWrapper) //nolint:errcheck

	// Connect to the proxy
	for i := 0; i < b.N; i++ {
		proxy.AvailableConnections()
		proxy.BusyConnections()
	}
}
