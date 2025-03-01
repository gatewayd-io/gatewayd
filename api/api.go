//go:build !embed

package api

import (
	"context"
	"encoding/json"

	sdkPlugin "github.com/gatewayd-io/gatewayd-plugin-sdk/plugin"
	v1 "github.com/gatewayd-io/gatewayd/api/v1"
	"github.com/gatewayd-io/gatewayd/config"
	"github.com/gatewayd-io/gatewayd/metrics"
	"github.com/gatewayd-io/gatewayd/network"
	"github.com/gatewayd-io/gatewayd/plugin"
	"github.com/gatewayd-io/gatewayd/pool"
	"github.com/gatewayd-io/gatewayd/raft"
	hcraft "github.com/hashicorp/raft"
	"github.com/rs/zerolog"
	"go.opentelemetry.io/otel"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/structpb"
)

type Options struct {
	Logger      zerolog.Logger
	GRPCNetwork string
	GRPCAddress string
	HTTPAddress string
	Servers     map[string]*network.Server
	RaftNode    *raft.Node
}

type API struct {
	v1.GatewayDAdminAPIServiceServer

	// Tracer context.
	ctx context.Context //nolint:containedctx

	Options        *Options
	Config         *config.Config
	PluginRegistry *plugin.Registry
	Pools          map[string]map[string]*pool.Pool
	Proxies        map[string]map[string]*network.Proxy
	Servers        map[string]*network.Server
}

// Version returns the version information of the GatewayD.
func (a *API) Version(context.Context, *emptypb.Empty) (*v1.VersionResponse, error) {
	_, span := otel.Tracer(config.TracerName).Start(a.ctx, "Get Version")
	defer span.End()

	metrics.APIRequests.WithLabelValues("GET", "/v1/GatewayDPluginService/Version").Inc()
	return &v1.VersionResponse{
		Version:     config.Version,
		VersionInfo: config.VersionInfo(),
	}, nil
}

// GetGlobalConfig returns the global configuration of the GatewayD.
//
//nolint:wrapcheck
func (a *API) GetGlobalConfig(_ context.Context, group *v1.Group) (*structpb.Struct, error) {
	_, span := otel.Tracer(config.TracerName).Start(a.ctx, "Getting Global Config")
	defer span.End()

	var (
		jsonData []byte
		global   map[string]any
		err      error
	)

	if group.GetGroupName() == "" {
		jsonData, err = json.Marshal(a.Config.Global)
	} else {
		configGroup := a.Config.Global.Filter(group.GetGroupName())
		if configGroup == nil {
			metrics.APIRequestsErrors.WithLabelValues(
				"GET", "/v1/GatewayDPluginService/GetGlobalConfig", codes.NotFound.String(),
			).Inc()
			return nil, status.Error(codes.NotFound, "group not found")
		}
		jsonData, err = json.Marshal(configGroup)
	}
	if err != nil {
		metrics.APIRequestsErrors.WithLabelValues(
			"GET", "/v1/GatewayDPluginService/GetGlobalConfig", codes.Internal.String(),
		).Inc()
		a.Options.Logger.Err(err).Msg("GetGroupName is nil")
		span.RecordError(err)
		return nil, status.Errorf(codes.Internal, "failed to marshal global config: %v", err)
	}

	err = json.Unmarshal(jsonData, &global)
	if err != nil {
		metrics.APIRequestsErrors.WithLabelValues(
			"GET", "/v1/GatewayDPluginService/GetGlobalConfig", codes.Internal.String(),
		).Inc()
		a.Options.Logger.Err(err).Msg("Failed to marshal global config")
		span.RecordError(err)
		return nil, status.Errorf(codes.Internal, "failed to marshal global config: %v", err)
	}

	globalConfig, err := structpb.NewStruct(global)
	if err != nil {
		metrics.APIRequestsErrors.WithLabelValues(
			"GET", "/v1/GatewayDPluginService/GetGlobalConfig", codes.Internal.String(),
		).Inc()
		a.Options.Logger.Err(err).Msg("Failed to marshal global config")
		span.RecordError(err)
		return nil, status.Errorf(codes.Internal, "failed to marshal global config: %v", err)
	}

	metrics.APIRequests.WithLabelValues("GET", "/v1/GatewayDPluginService/GetGlobalConfig").Inc()
	return globalConfig, nil
}

