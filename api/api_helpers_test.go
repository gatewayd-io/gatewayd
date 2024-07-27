package api

import (
	"context"

	"github.com/gatewayd-io/gatewayd/act"
	"github.com/gatewayd-io/gatewayd/config"
	"github.com/gatewayd-io/gatewayd/network"
	"github.com/gatewayd-io/gatewayd/plugin"
	"github.com/gatewayd-io/gatewayd/pool"
	"github.com/rs/zerolog"
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
