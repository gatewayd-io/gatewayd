package network

import (
	"testing"
	"time"

	"github.com/gatewayd-io/gatewayd/act"
	"github.com/gatewayd-io/gatewayd/config"
	"github.com/gatewayd-io/gatewayd/logging"
	"github.com/gatewayd-io/gatewayd/plugin"
	"github.com/gatewayd-io/gatewayd/pool"
	"github.com/gatewayd-io/gatewayd/testhelpers"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
)

// TestNewProxy tests the creation of a new proxy with a fixed connection pool.
func TestNewProxy(t *testing.T) {
	postgresHostIP, postgresMappedPort := testhelpers.SetupPostgreSQLTestContainer(t.Context(), t)

	logger := logging.NewLogger(t.Context(), logging.LoggerConfig{
		Output:            []config.LogOutput{config.Console},
		TimeFormat:        zerolog.TimeFormatUnix,
		ConsoleTimeFormat: time.RFC3339,
		Level:             zerolog.WarnLevel,
		NoColor:           true,
	})

	// Create a connection newPool
	newPool := pool.NewPool(t.Context(), config.EmptyPoolCapacity)

	client := NewClient(
		t.Context(),
		&config.Client{
			Network:            "tcp",
			Address:            postgresHostIP + ":" + postgresMappedPort.Port(),
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

	// Create a new act registry
	actRegistry := act.NewActRegistry(
		act.Registry{
			Signals:              act.BuiltinSignals(),
			Policies:             act.BuiltinPolicies(),
			Actions:              act.BuiltinActions(),
			DefaultPolicyName:    config.DefaultPolicy,
			PolicyTimeout:        config.DefaultPolicyTimeout,
			DefaultActionTimeout: config.DefaultActionTimeout,
			Logger:               logger,
		})

	// Create a proxy with a fixed buffer newPool
	proxy := NewProxy(
		t.Context(),
		Proxy{
			AvailableConnections: newPool,
			PluginRegistry: plugin.NewRegistry(
				t.Context(),
				plugin.Registry{
					ActRegistry: actRegistry,
					Logger:      logger,
				},
			),
			HealthCheckPeriod: config.DefaultHealthCheckPeriod,
			Logger:            logger,
			PluginTimeout:     config.DefaultPluginTimeout,
		},
	)
	defer proxy.Shutdown()

	assert.NotNil(t, proxy)
	assert.Equal(t, 0, proxy.busyConnections.Size(), "Proxy should have no connected clients")
	assert.Equal(t, 1, proxy.AvailableConnections.Size())
	if c, ok := proxy.AvailableConnections.Pop(client.ID).(*Client); ok {
		assert.NotEqual(t, "", c.ID)
	}
	assert.False(t, proxy.IsExhausted())
	c, err := proxy.IsHealthy(client)
	assert.Nil(t, err)
	assert.Equal(t, client, c)
}

func BenchmarkNewProxy(b *testing.B) {
	logger := logging.NewLogger(b.Context(), logging.LoggerConfig{
		Output:            []config.LogOutput{config.Console},
		TimeFormat:        zerolog.TimeFormatUnix,
		ConsoleTimeFormat: time.RFC3339,
		Level:             zerolog.WarnLevel,
		NoColor:           true,
	})

	// Create a connection newPool
	newPool := pool.NewPool(b.Context(), config.EmptyPoolCapacity)

	// Create a new act registry
	actRegistry := act.NewActRegistry(
		act.Registry{
			Signals:              act.BuiltinSignals(),
			Policies:             act.BuiltinPolicies(),
			Actions:              act.BuiltinActions(),
			DefaultPolicyName:    config.DefaultPolicy,
			PolicyTimeout:        config.DefaultPolicyTimeout,
			DefaultActionTimeout: config.DefaultActionTimeout,
			Logger:               logger,
		})

	// Create a proxy with a fixed buffer newPool
	for range b.N {
		proxy := NewProxy(
			b.Context(),
			Proxy{
				AvailableConnections: newPool,
				PluginRegistry: plugin.NewRegistry(
					b.Context(),
					plugin.Registry{
						ActRegistry: actRegistry,
						Logger:      logger,
					},
				),
				HealthCheckPeriod: config.DefaultHealthCheckPeriod,
				Logger:            logger,
				PluginTimeout:     config.DefaultPluginTimeout,
			},
		)
		proxy.Shutdown()
	}
}

func BenchmarkProxyConnectDisconnect(b *testing.B) {
	logger := logging.NewLogger(b.Context(), logging.LoggerConfig{
		Output:            []config.LogOutput{config.Console},
		TimeFormat:        zerolog.TimeFormatUnix,
		ConsoleTimeFormat: time.RFC3339,
		Level:             zerolog.PanicLevel,
		NoColor:           true,
	})

	// Create a connection newPool
	newPool := pool.NewPool(b.Context(), 1)

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
	newPool.Put("client", NewClient(b.Context(), &clientConfig, logger, nil)) //nolint:errcheck

	// Create a new act registry
	actRegistry := act.NewActRegistry(
		act.Registry{
			Signals:              act.BuiltinSignals(),
			Policies:             act.BuiltinPolicies(),
			Actions:              act.BuiltinActions(),
			DefaultPolicyName:    config.DefaultPolicy,
			PolicyTimeout:        config.DefaultPolicyTimeout,
			DefaultActionTimeout: config.DefaultActionTimeout,
			Logger:               logger,
		})

	// Create a proxy with a fixed buffer newPool
	proxy := NewProxy(
		b.Context(),
		Proxy{
			AvailableConnections: newPool,
			PluginRegistry: plugin.NewRegistry(
				b.Context(),
				plugin.Registry{
					ActRegistry: actRegistry,
					Logger:      logger,
				},
			),
			HealthCheckPeriod: config.DefaultHealthCheckPeriod,
			ClientConfig:      &clientConfig,
			Logger:            logger,
			PluginTimeout:     config.DefaultPluginTimeout,
		},
	)
	defer proxy.Shutdown()

	conn := testConnection{}

	// Connect to the proxy
	for range b.N {
		proxy.Connect(conn.ConnWrapper)    //nolint:errcheck
		proxy.Disconnect(conn.ConnWrapper) //nolint:errcheck
	}
}

func BenchmarkProxyPassThrough(b *testing.B) {
	logger := logging.NewLogger(b.Context(), logging.LoggerConfig{
		Output:            []config.LogOutput{config.Console},
		TimeFormat:        zerolog.TimeFormatUnix,
		ConsoleTimeFormat: time.RFC3339,
		Level:             zerolog.PanicLevel,
		NoColor:           true,
	})

	// Create a connection newPool
	newPool := pool.NewPool(b.Context(), 1)

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
	newPool.Put("client", NewClient(b.Context(), &clientConfig, logger, nil)) //nolint:errcheck

	// Create a new act registry
	actRegistry := act.NewActRegistry(
		act.Registry{
			Signals:              act.BuiltinSignals(),
			Policies:             act.BuiltinPolicies(),
			Actions:              act.BuiltinActions(),
			DefaultPolicyName:    config.DefaultPolicy,
			PolicyTimeout:        config.DefaultPolicyTimeout,
			DefaultActionTimeout: config.DefaultActionTimeout,
			Logger:               logger,
		})

	// Create a proxy with a fixed buffer newPool
	proxy := NewProxy(
		b.Context(),
		Proxy{
			AvailableConnections: newPool,
			PluginRegistry: plugin.NewRegistry(
				b.Context(),
				plugin.Registry{
					ActRegistry: actRegistry,
					Logger:      logger,
				},
			),
			HealthCheckPeriod: config.DefaultHealthCheckPeriod,
			ClientConfig:      &clientConfig,
			Logger:            logger,
			PluginTimeout:     config.DefaultPluginTimeout,
		},
	)
	defer proxy.Shutdown()

	conn := testConnection{}
	proxy.Connect(conn.ConnWrapper)          //nolint:errcheck
	defer proxy.Disconnect(conn.ConnWrapper) //nolint:errcheck

	stack := NewStack()

	// Connect to the proxy
	for range b.N {
		proxy.PassThroughToClient(conn.ConnWrapper, stack) //nolint:errcheck
		proxy.PassThroughToServer(conn.ConnWrapper, stack) //nolint:errcheck
	}
}

func BenchmarkProxyIsHealthyAndIsExhausted(b *testing.B) {
	logger := logging.NewLogger(b.Context(), logging.LoggerConfig{
		Output:            []config.LogOutput{config.Console},
		TimeFormat:        zerolog.TimeFormatUnix,
		ConsoleTimeFormat: time.RFC3339,
		Level:             zerolog.PanicLevel,
		NoColor:           true,
	})

	// Create a connection newPool
	newPool := pool.NewPool(b.Context(), 1)

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
	client := NewClient(b.Context(), &clientConfig, logger, nil)
	newPool.Put("client", client) //nolint:errcheck

	// Create a new act registry
	actRegistry := act.NewActRegistry(
		act.Registry{
			Signals:              act.BuiltinSignals(),
			Policies:             act.BuiltinPolicies(),
			Actions:              act.BuiltinActions(),
			DefaultPolicyName:    config.DefaultPolicy,
			PolicyTimeout:        config.DefaultPolicyTimeout,
			DefaultActionTimeout: config.DefaultActionTimeout,
			Logger:               logger,
		})

	// Create a proxy with a fixed buffer newPool
	proxy := NewProxy(
		b.Context(),
		Proxy{
			AvailableConnections: newPool,
			PluginRegistry: plugin.NewRegistry(
				b.Context(),
				plugin.Registry{
					ActRegistry: actRegistry,
					Logger:      logger,
				},
			),
			HealthCheckPeriod: config.DefaultHealthCheckPeriod,
			ClientConfig:      &clientConfig,
			Logger:            logger,
			PluginTimeout:     config.DefaultPluginTimeout,
		},
	)
	defer proxy.Shutdown()

	conn := testConnection{}
	proxy.Connect(conn.ConnWrapper)          //nolint:errcheck
	defer proxy.Disconnect(conn.ConnWrapper) //nolint:errcheck

	// Connect to the proxy
	for range b.N {
		proxy.IsHealthy(client) //nolint:errcheck
		proxy.IsExhausted()
	}
}

func BenchmarkProxyAvailableAndBusyConnectionsString(b *testing.B) {
	logger := logging.NewLogger(b.Context(), logging.LoggerConfig{
		Output:            []config.LogOutput{config.Console},
		TimeFormat:        zerolog.TimeFormatUnix,
		ConsoleTimeFormat: time.RFC3339,
		Level:             zerolog.PanicLevel,
		NoColor:           true,
	})

	// Create a connection newPool
	newPool := pool.NewPool(b.Context(), 1)

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
	client := NewClient(b.Context(), &clientConfig, logger, nil)
	newPool.Put("client", client) //nolint:errcheck

	// Create a new act registry
	actRegistry := act.NewActRegistry(
		act.Registry{
			Signals:              act.BuiltinSignals(),
			Policies:             act.BuiltinPolicies(),
			Actions:              act.BuiltinActions(),
			DefaultPolicyName:    config.DefaultPolicy,
			PolicyTimeout:        config.DefaultPolicyTimeout,
			DefaultActionTimeout: config.DefaultActionTimeout,
			Logger:               logger,
		})

	// Create a proxy with a fixed buffer newPool
	proxy := NewProxy(
		b.Context(),
		Proxy{
			AvailableConnections: newPool,
			PluginRegistry: plugin.NewRegistry(
				b.Context(),
				plugin.Registry{
					ActRegistry: actRegistry,
					Logger:      logger,
				},
			),
			HealthCheckPeriod: config.DefaultHealthCheckPeriod,
			ClientConfig:      &clientConfig,
			Logger:            logger,
			PluginTimeout:     config.DefaultPluginTimeout,
		},
	)
	defer proxy.Shutdown()

	conn := testConnection{}
	proxy.Connect(conn.ConnWrapper)          //nolint:errcheck
	defer proxy.Disconnect(conn.ConnWrapper) //nolint:errcheck

	// Connect to the proxy
	for range b.N {
		proxy.AvailableConnectionsString()
		proxy.BusyConnectionsString()
	}
}
