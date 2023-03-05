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
func StartHTTPAPI(options *Options) {
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
		options.Logger.Err(err).Msg("failed to start HTTP API")
	}

	mux := http.NewServeMux()
	mux.Handle("/", rmux)
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	mux.HandleFunc("/version", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		if _, err := w.Write([]byte(config.Version)); err != nil {
			options.Logger.Err(err).Msg("failed to serve version")
			w.WriteHeader(http.StatusInternalServerError)
		}
	})

	if IsSwaggerEmbedded() {
		mux.HandleFunc("/swagger.json", func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			data, _ := swaggerUI.ReadFile("v1/api.swagger.json")
			if _, err := w.Write(data); err != nil {
				options.Logger.Err(err).Msg("failed to serve swagger.json")
				w.WriteHeader(http.StatusInternalServerError)
			}
		})

		fsys, err := fs.Sub(swaggerUI, "v1/swagger-ui")
		if err != nil {
			options.Logger.Err(err).Msg("failed to serve swagger-ui")
			return
		}
		mux.Handle("/swagger-ui/", http.StripPrefix("/swagger-ui/", http.FileServer(http.FS(fsys))))
	}

	// Start HTTP server (and proxy calls to gRPC server endpoint)
	if err := http.ListenAndServe(options.HTTPAddress, mux); err != nil {
		options.Logger.Err(err).Msg("failed to start HTTP API")
	}
}
