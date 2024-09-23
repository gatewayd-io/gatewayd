package api

import (
	"context"
	"regexp"
	"testing"

	sdkPlugin "github.com/gatewayd-io/gatewayd-plugin-sdk/plugin"
	pluginV1 "github.com/gatewayd-io/gatewayd-plugin-sdk/plugin/v1"
	"github.com/gatewayd-io/gatewayd/act"
	v1 "github.com/gatewayd-io/gatewayd/api/v1"
	"github.com/gatewayd-io/gatewayd/config"
	"github.com/gatewayd-io/gatewayd/network"
	"github.com/gatewayd-io/gatewayd/plugin"
	"github.com/gatewayd-io/gatewayd/pool"
	"github.com/gatewayd-io/gatewayd/testhelpers"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/emptypb"
)

func TestGetVersion(t *testing.T) {
	api := API{ctx: context.Background()}
	version, err := api.Version(context.Background(), &emptypb.Empty{})
	require.NoError(t, err)
	assert.Regexp(t, regexp.MustCompile(`^\d+\.\d+\.\d+$`), version.GetVersion())
	assert.Regexp(t, regexp.MustCompile(`^GatewayD \d+\.\d+\.\d+ \(, go\d+\.\d+\.\d+, \w+/\w+\)$`), version.GetVersionInfo()) //nolint:lll
}

func TestGetGlobalConfig(t *testing.T) {
	// Load config from the default config file.
	conf := config.NewConfig(context.TODO(),
		config.Config{GlobalConfigFile: "../gatewayd.yaml", PluginConfigFile: "../gatewayd_plugins.yaml"})
	gerr := conf.InitConfig(context.TODO())
	require.Nil(t, gerr)
	assert.NotEmpty(t, conf.Global)

	api := API{
		Config: conf,
		ctx:    context.Background(),
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
	gerr := conf.InitConfig(context.TODO())
	require.Nil(t, gerr)
	assert.NotEmpty(t, conf.Global)

	api := API{
		Config: conf,
		ctx:    context.Background(),
	}
	defaultGroup := config.Default
	globalConfig, err := api.GetGlobalConfig(
		context.Background(),
		&v1.Group{GroupName: &defaultGroup},
	)
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
	if _, ok := globalconf["loggers"].(map[string]interface{})[config.Default]; !ok {
		t.Errorf("loggers.default is not found")
	}
}

func TestGetGlobalConfigWithNonExistingGroupName(t *testing.T) {
	// Load config from the default config file.
	conf := config.NewConfig(context.TODO(),
		config.Config{GlobalConfigFile: "../gatewayd.yaml", PluginConfigFile: "../gatewayd_plugins.yaml"})
	gerr := conf.InitConfig(context.TODO())
	require.Nil(t, gerr)
	assert.NotEmpty(t, conf.Global)

	api := API{
		Config: conf,
		ctx:    context.Background(),
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
	gerr := conf.InitConfig(context.TODO())
	require.Nil(t, gerr)
	assert.NotEmpty(t, conf.Global)

	api := API{
		Config: conf,
		ctx:    context.Background(),
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
		Requires: []sdkPlugin.Identifier{
			{
				Name:      "plugin1-name",
				Version:   "plugin1-version",
				RemoteURL: "plugin1-url",
				Checksum:  "plugin1-checksum",
			},
		},
		Hooks: []pluginV1.HookName{
			pluginV1.HookName_HOOK_NAME_ON_TRAFFIC_FROM_CLIENT,
		},
	})

	api := API{
		PluginRegistry: pluginRegistry,
		ctx:            context.Background(),
	}
	plugins, err := api.GetPlugins(context.Background(), &emptypb.Empty{})
	require.NoError(t, err)
	assert.NotEmpty(t, plugins)
	assert.NotEmpty(t, plugins.GetConfigs())
	assert.NotEmpty(t, plugins.GetConfigs()[0].GetRequires())
	assert.Equal(
		t,
		int32(pluginV1.HookName_HOOK_NAME_ON_TRAFFIC_FROM_CLIENT),
		plugins.GetConfigs()[0].GetHooks()[0])
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
		ctx:            context.Background(),
	}
	plugins, err := api.GetPlugins(context.Background(), &emptypb.Empty{})
	require.NoError(t, err)
	assert.NotEmpty(t, plugins)
	assert.Empty(t, plugins.GetConfigs())
}

