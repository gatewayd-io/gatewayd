package api

import (
	"context"
	"io"
	"regexp"
	"testing"
	"time"

	sdkPlugin "github.com/gatewayd-io/gatewayd-plugin-sdk/plugin"
	pluginV1 "github.com/gatewayd-io/gatewayd-plugin-sdk/plugin/v1"
	"github.com/gatewayd-io/gatewayd/act"
	v1 "github.com/gatewayd-io/gatewayd/api/v1"
	"github.com/gatewayd-io/gatewayd/config"
	"github.com/gatewayd-io/gatewayd/network"
	"github.com/gatewayd-io/gatewayd/plugin"
	"github.com/gatewayd-io/gatewayd/pool"
	"github.com/gatewayd-io/gatewayd/raft"
	"github.com/gatewayd-io/gatewayd/testhelpers"
	hcRaft "github.com/hashicorp/raft"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
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
	conf := config.NewConfig(context.Background(),
		config.Config{GlobalConfigFile: "../gatewayd.yaml", PluginConfigFile: "../gatewayd_plugins.yaml"})
	gerr := conf.InitConfig(context.Background())
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
	conf := config.NewConfig(context.Background(),
		config.Config{GlobalConfigFile: "../gatewayd.yaml", PluginConfigFile: "../gatewayd_plugins.yaml"})
	gerr := conf.InitConfig(context.Background())
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
	if _, ok := globalconf["loggers"].(map[string]any)[config.Default]; !ok {
		t.Errorf("loggers.default is not found")
	}
}