// GetPluginConfig returns the plugin configuration of the GatewayD.
func (a *API) GetPluginConfig(context.Context, *emptypb.Empty) (*structpb.Struct, error) {
	_, span := otel.Tracer(config.TracerName).Start(a.ctx, "Get GetPlugin Config")
	defer span.End()

	jsonData, err := json.Marshal(a.Config.Plugin)
	if err != nil {
		metrics.APIRequestsErrors.WithLabelValues(
			"GET", "/v1/GatewayDPluginService/GetPluginConfig", codes.Internal.String(),
		).Inc()
		a.Options.Logger.Err(err).Msg("Failed to marshal plugin config")
		span.RecordError(err)
		return nil, status.Errorf(codes.Internal, "failed to marshal plugin config: %v", err)
	}

	var pluginConfigMap map[string]any

	err = json.Unmarshal(jsonData, &pluginConfigMap)
	if err != nil {
		metrics.APIRequestsErrors.WithLabelValues(
			"GET", "/v1/GatewayDPluginService/GetPluginConfig", codes.Internal.String(),
		).Inc()
		a.Options.Logger.Err(err).Msg("Failed to unmarshal plugin config")
		span.RecordError(err)
		return nil, status.Errorf(codes.Internal, "failed to unmarshal plugin config: %v", err)
	}

	pluginConfig, err := structpb.NewStruct(pluginConfigMap)
	if err != nil {
		metrics.APIRequestsErrors.WithLabelValues(
			"GET", "/v1/GatewayDPluginService/GetPluginConfig", codes.Internal.String(),
		).Inc()
		a.Options.Logger.Err(err).Msg("Failed to marshal plugin config")
		span.RecordError(err)
		return nil, status.Errorf(codes.Internal, "failed to marshal plugin config: %v", err)
	}

	metrics.APIRequests.WithLabelValues("GET", "/v1/GatewayDPluginService/GetPluginConfig").Inc()
	return pluginConfig, nil
}

// GetPlugins returns the active plugin configuration of the GatewayD.
func (a *API) GetPlugins(context.Context, *emptypb.Empty) (*v1.PluginConfigs, error) {
	_, span := otel.Tracer(config.TracerName).Start(a.ctx, "Get Plugins")
	defer span.End()

	plugins := make([]*v1.PluginConfig, 0)
	a.PluginRegistry.ForEach(
		func(pluginID sdkPlugin.Identifier, plugIn *plugin.Plugin) {
			requires := make(map[string]string)
			if plugIn.Requires != nil {
				for _, r := range plugIn.Requires {
					requires[r.Name] = r.Version
				}
			}

			hooks := make([]int32, 0)
			for _, hook := range plugIn.Hooks {
				hooks = append(hooks, int32(hook.Number()))
			}

			plugins = append(plugins, &v1.PluginConfig{
				Id: &v1.PluginID{
					Name:      pluginID.Name,
					Version:   pluginID.Version,
					RemoteUrl: pluginID.RemoteURL,
					Checksum:  pluginID.Checksum,
				},
				Description: plugIn.Description,
				Authors:     plugIn.Authors,
				License:     plugIn.License,
				ProjectUrl:  plugIn.ProjectURL,
				Config:      plugIn.Config,
				Hooks:       hooks,
				Requires:    requires,
				Tags:        plugIn.Tags,
				Categories:  plugIn.Categories,
			})
		},
	)

	metrics.APIRequests.WithLabelValues("GET", "/v1/GatewayDPluginService/GetPlugins").Inc()
	return &v1.PluginConfigs{
		Configs: plugins,
	}, nil
}

