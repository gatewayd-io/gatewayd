package api

import (
	"context"
	"encoding/json"
	"io/fs"
	"net/http"

	v1 "github.com/gatewayd-io/gatewayd/api/v1"
	"github.com/gatewayd-io/gatewayd/config"
	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type Healthz struct {
	Status string `json:"status"`
}

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
	mux.HandleFunc("/healthz", func(writer http.ResponseWriter, r *http.Request) {
		if liveness(options.Servers) {
			writer.Header().Set("Content-Type", "application/json")
			writer.WriteHeader(http.StatusOK)
			if err := json.NewEncoder(writer).Encode(Healthz{Status: "SERVING"}); err != nil {
				options.Logger.Err(err).Msg("failed to serve healthcheck")
				writer.WriteHeader(http.StatusInternalServerError)
			}
		} else {
			writer.Header().Set("Content-Type", "application/json")
			writer.WriteHeader(http.StatusServiceUnavailable)
			if err := json.NewEncoder(writer).Encode(Healthz{Status: "NOT_SERVING"}); err != nil {
				options.Logger.Err(err).Msg("failed to serve healthcheck")
			}
		}
	})

	mux.HandleFunc("/version", func(writer http.ResponseWriter, r *http.Request) {
		writer.WriteHeader(http.StatusOK)
		if _, err := writer.Write([]byte(config.Version)); err != nil {
			options.Logger.Err(err).Msg("failed to serve version")
			writer.WriteHeader(http.StatusInternalServerError)
		}
	})

	if IsSwaggerEmbedded() {
		mux.HandleFunc("/swagger.json", func(writer http.ResponseWriter, r *http.Request) {
			writer.WriteHeader(http.StatusOK)
			data, _ := swaggerUI.ReadFile("v1/api.swagger.json")
			if _, err := writer.Write(data); err != nil {
				options.Logger.Err(err).Msg("failed to serve swagger.json")
				writer.WriteHeader(http.StatusInternalServerError)
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
	if err := http.ListenAndServe(options.HTTPAddress, mux); err != nil { //nolint:gosec
		options.Logger.Err(err).Msg("failed to start HTTP API")
	}
}
