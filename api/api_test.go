package api

import (
	"context"
	"regexp"
	"testing"

	sdkPlugin "github.com/gatewayd-io/gatewayd-plugin-sdk/plugin"
	"github.com/gatewayd-io/gatewayd/act"
	v1 "github.com/gatewayd-io/gatewayd/api/v1"
	"github.com/gatewayd-io/gatewayd/config"
	"github.com/gatewayd-io/gatewayd/network"
	"github.com/gatewayd-io/gatewayd/plugin"
	"github.com/gatewayd-io/gatewayd/pool"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/emptypb"
)

func TestGetVersion(t *testing.T) {
	api := API{}
	version, err := api.Version(context.Background(), &emptypb.Empty{})
	require.NoError(t, err)
	assert.Regexp(t, regexp.MustCompile(`^\d+\.\d+\.\d+$`), version.GetVersion())
	assert.Regexp(t, regexp.MustCompile(`^GatewayD \d+\.\d+\.\d+ \(, go\d+\.\d+\.\d+, \w+/\w+\)$`), version.GetVersionInfo()) //nolint:lll
}

func TestGetGlobalConfig(t *testing.T) {
	// Load config from the default config file.
	conf := config.NewConfig(context.TODO(),
		config.Config{GlobalConfigFile: "../gatewayd.yaml", PluginConfigFile: "../gatewayd_plugins.yaml"})
	conf.InitConfig(context.TODO())
	assert.NotEmpty(t, conf.Global)

	api := API{
		Config: conf,
	}
	globalConfig, err := api.GetGlobalConfig(context.Background(), &v1.Group{GroupName: nil})
	require.NoError(t, err)
	globalconf := globalConfig.AsMap()
	assert.NotEmpty(t, globalconf)
	assert.NotEmpty(t, globalconf["loggers"])
	assert.NotEmpty(t, globalconf["clients"])
	assert.NotEmpty(t, globalconf["pools"])
	assert.NotEmpty(t, globalconf["proxies"])
	assert.NotEmpty(t, globalconf["servers"])
	assert.NotEmpty(t, globalconf["metrics"])
	assert.NotEmpty(t, globalconf["api"])
}

func TestGetGlobalConfigWithGroupName(t *testing.T) {
	// Load config from the default config file.
	conf := config.NewConfig(context.TODO(),
		config.Config{GlobalConfigFile: "../gatewayd.yaml", PluginConfigFile: "../gatewayd_plugins.yaml"})
	conf.InitConfig(context.TODO())
	assert.NotEmpty(t, conf.Global)

	api := API{
		Config: conf,
	}
	globalConfig, err := api.GetGlobalConfig(context.Background(), &v1.Group{GroupName: nil})
	require.NoError(t, err)
	globalconf := globalConfig.AsMap()
	assert.NotEmpty(t, globalconf)
	assert.NotEmpty(t, globalconf)
	assert.NotEmpty(t, globalconf["loggers"])
	assert.NotEmpty(t, globalconf["clients"])
	assert.NotEmpty(t, globalconf["pools"])
	assert.NotEmpty(t, globalconf["proxies"])
	assert.NotEmpty(t, globalconf["servers"])
	assert.NotEmpty(t, globalconf["metrics"])
	assert.NotEmpty(t, globalconf["api"])
	if _, ok := globalconf["loggers"].(map[string]interface{})["default"]; !ok {
		t.Errorf("loggers.default is not found")
	}
}

func TestGetGlobalConfigWithNonExistingGroupName(t *testing.T) {
	// Load config from the default config file.
	conf := config.NewConfig(context.TODO(),
		config.Config{GlobalConfigFile: "../gatewayd.yaml", PluginConfigFile: "../gatewayd_plugins.yaml"})
	conf.InitConfig(context.TODO())
	assert.NotEmpty(t, conf.Global)

	api := API{
		Config: conf,
	}
	nonExistingGroupName := "non-existing-group"
	_, err := api.GetGlobalConfig(context.Background(), &v1.Group{GroupName: &nonExistingGroupName})
	require.Error(t, err)
	assert.Errorf(t, err, "group not found")
}

func TestGetPluginConfig(t *testing.T) {
	// Load config from the default config file.
	conf := config.NewConfig(context.TODO(),
		config.Config{GlobalConfigFile: "../gatewayd.yaml", PluginConfigFile: "../gatewayd_plugins.yaml"})
	conf.InitConfig(context.TODO())
	assert.NotEmpty(t, conf.Global)

	api := API{
		Config: conf,
	}
	pluginConfig, err := api.GetPluginConfig(context.Background(), &emptypb.Empty{})
	require.NoError(t, err)
	pluginconf := pluginConfig.AsMap()
	assert.NotEmpty(t, pluginconf)
	assert.NotEmpty(t, pluginconf["plugins"])
}

