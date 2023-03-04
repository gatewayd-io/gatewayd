package api

import (
	"context"
	"embed"
	"io/fs"
	"net"
	"net/http"

	v1 "github.com/gatewayd-io/gatewayd/api/v1"
	"github.com/gatewayd-io/gatewayd/config"
	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/reflection"
	"google.golang.org/protobuf/types/known/emptypb"
)

//go:embed v1/api.swagger.json
//go:embed v1/swagger-ui
var swaggerUI embed.FS

type API struct {
	v1.UnimplementedGatewayDAdminAPIServiceServer
}

func (a *API) Version(ctx context.Context, _ *emptypb.Empty) (*v1.VersionResponse, error) {
	return &v1.VersionResponse{
		Version: config.Version,
	}, nil
}

func RungRPCAPI() error {
	listener, err := net.Listen("tcp", ":9090")
	if err != nil {
		return err
	}

	grpcServer := grpc.NewServer()
	reflection.Register(grpcServer)
	v1.RegisterGatewayDAdminAPIServiceServer(grpcServer, &API{})
	return grpcServer.Serve(listener)
}

func RunHTTPAPI() error {
	ctx := context.Background()
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	// Register gRPC server endpoint
	// Note: Make sure the gRPC server is running properly and accessible
	rmux := runtime.NewServeMux()
	opts := []grpc.DialOption{grpc.WithTransportCredentials(insecure.NewCredentials())}
	err := v1.RegisterGatewayDAdminAPIServiceHandlerFromEndpoint(ctx, rmux, "localhost:9090", opts)
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
	return http.ListenAndServe(":8081", mux)
}
