package api

import (
	"context"
	"io/fs"
	"net/http"

	v1 "github.com/gatewayd-io/gatewayd/api/v1"
	"github.com/gatewayd-io/gatewayd/config"
	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// StartHTTPAPI starts the HTTP API.
func StartHTTPAPI(options *Options) error {
	ctx := context.Background()
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	// Register gRPC server endpoint
	// TODO: Make this configurable with TLS and Auth.
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

	if IsSwaggerEmbedded() {
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
	}

	// Start HTTP server (and proxy calls to gRPC server endpoint)
	return http.ListenAndServe(options.HTTPAddress, mux)
}
