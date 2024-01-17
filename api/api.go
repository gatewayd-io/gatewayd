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
	"github.com/rs/zerolog"
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
}

type API struct {
	v1.GatewayDAdminAPIServiceServer

	Options *Options

	Config         *config.Config
	PluginRegistry *plugin.Registry
	Pools          map[string]*pool.Pool
	Proxies        map[string]*network.Proxy
	Servers        map[string]*network.Server
}

// Version returns the version information of the GatewayD.
func (a *API) Version(context.Context, *emptypb.Empty) (*v1.VersionResponse, error) {
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
	var (
		jsonData []byte
		global   map[string]interface{}
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
		return nil, status.Errorf(codes.Internal, "failed to marshal global config: %v", err)
	}

	err = json.Unmarshal(jsonData, &global)
	if err != nil {
		metrics.APIRequestsErrors.WithLabelValues(
			"GET", "/v1/GatewayDPluginService/GetGlobalConfig", codes.Internal.String(),
		).Inc()
		return nil, status.Errorf(codes.Internal, "failed to marshal global config: %v", err)
	}

	globalConfig, err := structpb.NewStruct(global)
	if err != nil {
		metrics.APIRequestsErrors.WithLabelValues(
			"GET", "/v1/GatewayDPluginService/GetGlobalConfig", codes.Internal.String(),
		).Inc()
		return nil, status.Errorf(codes.Internal, "failed to marshal global config: %v", err)
	}

	metrics.APIRequests.WithLabelValues("GET", "/v1/GatewayDPluginService/GetGlobalConfig").Inc()
	return globalConfig, nil
}

// GetPluginConfig returns the plugin configuration of the GatewayD.
func (a *API) GetPluginConfig(context.Context, *emptypb.Empty) (*structpb.Struct, error) {
	pluginConfig, err := structpb.NewStruct(a.Config.PluginKoanf.All())
	if err != nil {
		metrics.APIRequestsErrors.WithLabelValues(
			"GET", "/v1/GatewayDPluginService/GetPluginConfig", codes.Internal.String(),
		).Inc()
		return nil, status.Errorf(codes.Internal, "failed to marshal plugin config: %v", err)
	}

	metrics.APIRequests.WithLabelValues("GET", "/v1/GatewayDPluginService/GetPluginConfig").Inc()
	return pluginConfig, nil
}

// GetPlugins returns the active plugin configuration of the GatewayD.
func (a *API) GetPlugins(context.Context, *emptypb.Empty) (*v1.PluginConfigs, error) {
	plugins := make([]*v1.PluginConfig, 0)
	a.PluginRegistry.ForEach(
		func(pluginID sdkPlugin.Identifier, plugIn *plugin.Plugin) {
			requires := make(map[string]string, 0)
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
	pools := make(map[string]interface{}, 0)
	for name, p := range a.Pools {
		pools[name] = map[string]interface{}{
			"cap":  p.Cap(),
			"size": p.Size(),
		}
	}

	poolsConfig, err := structpb.NewStruct(pools)
	if err != nil {
		metrics.APIRequestsErrors.WithLabelValues(
			"GET", "/v1/GatewayDPluginService/GetPools", codes.Internal.String(),
		).Inc()
		return nil, status.Errorf(codes.Internal, "failed to marshal pools config: %v", err)
	}

	metrics.APIRequests.WithLabelValues("GET", "/v1/GatewayDPluginService/GetPools").Inc()
	return poolsConfig, nil
}

// GetProxies returns the proxy configuration of the GatewayD.
func (a *API) GetProxies(context.Context, *emptypb.Empty) (*structpb.Struct, error) {
	proxies := make(map[string]interface{}, 0)
	for name, proxy := range a.Proxies {
		available := make([]interface{}, 0)
		for _, c := range proxy.AvailableConnections() {
			available = append(available, c)
		}

		busy := make([]interface{}, 0)
		for _, conn := range proxy.BusyConnections() {
			busy = append(busy, conn)
		}

		proxies[name] = map[string]interface{}{
			"available": available,
			"busy":      busy,
			"total":     len(available) + len(busy),
		}
	}

	proxiesConfig, err := structpb.NewStruct(proxies)
	if err != nil {
		metrics.APIRequestsErrors.WithLabelValues(
			"GET", "/v1/GatewayDPluginService/GetProxies", codes.Internal.String(),
		).Inc()
		return nil, status.Errorf(codes.Internal, "failed to marshal proxies config: %v", err)
	}

	metrics.APIRequests.WithLabelValues("GET", "/v1/GatewayDPluginService/GetProxies").Inc()
	return proxiesConfig, nil
}

// GetServers returns the server configuration of the GatewayD.
func (a *API) GetServers(context.Context, *emptypb.Empty) (*structpb.Struct, error) {
	servers := make(map[string]interface{}, 0)
	for name, server := range a.Servers {
		servers[name] = map[string]interface{}{
			"network":      server.Network,
			"address":      server.Address,
			"status":       uint(server.Status),
			"tickInterval": server.TickInterval.Nanoseconds(),
		}
	}

	serversConfig, err := structpb.NewStruct(servers)
	if err != nil {
		metrics.APIRequestsErrors.WithLabelValues(
			"GET", "/v1/GatewayDPluginService/GetServers", codes.Internal.String(),
		).Inc()
		return nil, status.Errorf(codes.Internal, "failed to marshal servers config: %v", err)
	}

	metrics.APIRequests.WithLabelValues("GET", "/v1/GatewayDPluginService/GetServers").Inc()
	return serversConfig, nil
}