func TestGetGlobalConfigWithNonExistingGroupName(t *testing.T) {
	// Load config from the default config file.
	conf := config.NewConfig(context.Background(),
		config.Config{GlobalConfigFile: "../gatewayd.yaml", PluginConfigFile: "../gatewayd_plugins.yaml"})
	gerr := conf.InitConfig(context.Background())
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
	conf := config.NewConfig(context.Background(),
		config.Config{GlobalConfigFile: "../gatewayd.yaml", PluginConfigFile: "../gatewayd_plugins.yaml"})
	gerr := conf.InitConfig(context.Background())
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
		context.Background(),
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
		context.Background(),
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
			config.Default: {config.DefaultConfigurationBlock: pool.NewPool(context.Background(), config.EmptyPoolCapacity)},
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
	client := network.NewClient(context.Background(), clientConfig, zerolog.Logger{}, nil)
	require.NotNil(t, client)
	newPool := pool.NewPool(context.Background(), 1)
	assert.Nil(t, newPool.Put(client.ID, client))

	proxy := network.NewProxy(
		context.Background(),
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
	client := network.NewClient(context.Background(), clientConfig, zerolog.Logger{}, nil)
	newPool := pool.NewPool(context.Background(), 1)
	require.NotNil(t, newPool)
	assert.Nil(t, newPool.Put(client.ID, client))

	proxy := network.NewProxy(
		context.Background(),
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
		context.Background(),
		plugin.Registry{
			ActRegistry:   actRegistry,
			Compatibility: config.Loose,
			Logger:        zerolog.Logger{},
			DevMode:       true,
		},
	)

	server := network.NewServer(
		context.Background(),
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

	if defaultServer, ok := servers.AsMap()[config.Default].(map[string]any); ok {
		assert.Equal(t, config.DefaultNetwork, defaultServer["network"])
		statusFloat, isStatusFloat := defaultServer["status"].(float64)
		assert.True(t, isStatusFloat, "status should be of type float64")
		status := config.Status(statusFloat)
		assert.Equal(t, config.Stopped, status)
		tickIntervalFloat, isTickIntervalFloat := defaultServer["tickInterval"].(float64)
		assert.True(t, isTickIntervalFloat, "tickInterval should be of type float64")
		assert.Equal(t, config.DefaultTickInterval.Nanoseconds(), int64(tickIntervalFloat))
		loadBalancerMap, isLoadBalancerMap := defaultServer["loadBalancer"].(map[string]any)
		assert.True(t, isLoadBalancerMap, "loadBalancer should be a map")
		assert.Equal(t, config.DefaultLoadBalancerStrategy, loadBalancerMap["strategy"])
	} else {
		t.Errorf("servers.default is not found or not a map")
	}
}

func TestRemovePeerAPI(t *testing.T) {
	tempDir := t.TempDir()

	// Configure three nodes
	nodeConfigs := []config.Raft{
		{
			NodeID:      "testRemovePeerNode1",
			Address:     "127.0.0.1:6879",
			IsBootstrap: true,
			Directory:   tempDir,
			GRPCAddress: "127.0.0.1:6880",
			Peers: []config.RaftPeer{
				{ID: "testRemovePeerNode2", Address: "127.0.0.1:6889", GRPCAddress: "127.0.0.1:6890"},
				{ID: "testRemovePeerNode3", Address: "127.0.0.1:6891", GRPCAddress: "127.0.0.1:6892"},
			},
		},
		{
			NodeID:      "testRemovePeerNode2",
			Address:     "127.0.0.1:6889",
			IsBootstrap: false,
			Directory:   tempDir,
			GRPCAddress: "127.0.0.1:6890",
			Peers: []config.RaftPeer{
				{ID: "testRemovePeerNode1", Address: "127.0.0.1:6879", GRPCAddress: "127.0.0.1:6880"},
				{ID: "testRemovePeerNode3", Address: "127.0.0.1:6891", GRPCAddress: "127.0.0.1:6892"},
			},
		},
		{
			NodeID:      "testRemovePeerNode3",
			Address:     "127.0.0.1:6891",
			IsBootstrap: false,
			Directory:   tempDir,
			GRPCAddress: "127.0.0.1:6892",
			Peers: []config.RaftPeer{
				{ID: "testRemovePeerNode1", Address: "127.0.0.1:6879", GRPCAddress: "127.0.0.1:6880"},
				{ID: "testRemovePeerNode2", Address: "127.0.0.1:6889", GRPCAddress: "127.0.0.1:6890"},
			},
		},
	}

	// Start all nodes
	nodes := make([]*raft.Node, len(nodeConfigs))
	defer func() {
		for _, node := range nodes {
			if node != nil {
				_ = node.Shutdown()
			}
		}
	}()

	// Initialize nodes
	for i, cfg := range nodeConfigs {
		node, err := raft.NewRaftNode(zerolog.New(io.Discard).With().Timestamp().Logger(), cfg)
		require.NoError(t, err)
		nodes[i] = node
	}

	// Wait for cluster to stabilize and leader election
	require.Eventually(t, func() bool {
		leaderCount := 0
		followerCount := 0
		for _, node := range nodes {
			state, _ := node.GetState()
			if state == hcRaft.Leader {
				leaderCount++
			}
			if state == hcRaft.Follower {
				followerCount++
			}
		}
		return leaderCount == 1 && followerCount == 2
	}, 10*time.Second, 100*time.Millisecond, "Failed to elect a leader")

	tests := []struct {
		name    string
		api     *API
		req     *v1.RemovePeerRequest
		wantErr bool
		errCode codes.Code
	}{
		{
			name: "successful peer removal",
			api: &API{
				ctx: context.Background(),
				Options: &Options{
					Logger:   zerolog.New(io.Discard),
					RaftNode: nodes[0],
				},
			},
			req:     &v1.RemovePeerRequest{PeerId: "testRemovePeerNode1"},
			wantErr: false,
		},
		{
			name: "raft node not initialized",
			api: &API{
				ctx: context.Background(),
				Options: &Options{
					Logger:   zerolog.New(io.Discard),
					RaftNode: nil,
				},
			},
			req:     &v1.RemovePeerRequest{PeerId: "test-peer"},
			wantErr: true,
			errCode: codes.Unavailable,
		},
		{
			name: "raft error during removal",
			api: &API{
				ctx: context.Background(),
				Options: &Options{
					Logger:   zerolog.New(io.Discard),
					RaftNode: nodes[0],
				},
			},
			req:     &v1.RemovePeerRequest{PeerId: "not-existing-peer"},
			wantErr: true,
			errCode: codes.Internal,
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			resp, err := testCase.api.RemovePeer(context.Background(), testCase.req)

			if testCase.wantErr {
				require.Error(t, err)
				st, ok := status.FromError(err)
				require.True(t, ok)
				assert.Equal(t, testCase.errCode, st.Code())
			} else {
				require.NoError(t, err)
				require.NotNil(t, resp)
				assert.True(t, resp.GetSuccess())
			}
		})
	}
}

func TestGetPeers(t *testing.T) {
	tempDir := t.TempDir()

	// Configure test nodes
	nodeConfig := config.Raft{
		NodeID:      "testGetPeersNode1",
		Address:     "127.0.0.1:7879",
		IsBootstrap: true,
		Directory:   tempDir,
		GRPCAddress: "127.0.0.1:7880",
		Peers:       []config.RaftPeer{},
	}

	// Initialize node
	node, err := raft.NewRaftNode(zerolog.New(io.Discard).With().Timestamp().Logger(), nodeConfig)
	require.NoError(t, err)
	defer func() {
		if node != nil {
			_ = node.Shutdown()
		}
	}()

	require.Eventually(t, func() bool {
		state, _ := node.GetState()
		return state == hcRaft.Leader
	}, 10*time.Second, 100*time.Millisecond, "Failed to elect a leader")

	tests := []struct {
		name    string
		api     *API
		wantErr bool
		errCode codes.Code
	}{
		{
			name: "successful get peers",
			api: &API{
				ctx: context.Background(),
				Options: &Options{
					Logger:   zerolog.New(io.Discard),
					RaftNode: node,
				},
			},
			wantErr: false,
		},
		{
			name: "raft node not initialized",
			api: &API{
				ctx: context.Background(),
				Options: &Options{
					Logger:   zerolog.New(io.Discard),
					RaftNode: nil,
				},
			},
			wantErr: true,
			errCode: codes.Unavailable,
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			peers, err := testCase.api.GetPeers(context.Background(), &emptypb.Empty{})

			if testCase.wantErr {
				require.Error(t, err)
				st, ok := status.FromError(err)
				require.True(t, ok)
				assert.Equal(t, testCase.errCode, st.Code())
			} else {
				require.NoError(t, err)
				require.NotNil(t, peers)

				// Verify the peers map structure
				peersMap := peers.AsMap()
				assert.NotEmpty(t, peersMap)

				// Check that configured peers exist in the response
				for _, peer := range nodeConfig.Peers {
					peerData, exists := peersMap[peer.ID].(map[string]any)
					assert.True(t, exists, "Peer %s should exist in response", peer.ID)
					if exists {
						assert.Equal(t, peer.ID, peerData["id"])
						assert.Equal(t, peer.Address, peerData["address"])
					}
				}
			}
		})
	}
}