// GetPools returns the pool configuration of the GatewayD.
func (a *API) GetPools(context.Context, *emptypb.Empty) (*structpb.Struct, error) {
	_, span := otel.Tracer(config.TracerName).Start(a.ctx, "Get Pools")
	defer span.End()

	pools := make(map[string]any)

	for configGroupName, configGroupPools := range a.Pools {
		groupPools := make(map[string]any)
		for name, p := range configGroupPools {
			groupPools[name] = map[string]any{
				"cap":  p.Cap(),
				"size": p.Size(),
			}
		}
		pools[configGroupName] = groupPools
	}

	poolsConfig, err := structpb.NewStruct(pools)
	if err != nil {
		metrics.APIRequestsErrors.WithLabelValues(
			"GET", "/v1/GatewayDPluginService/GetPools", codes.Internal.String(),
		).Inc()
		a.Options.Logger.Err(err).Msg("Failed to marshal pools config")
		return nil, status.Errorf(codes.Internal, "failed to marshal pools config: %v", err)
	}

	metrics.APIRequests.WithLabelValues("GET", "/v1/GatewayDPluginService/GetPools").Inc()
	return poolsConfig, nil
}

// GetProxies returns the proxy configuration of the GatewayD.
func (a *API) GetProxies(context.Context, *emptypb.Empty) (*structpb.Struct, error) {
	_, span := otel.Tracer(config.TracerName).Start(a.ctx, "Get Proxies")
	defer span.End()

	// Create a new map to hold the flattened proxies data
	proxies := make(map[string]any)

	for configGroupName, configGroupProxies := range a.Proxies {
		// Create a map for each configuration group
		groupProxies := make(map[string]any)
		for name, proxy := range configGroupProxies {
			available := make([]any, 0)
			for _, c := range proxy.AvailableConnectionsString() {
				available = append(available, c)
			}

			busy := make([]any, 0)
			for _, conn := range proxy.BusyConnectionsString() {
				busy = append(busy, conn)
			}

			groupProxies[name] = map[string]any{
				"available": available,
				"busy":      busy,
				"total":     len(available) + len(busy),
			}
		}

		proxies[configGroupName] = groupProxies
	}

	proxiesConfig, err := structpb.NewStruct(proxies)
	if err != nil {
		metrics.APIRequestsErrors.WithLabelValues(
			"GET", "/v1/GatewayDPluginService/GetProxies", codes.Internal.String(),
		).Inc()
		a.Options.Logger.Err(err).Msg("Failed to marshal proxies config")
		return nil, status.Errorf(codes.Internal, "failed to marshal proxies config: %v", err)
	}

	metrics.APIRequests.WithLabelValues("GET", "/v1/GatewayDPluginService/GetProxies").Inc()
	return proxiesConfig, nil
}

// GetServers returns the server configuration of the GatewayD.
func (a *API) GetServers(context.Context, *emptypb.Empty) (*structpb.Struct, error) {
	_, span := otel.Tracer(config.TracerName).Start(a.ctx, "Get Servers")
	defer span.End()

	servers := make(map[string]any)
	for name, server := range a.Servers {
		servers[name] = map[string]any{
			"network":      server.Network,
			"address":      server.Address,
			"status":       uint(server.Status),
			"tickInterval": server.TickInterval.Nanoseconds(),
			"loadBalancer": map[string]any{"strategy": server.LoadbalancerStrategyName},
			"isTLSEnabled": server.IsTLSEnabled(),
		}
	}

	serversConfig, err := structpb.NewStruct(servers)
	if err != nil {
		metrics.APIRequestsErrors.WithLabelValues(
			"GET", "/v1/GatewayDPluginService/GetServers", codes.Internal.String(),
		).Inc()
		a.Options.Logger.Err(err).Msg("Failed to marshal servers config")
		return nil, status.Errorf(codes.Internal, "failed to marshal servers config: %v", err)
	}

	metrics.APIRequests.WithLabelValues("GET", "/v1/GatewayDPluginService/GetServers").Inc()
	return serversConfig, nil
}