func TestPools(t *testing.T) {
	api := API{
		Pools: map[string]map[string]*pool.Pool{
			config.Default: {config.DefaultConfigurationBlock: pool.NewPool(context.TODO(), config.EmptyPoolCapacity)},
		},
		ctx: context.Background(),
	}
	pools, err := api.GetPools(context.Background(), &emptypb.Empty{})
	require.NoError(t, err)
	assert.NotEmpty(t, pools)
	assert.NotEmpty(t, pools.AsMap())

	assert.Equal(t,
		map[string]any{
			config.DefaultConfigurationBlock: map[string]any{"cap": 0.0, "size": 0.0},
		},
		pools.AsMap()[config.Default])
}

func TestPoolsWithEmptyPools(t *testing.T) {
	api := API{
		Pools: map[string]map[string]*pool.Pool{},
		ctx:   context.Background(),
	}
	pools, err := api.GetPools(context.Background(), &emptypb.Empty{})
	require.NoError(t, err)
	assert.NotEmpty(t, pools)
	assert.Empty(t, pools.AsMap())
}

func TestGetProxies(t *testing.T) {
	postgresHostIP, postgresMappedPort := testhelpers.SetupPostgreSQLTestContainer(context.Background(), t)
	postgresAddress := postgresHostIP + ":" + postgresMappedPort.Port()

	clientConfig := &config.Client{
		Network: config.DefaultNetwork,
		Address: postgresAddress,
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
				Address: postgresAddress,
			},
			Logger:        zerolog.Logger{},
			PluginTimeout: config.DefaultPluginTimeout,
		},
	)

	api := API{
		Proxies: map[string]map[string]*network.Proxy{
			config.Default: {config.DefaultConfigurationBlock: proxy},
		},
		ctx: context.Background(),
	}
	proxies, err := api.GetProxies(context.Background(), &emptypb.Empty{})
	require.NoError(t, err)
	assert.NotEmpty(t, proxies)
	assert.NotEmpty(t, proxies.AsMap())

	if defaultProxies, ok := proxies.AsMap()[config.Default].(map[string]any); ok {
		if defaultProxy, ok := defaultProxies[config.DefaultConfigurationBlock].(map[string]any); ok {
			assert.Equal(t, 1.0, defaultProxy["total"])
			assert.NotEmpty(t, defaultProxy["available"])
			assert.Empty(t, defaultProxy["busy"])
		} else {
			t.Errorf("proxies.default.%s is not found or not a map", config.DefaultConfigurationBlock)
		}
	} else {
		t.Errorf("proxies.default is not found or not a map")
	}

	proxy.Shutdown()
}

func TestGetServers(t *testing.T) {
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
			Proxies:                  []network.IProxy{proxy},
			Logger:                   zerolog.Logger{},
			PluginRegistry:           pluginRegistry,
			PluginTimeout:            config.DefaultPluginTimeout,
			HandshakeTimeout:         config.DefaultHandshakeTimeout,
			LoadbalancerStrategyName: config.DefaultLoadBalancerStrategy,
		},
	)

	api := API{
		Pools: map[string]map[string]*pool.Pool{
			config.Default: {config.DefaultConfigurationBlock: newPool},
		},
		Proxies: map[string]map[string]*network.Proxy{
			config.Default: {config.DefaultConfigurationBlock: proxy},
		},
		Servers: map[string]*network.Server{
			config.Default: server,
		},
		ctx: context.Background(),
	}
	servers, err := api.GetServers(context.Background(), &emptypb.Empty{})
	require.NoError(t, err)
	assert.NotEmpty(t, servers)
	assert.NotEmpty(t, servers.AsMap())

	if defaultServer, ok := servers.AsMap()[config.Default].(map[string]interface{}); ok {
		assert.Equal(t, config.DefaultNetwork, defaultServer["network"])
		statusFloat, isStatusFloat := defaultServer["status"].(float64)
		assert.True(t, isStatusFloat, "status should be of type float64")
		status := config.Status(statusFloat)
		assert.Equal(t, config.Stopped, status)
		tickIntervalFloat, isTickIntervalFloat := defaultServer["tickInterval"].(float64)
		assert.True(t, isTickIntervalFloat, "tickInterval should be of type float64")
		assert.Equal(t, config.DefaultTickInterval.Nanoseconds(), int64(tickIntervalFloat))
		loadBalancerMap, isLoadBalancerMap := defaultServer["loadBalancer"].(map[string]interface{})
		assert.True(t, isLoadBalancerMap, "loadBalancer should be a map")
		assert.Equal(t, config.DefaultLoadBalancerStrategy, loadBalancerMap["strategy"])
	} else {
		t.Errorf("servers.default is not found or not a map")
	}
}
