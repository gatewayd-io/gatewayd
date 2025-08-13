package api

import (
	"testing"
	"time"

	"github.com/gatewayd-io/gatewayd/act"
	"github.com/gatewayd-io/gatewayd/config"
	"github.com/gatewayd-io/gatewayd/network"
	"github.com/gatewayd-io/gatewayd/plugin"
	"github.com/gatewayd-io/gatewayd/pool"
	"github.com/gatewayd-io/gatewayd/testhelpers"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/health/grpc_health_v1"
)

func Test_Healthchecker(t *testing.T) {
	postgresHostIP, postgresMappedPort := testhelpers.SetupPostgreSQLTestContainer(t.Context(), t)
	postgresAddress := postgresHostIP + ":" + postgresMappedPort.Port()
	clientConfig := &config.Client{
		Network: config.DefaultNetwork,
		Address: postgresAddress,
	}
	client := network.NewClient(t.Context(), clientConfig, zerolog.Nop(), nil)
	newPool := pool.NewPool(t.Context(), 1)
	require.NotNil(t, newPool)
	assert.Nil(t, newPool.Put(client.ID, client))

	proxy := network.NewProxy(
		t.Context(),
		network.Proxy{
			AvailableConnections: newPool,
			HealthCheckPeriod:    config.DefaultHealthCheckPeriod,
			ClientConfig: &config.Client{
				Network: config.DefaultNetwork,
				Address: postgresAddress,
			},
			Logger:        zerolog.Nop(),
			PluginTimeout: config.DefaultPluginTimeout,
		},
	)

	actRegistry := act.NewActRegistry(
		act.Registry{
			Signals:              act.BuiltinSignals(),
			Policies:             act.BuiltinPolicies(),
			Actions:              act.BuiltinActions(),
			DefaultPolicyName:    config.DefaultPolicy,
			PolicyTimeout:        config.DefaultPolicyTimeout,
			DefaultActionTimeout: config.DefaultActionTimeout,
			Logger:               zerolog.Nop(),
		})

	pluginRegistry := plugin.NewRegistry(
		t.Context(),
		plugin.Registry{
			ActRegistry: actRegistry,
			Logger:      zerolog.Nop(),
			DevMode:     true,
		},
	)

	raftHelper, err := testhelpers.NewTestRaftNode(t)
	if err != nil {
		t.Fatalf("Failed to create test raft node: %v", err)
	}
	defer func() {
		if err := raftHelper.Cleanup(); err != nil {
			t.Errorf("Failed to cleanup raft: %v", err)
		}
	}()

	server := network.NewServer(
		t.Context(),
		network.Server{
			Network:      config.DefaultNetwork,
			Address:      "127.0.0.1:15432",
			TickInterval: config.DefaultTickInterval,
			Options: network.Option{
				EnableTicker: false,
			},
			Proxies:          []network.IProxy{proxy},
			Logger:           zerolog.Nop(),
			PluginRegistry:   pluginRegistry,
			PluginTimeout:    config.DefaultPluginTimeout,
			HandshakeTimeout: config.DefaultHandshakeTimeout,
			RaftNode:         raftHelper.Node,
		},
	)

	healthchecker := HealthChecker{
		Servers: map[string]*network.Server{
			config.Default: server,
		},
		raftNode: raftHelper.Node,
	}
	assert.NotNil(t, healthchecker)
	hcr, err := healthchecker.Check(t.Context(), &grpc_health_v1.HealthCheckRequest{})
	assert.NoError(t, err)
	assert.NotNil(t, hcr)
	assert.Equal(t, grpc_health_v1.HealthCheckResponse_NOT_SERVING, hcr.GetStatus())

	go func(t *testing.T, server *network.Server) {
		t.Helper()

		if err := server.Run(); err != nil {
			t.Errorf("server.Run() error = %v", err)
		}
	}(t, server)

	time.Sleep(1 * time.Second)
	// Test for SERVING status
	hcr, err = healthchecker.Check(t.Context(), &grpc_health_v1.HealthCheckRequest{})
	assert.NoError(t, err)
	assert.NotNil(t, hcr)
	assert.Equal(t, grpc_health_v1.HealthCheckResponse_SERVING, hcr.GetStatus())

	err = healthchecker.Watch(&grpc_health_v1.HealthCheckRequest{}, nil)
	assert.Error(t, err)
	assert.Equal(t, "rpc error: code = Unimplemented desc = not implemented", err.Error())

	server.Shutdown()
	pluginRegistry.Shutdown()

	// Wait for the server to stop.
	<-time.After(100 * time.Millisecond)

	// check server status and connections
	assert.False(t, server.IsRunning())
	assert.Zero(t, server.CountConnections())
}
