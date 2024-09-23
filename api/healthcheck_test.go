package api

import (
	"context"
	"testing"

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
	postgresHostIP, postgresMappedPort := testhelpers.SetupPostgreSQLTestContainer(context.Background(), t)
	postgresAddress := postgresHostIP + ":" + postgresMappedPort.Port()
	clientConfig := &config.Client{
		Network: config.DefaultNetwork,
		Address: postgresAddress,
	}
	client := network.NewClient(context.TODO(), clientConfig, zerolog.Logger{}, nil)
	newPool := pool.NewPool(context.TODO(), 1)
	require.NotNil(t, newPool)
	assert.Nil(t, newPool.Put(client.ID, client))

	proxy := network.NewProxy(
		context.TODO(),
		network.Proxy{
			AvailableConnections: newPool,
			HealthCheckPeriod:    config.DefaultHealthCheckPeriod,
			ClientConfig: &config.Client{
				Network: config.DefaultNetwork,
				Address: postgresAddress,
			},
			Logger:        zerolog.Logger{},
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
			Logger:               zerolog.Logger{},
		})

	pluginRegistry := plugin.NewRegistry(
		context.TODO(),
		plugin.Registry{
			ActRegistry:   actRegistry,
			Compatibility: config.Loose,
			Logger:        zerolog.Logger{},
			DevMode:       true,
		},
	)

	server := network.NewServer(
		context.TODO(),
		network.Server{
			Network:      config.DefaultNetwork,
			Address:      postgresAddress,
			TickInterval: config.DefaultTickInterval,
			Options: network.Option{
				EnableTicker: false,
			},
			Proxies:          []network.IProxy{proxy},
			Logger:           zerolog.Logger{},
			PluginRegistry:   pluginRegistry,
			PluginTimeout:    config.DefaultPluginTimeout,
			HandshakeTimeout: config.DefaultHandshakeTimeout,
		},
	)

	healthchecker := HealthChecker{
		Servers: map[string]*network.Server{
			config.Default: server,
		},
	}
	assert.NotNil(t, healthchecker)
	hcr, err := healthchecker.Check(context.TODO(), &grpc_health_v1.HealthCheckRequest{})
	assert.NoError(t, err)
	assert.NotNil(t, hcr)
	assert.Equal(t, grpc_health_v1.HealthCheckResponse_NOT_SERVING, hcr.GetStatus())

	err = healthchecker.Watch(&grpc_health_v1.HealthCheckRequest{}, nil)
	assert.Error(t, err)
	assert.Equal(t, "rpc error: code = Unimplemented desc = not implemented", err.Error())
}
