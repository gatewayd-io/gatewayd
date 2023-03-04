package api

import (
	"context"
	"embed"
	"encoding/json"
	"io/fs"
	"net"
	"net/http"

	sdkPlugin "github.com/gatewayd-io/gatewayd-plugin-sdk/plugin"
	v1 "github.com/gatewayd-io/gatewayd/api/v1"
	"github.com/gatewayd-io/gatewayd/config"
	"github.com/gatewayd-io/gatewayd/network"
	"github.com/gatewayd-io/gatewayd/plugin"
	"github.com/gatewayd-io/gatewayd/pool"
	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/reflection"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/structpb"
)

//go:embed v1/api.swagger.json
//go:embed v1/swagger-ui
var swaggerUI embed.FS

type Options struct {
	GRPCNetwork string
	GRPCAddress string
	HTTPAddress string
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

func (a *API) Version(ctx context.Context, _ *emptypb.Empty) (*v1.VersionResponse, error) {
	return &v1.VersionResponse{
		Version:     config.Version,
		VersionInfo: config.VersionInfo(),
	}, nil
}

func (a *API) GetGlobalConfig(ctx context.Context, _ *emptypb.Empty) (*structpb.Struct, error) {
	jsonData, err := json.Marshal(a.Config.Global)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to marshal global config: %v", err)
	}
	var global map[string]interface{}
	err = json.Unmarshal(jsonData, &global)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to marshal global config: %v", err)
	}

	globalConfig, err := structpb.NewStruct(global)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to marshal global config: %v", err)
	}
	return globalConfig, nil
}

func (a *API) GetPluginConfig(ctx context.Context, _ *emptypb.Empty) (*structpb.Struct, error) {
	pluginConfig, err := structpb.NewStruct(a.Config.PluginKoanf.All())
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to marshal plugin config: %v", err)
	}
	return pluginConfig, nil
}

func (a *API) GetPlugins(context.Context, *emptypb.Empty) (*v1.PluginConfigs, error) {
	plugins := make([]*v1.PluginConfig, 0)
	a.PluginRegistry.ForEach(
		func(id sdkPlugin.Identifier, p *plugin.Plugin) {
			requires := make(map[string]string, 0)
			if p.Requires != nil {
				for _, r := range p.Requires {
					requires[r.Name] = r.Version
				}
			}
			plugins = append(plugins, &v1.PluginConfig{
				Id: &v1.PluginID{
					Name:      id.Name,
					Version:   id.Version,
					RemoteUrl: id.RemoteURL,
					Checksum:  id.Checksum,
				},
				Description: p.Description,
				Authors:     p.Authors,
				License:     p.License,
				ProjectUrl:  p.ProjectURL,
				Config:      p.Config,
				Hooks:       p.Hooks,
				Requires:    requires,
				Tags:        p.Tags,
				Categories:  p.Categories,
			})
		},
	)
	return &v1.PluginConfigs{
		Configs: plugins,
	}, nil
}

func (a *API) GetPools(ctx context.Context, _ *emptypb.Empty) (*structpb.Struct, error) {
	pools := make(map[string]interface{}, 0)
	for name, p := range a.Pools {
		pools[name] = map[string]interface{}{
			"cap":  p.Cap(),
			"size": p.Size(),
		}
	}
	poolsConfig, err := structpb.NewStruct(pools)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to marshal pools config: %v", err)
	}
	return poolsConfig, nil
}

func (a *API) GetProxies(ctx context.Context, _ *emptypb.Empty) (*structpb.Struct, error) {
	proxies := make(map[string]interface{}, 0)
	for name, p := range a.Proxies {
		available := make([]interface{}, 0)
		for _, c := range p.AvailableConnections() {
			available = append(available, c)
		}

		busy := make([]interface{}, 0)
		for _, c := range p.BusyConnections() {
			busy = append(busy, c)
		}

		proxies[name] = map[string]interface{}{
			"available": available,
			"busy":      busy,
			"total":     len(available) + len(busy),
		}
	}
	proxiesConfig, err := structpb.NewStruct(proxies)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to marshal proxies config: %v", err)
	}
	return proxiesConfig, nil
}

func (a *API) GetServers(ctx context.Context, _ *emptypb.Empty) (*structpb.Struct, error) {
	servers := make(map[string]interface{}, 0)
	for name, s := range a.Servers {
		servers[name] = map[string]interface{}{
			"network":      s.Network,
			"address":      s.Address,
			"status":       uint(s.Status),
			"softLimit":    s.SoftLimit,
			"hardLimit":    s.HardLimit,
			"tickInterval": s.TickInterval.Nanoseconds(),
		}
	}
	serversConfig, err := structpb.NewStruct(servers)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to marshal servers config: %v", err)
	}
	return serversConfig, nil
}

func StartGRPCAPI(api *API) error {
	listener, err := net.Listen(api.Options.GRPCNetwork, api.Options.GRPCAddress)
	if err != nil {
		return err
	}

	grpcServer := grpc.NewServer()
	reflection.Register(grpcServer)
	v1.RegisterGatewayDAdminAPIServiceServer(grpcServer, api)
	return grpcServer.Serve(listener)
}

func StartHTTPAPI(options *Options) error {
	ctx := context.Background()
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	// Register gRPC server endpoint
	// Note: Make sure the gRPC server is running properly and accessible
	rmux := runtime.NewServeMux()
	opts := []grpc.DialOption{grpc.WithTransportCredentials(insecure.NewCredentials())}
	err := v1.RegisterGatewayDAdminAPIServiceHandlerFromEndpoint(
		ctx, rmux, options.GRPCAddress, opts)
	if err != nil {
		return err
	}

	mux := http.NewServeMux()
	mux.Handle("/", rmux)
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	mux.HandleFunc("/version", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(config.Version))
	})
	mux.HandleFunc("/swagger.json", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		data, _ := swaggerUI.ReadFile("v1/api.swagger.json")
		w.Write(data)
	})

	fsys, err := fs.Sub(swaggerUI, "v1/swagger-ui")
	if err != nil {
		return err
	}
	mux.Handle("/swagger-ui/", http.StripPrefix("/swagger-ui/", http.FileServer(http.FS(fsys))))

	// Start HTTP server (and proxy calls to gRPC server endpoint)
	return http.ListenAndServe(options.HTTPAddress, mux)
}