// GetPeers returns the raft peers configuration of the GatewayD.
func (a *API) GetPeers(context.Context, *emptypb.Empty) (*structpb.Struct, error) {
	_, span := otel.Tracer(config.TracerName).Start(a.ctx, "Get Peers")
	defer span.End()

	if a.Options.RaftNode == nil {
		return nil, status.Errorf(codes.Unavailable, "raft node not initialized")
	}

	peers := a.Options.RaftNode.GetPeers()
	peerMap := make(map[string]any)

	// Get current leader ID for comparison
	_, leaderID := a.Options.RaftNode.GetState()

	for _, peer := range peers {
		// Determine peer status
		var status string
		switch {
		case string(peer.ID) == string(leaderID):
			status = "Leader"
		case peer.Suffrage == hcraft.Voter:
			status = "Follower"
		case peer.Suffrage == hcraft.Nonvoter:
			status = "NonVoter"
		default:
			status = "Unknown"
		}

		peerMap[string(peer.ID)] = map[string]any{
			"id":       string(peer.ID),
			"address":  string(peer.Address),
			"status":   status,
			"suffrage": peer.Suffrage.String(),
		}
	}

	raftPeers, err := structpb.NewStruct(peerMap)
	if err != nil {
		metrics.APIRequestsErrors.WithLabelValues(
			"GET", "/v1/raft/peers", codes.Internal.String(),
		).Inc()
		a.Options.Logger.Err(err).Msg("Failed to marshal peers config")
		return nil, status.Errorf(codes.Internal, "failed to marshal peers config: %v", err)
	}

	metrics.APIRequests.WithLabelValues("GET", "/v1/raft/peers").Inc()
	return raftPeers, nil
}

// AddPeer adds a new peer to the raft cluster.
func (a *API) AddPeer(ctx context.Context, req *v1.AddPeerRequest) (*v1.AddPeerResponse, error) {
	_, span := otel.Tracer(config.TracerName).Start(ctx, "Add Peer")
	defer span.End()

	if a.Options.RaftNode == nil {
		return nil, status.Errorf(codes.Unavailable, "AddPeer: raft node not initialized")
	}

	if req.GetPeerId() == "" || req.GetAddress() == "" || req.GetGrpcAddress() == "" {
		return nil, status.Errorf(codes.InvalidArgument, "AddPeer: peer id, address, and grpc address are required")
	}

	err := a.Options.RaftNode.AddPeer(ctx, req.GetPeerId(), req.GetAddress(), req.GetGrpcAddress())
	if err != nil {
		metrics.APIRequestsErrors.WithLabelValues(
			"POST", "/v1/raft/peers", codes.Internal.String(),
		).Inc()
		a.Options.Logger.Err(err).Msg("Failed to add peer")
		return nil, status.Errorf(codes.Internal, "AddPeer: failed to add peer: %v", err)
	}

	metrics.APIRequests.WithLabelValues("POST", "/v1/raft/peers").Inc()
	return &v1.AddPeerResponse{Success: true}, nil
}

// RemovePeer removes a peer from the raft cluster.
func (a *API) RemovePeer(ctx context.Context, req *v1.RemovePeerRequest) (*v1.RemovePeerResponse, error) {
	_, span := otel.Tracer(config.TracerName).Start(ctx, "Remove Peer")
	defer span.End()

	if a.Options.RaftNode == nil {
		return nil, status.Errorf(codes.Unavailable, "RemovePeer: raft node not initialized")
	}

	if req.GetPeerId() == "" {
		return nil, status.Errorf(codes.InvalidArgument, "RemovePeer: peer id is required")
	}

	err := a.Options.RaftNode.RemovePeer(ctx, req.GetPeerId())
	if err != nil {
		metrics.APIRequestsErrors.WithLabelValues(
			"DELETE", "/v1/raft/peers", codes.Internal.String(),
		).Inc()
		a.Options.Logger.Err(err).Msg("Failed to remove peer")
		return nil, status.Errorf(codes.Internal, "RemovePeer: failed to remove peer: %v", err)
	}

	metrics.APIRequests.WithLabelValues("DELETE", "/v1/raft/peers").Inc()
	return &v1.RemovePeerResponse{Success: true}, nil
}