func TestGetPlugins(t *testing.T) {
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
	pluginRegistry.Add(&plugin.Plugin{
		ID: sdkPlugin.Identifier{
			Name:      "plugin-name",
			Version:   "plugin-version",
			RemoteURL: "plugin-url",
			Checksum:  "plugin-checksum",
		},
	})

	api := API{
		PluginRegistry: pluginRegistry,
	}
	plugins, err := api.GetPlugins(context.Background(), &emptypb.Empty{})
	require.NoError(t, err)
	assert.NotEmpty(t, plugins)
	assert.NotEmpty(t, plugins.GetConfigs())
}

func TestGetPluginsWithEmptyPluginRegistry(t *testing.T) {
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

	api := API{
		PluginRegistry: pluginRegistry,
	}
	plugins, err := api.GetPlugins(context.Background(), &emptypb.Empty{})
	require.NoError(t, err)
	assert.NotEmpty(t, plugins)
	assert.Empty(t, plugins.GetConfigs())
}

func TestPools(t *testing.T) {
	api := API{
		Pools: map[string]*pool.Pool{
			config.Default: pool.NewPool(context.TODO(), config.EmptyPoolCapacity),
		},
	}
	pools, err := api.GetPools(context.Background(), &emptypb.Empty{})
	require.NoError(t, err)
	assert.NotEmpty(t, pools)
	assert.NotEmpty(t, pools.AsMap())
	assert.Equal(t, pools.AsMap()[config.Default], map[string]interface{}{"cap": 0.0, "size": 0.0})
}

func TestPoolsWithEmptyPools(t *testing.T) {
	api := API{
		Pools: map[string]*pool.Pool{},
	}
	pools, err := api.GetPools(context.Background(), &emptypb.Empty{})
	require.NoError(t, err)
	assert.NotEmpty(t, pools)
	assert.Empty(t, pools.AsMap())
}

func TestGetProxies(t *testing.T) {
	clientConfig := &config.Client{
		Network: config.DefaultNetwork,
		Address: config.DefaultAddress,
	}
	client := network.NewClient(context.TODO(), clientConfig, zerolog.Logger{}, nil)
	require.NotNil(t, client)
	newPool := pool.NewPool(context.TODO(), 1)
	assert.Nil(t, newPool.Put(client.ID, client))

	proxy := network.NewProxy(
		context.TODO(),
		network.Proxy{
			AvailableConnections: newPool,
			HealthCheckPeriod:    config.DefaultHealthCheckPeriod,
			ClientConfig: &config.Client{
				Network: config.DefaultNetwork,
				Address: config.DefaultAddress,
			},
			Logger:        zerolog.Logger{},
			PluginTimeout: config.DefaultPluginTimeout,
		},
	)

	api := API{
		Proxies: map[string]*network.Proxy{
			config.Default: proxy,
		},
	}
	proxies, err := api.GetProxies(context.Background(), &emptypb.Empty{})
	require.NoError(t, err)
	assert.NotEmpty(t, proxies)
	assert.NotEmpty(t, proxies.AsMap())

	if defaultProxy, ok := proxies.AsMap()[config.Default].(map[string]interface{}); ok {
		assert.Equal(t, 1.0, defaultProxy["total"])
		assert.NotEmpty(t, defaultProxy["available"])
		assert.Empty(t, defaultProxy["busy"])
	} else {
		t.Errorf("proxies.default is not found or not a map")
	}

	proxy.Shutdown()
}

func TestGetServers(t *testing.T) {
	clientConfig := &config.Client{
		Network: config.DefaultNetwork,
		Address: config.DefaultAddress,
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
				Address: config.DefaultAddress,
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
			Address:      config.DefaultAddress,
			TickInterval: config.DefaultTickInterval,
			Options: network.Option{
				EnableTicker: false,
			},
			Proxy:            proxy,
			Logger:           zerolog.Logger{},
			PluginRegistry:   pluginRegistry,
			PluginTimeout:    config.DefaultPluginTimeout,
			HandshakeTimeout: config.DefaultHandshakeTimeout,
		},
	)

	api := API{
		Pools: map[string]*pool.Pool{
			config.Default: newPool,
		},
		Proxies: map[string]*network.Proxy{
			config.Default: proxy,
		},
		Servers: map[string]*network.Server{
			config.Default: server,
		},
	}
	servers, err := api.GetServers(context.Background(), &emptypb.Empty{})
	require.NoError(t, err)
	assert.NotEmpty(t, servers)
	assert.NotEmpty(t, servers.AsMap())

	if defaultServer, ok := servers.AsMap()[config.Default].(map[string]interface{}); ok {
		assert.Equal(t, config.DefaultNetwork, defaultServer["network"])
		assert.Equal(t, config.DefaultAddress, "localhost:5432")
		status, ok := defaultServer["status"].(float64)
		assert.True(t, ok)
		assert.Equal(t, config.Stopped, config.Status(status))
		tickInterval, ok := defaultServer["tickInterval"].(float64)
		assert.True(t, ok)
		assert.Equal(t, config.DefaultTickInterval.Nanoseconds(), int64(tickInterval))
	} else {
		t.Errorf("servers.default is not found or not a map")
	}
}