func TestAddPeer(t *testing.T) {
	tempDir := t.TempDir()

	// Configure test node
	nodeConfig := config.Raft{
		NodeID:      "testAddPeerNode1",
		Address:     "127.0.0.1:8879",
		IsBootstrap: true,
		Directory:   tempDir,
		GRPCAddress: "127.0.0.1:8880",
		Peers:       []config.RaftPeer{},
	}

	// Initialize node
	node, err := raft.NewRaftNode(zerolog.New(io.Discard).With().Timestamp().Logger(), nodeConfig)
	require.NoError(t, err)
	defer func() {
		if node != nil {
			_ = node.Shutdown()
		}
	}()

	require.Eventually(t, func() bool {
		state, _ := node.GetState()
		return state == hcRaft.Leader
	}, 10*time.Second, 100*time.Millisecond, "Failed to elect a leader")

	tests := []struct {
		name    string
		api     *API
		req     *v1.AddPeerRequest
		wantErr bool
		errCode codes.Code
	}{
		{
			name: "successful peer addition",
			api: &API{
				ctx: context.Background(),
				Options: &Options{
					Logger:   zerolog.New(io.Discard),
					RaftNode: node,
				},
			},
			req: &v1.AddPeerRequest{
				PeerId:      "testAddPeerNode2",
				Address:     "127.0.0.1:8889",
				GrpcAddress: "127.0.0.1:8890",
			},
			wantErr: false,
		},
		{
			name: "raft node not initialized",
			api: &API{
				ctx: context.Background(),
				Options: &Options{
					Logger:   zerolog.New(io.Discard),
					RaftNode: nil,
				},
			},
			req: &v1.AddPeerRequest{
				PeerId:      "test-peer",
				Address:     "127.0.0.1:8891",
				GrpcAddress: "127.0.0.1:8892",
			},
			wantErr: true,
			errCode: codes.Unavailable,
		},
		{
			name: "missing peer id",
			api: &API{
				ctx: context.Background(),
				Options: &Options{
					Logger:   zerolog.New(io.Discard),
					RaftNode: node,
				},
			},
			req: &v1.AddPeerRequest{
				Address:     "127.0.0.1:8893",
				GrpcAddress: "127.0.0.1:8894",
			},
			wantErr: true,
			errCode: codes.InvalidArgument,
		},
		{
			name: "missing address",
			api: &API{
				ctx: context.Background(),
				Options: &Options{
					Logger:   zerolog.New(io.Discard),
					RaftNode: node,
				},
			},
			req: &v1.AddPeerRequest{
				PeerId:      "testAddPeerNode3",
				GrpcAddress: "127.0.0.1:8896",
			},
			wantErr: true,
			errCode: codes.InvalidArgument,
		},
		{
			name: "missing grpc address",
			api: &API{
				ctx: context.Background(),
				Options: &Options{
					Logger:   zerolog.New(io.Discard),
					RaftNode: node,
				},
			},
			req: &v1.AddPeerRequest{
				PeerId:  "testAddPeerNode4",
				Address: "127.0.0.1:8897",
			},
			wantErr: true,
			errCode: codes.InvalidArgument,
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			resp, err := testCase.api.AddPeer(context.Background(), testCase.req)

			if testCase.wantErr {
				require.Error(t, err)
				st, ok := status.FromError(err)
				require.True(t, ok)
				assert.Equal(t, testCase.errCode, st.Code())
			} else {
				require.NoError(t, err)
				require.NotNil(t, resp)
				assert.True(t, resp.GetSuccess())
			}
		})
	}
}
