package api

import (
	"context"
	"testing"

	"github.com/docker/go-connections/nat"
	"github.com/gatewayd-io/gatewayd/act"
	"github.com/gatewayd-io/gatewayd/config"
	"github.com/gatewayd-io/gatewayd/network"
	"github.com/gatewayd-io/gatewayd/plugin"
	"github.com/gatewayd-io/gatewayd/pool"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
)

// getAPIConfig returns a new API configuration with all the necessary components.
func getAPIConfig() *API {
	logger := zerolog.New(nil)
	defaultPool := pool.NewPool(context.Background(), config.DefaultPoolSize)
	pluginReg := plugin.NewRegistry(
		context.Background(),
		plugin.Registry{
			ActRegistry: act.NewActRegistry(act.Registry{
				Logger:               logger,
				PolicyTimeout:        config.DefaultPolicyTimeout,
				DefaultActionTimeout: config.DefaultActionTimeout,
				Signals:              act.BuiltinSignals(),
				Policies:             act.BuiltinPolicies(),
				Actions:              act.BuiltinActions(),
				DefaultPolicyName:    config.DefaultPolicy,
			}),
			DevMode:       true,
			Logger:        logger,
			Compatibility: config.DefaultCompatibilityPolicy,
			StartTimeout:  config.DefaultPluginStartTimeout,
		},
	)
	defaultProxy := network.NewProxy(
		context.Background(),
		network.Proxy{
			AvailableConnections: defaultPool,
			Logger:               logger,
			PluginRegistry:       pluginReg,
			PluginTimeout:        config.DefaultPluginTimeout,
			HealthCheckPeriod:    config.DefaultHealthCheckPeriod,
			ClientConfig:         &config.Client{},
		},
	)
	servers := map[string]*network.Server{
		config.Default: network.NewServer(
			context.Background(),
			network.Server{
				Logger:         logger,
				Proxies:        []network.IProxy{defaultProxy},
				PluginRegistry: pluginReg,
				PluginTimeout:  config.DefaultPluginTimeout,
				Network:        "tcp",
				Address:        "localhost:15432",
			},
		),
	}
	return &API{
		Options: &Options{
			GRPCNetwork: "tcp",
			GRPCAddress: "localhost:19090",
			HTTPAddress: "localhost:18080",
			Logger:      logger,
			Servers:     servers,
		},
		Config: config.NewConfig(
			context.Background(),
			config.Config{
				GlobalConfigFile: "gatewayd.yaml",
				PluginConfigFile: "gatewayd_plugins.yaml",
			},
		),
		PluginRegistry: pluginReg,
		Pools: map[string]map[string]*pool.Pool{
			config.Default: {config.DefaultConfigurationBlock: defaultPool},
		},
		Proxies: map[string]map[string]*network.Proxy{
			config.Default: {config.DefaultConfigurationBlock: defaultProxy},
		},
		Servers: servers,
	}
}

// setupTestContainer initializes and starts the PostgreSQL test container.
func setupPostgreSQLTestContainer(ctx context.Context, t *testing.T) (string, nat.Port) {
	t.Helper()

	postgresPort := "5432"
	container, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: testcontainers.ContainerRequest{
			Image:        "postgres:latest",
			ExposedPorts: []string{postgresPort + "/tcp"},
			Env: map[string]string{
				"POSTGRES_USER":     "postgres",
				"POSTGRES_PASSWORD": "postgres",
			},
			WaitingFor: wait.ForAll(
				wait.ForLog("database system is ready to accept connections"),
				wait.ForListeningPort(nat.Port(postgresPort+"/tcp")),
			),
		},
		Started: true,
	})
	require.NoError(t, err)

	hostIP, err := container.Host(ctx)
	require.NoError(t, err, "Failed to retrieve PostgreSQL test container host IP")

	mappedPort, err := container.MappedPort(ctx, nat.Port(postgresPort+"/tcp"))
	require.NoError(t, err, "Failed to map PostgreSQL test container port")

	return hostIP, mappedPort
}
