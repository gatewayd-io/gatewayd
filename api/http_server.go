package api

import (
	"context"
	"encoding/json"
	"errors"
	"io/fs"
	"net/http"
	"time"

	v1 "github.com/gatewayd-io/gatewayd/api/v1"
	"github.com/gatewayd-io/gatewayd/config"
	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"github.com/rs/zerolog"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

const headerReadTimeout = 10 * time.Second

type Healthz struct {
	Status string `json:"status"`
}

type HTTPServer struct {
	httpServer *http.Server
	options    *Options
	logger     zerolog.Logger
}

// NewHTTPServer creates a new HTTP server.
func NewHTTPServer(options *Options) *HTTPServer {
	httpServer := createHTTPAPI(options)
	return &HTTPServer{
		httpServer: httpServer,
		options:    options,
		logger:     options.Logger,
	}
}

// Start starts the HTTP server.
func (s *HTTPServer) Start() {
	// Start HTTP server (and proxy calls to gRPC server endpoint)
	if err := s.httpServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		s.options.Logger.Err(err).Msg("failed to start HTTP API")
	}
}

// Shutdown shuts down the HTTP server.
func (s *HTTPServer) Shutdown(ctx context.Context) {
	if err := s.httpServer.Shutdown(ctx); err != nil {
		s.logger.Err(err).Msg("failed to shutdown HTTP API")
	}
}

// CreateHTTPAPI creates a new HTTP API.
func createHTTPAPI(options *Options) *http.Server {
	// Register gRPC server endpoint
	// TODO: Make this configurable with TLS and Auth.
	rmux := runtime.NewServeMux()
	opts := []grpc.DialOption{grpc.WithTransportCredentials(insecure.NewCredentials())}
	err := v1.RegisterGatewayDAdminAPIServiceHandlerFromEndpoint(
		context.Background(), rmux, options.GRPCAddress, opts)
	if err != nil {
		options.Logger.Err(err).Msg("failed to start HTTP API")
	}

	mux := http.NewServeMux()
	mux.Handle("/", rmux)
	mux.HandleFunc("/healthz", func(writer http.ResponseWriter, _ *http.Request) {
		if liveness(options.Servers, options.RaftNode) {
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

	mux.HandleFunc("/version", func(writer http.ResponseWriter, _ *http.Request) {
		writer.WriteHeader(http.StatusOK)
		if _, err := writer.Write([]byte(config.Version)); err != nil {
			options.Logger.Err(err).Msg("failed to serve version")
			writer.WriteHeader(http.StatusInternalServerError)
		}
	})

	if IsSwaggerEmbedded() {
		mux.HandleFunc("/swagger.json", func(writer http.ResponseWriter, _ *http.Request) {
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
			return nil
		}
		mux.Handle("/swagger-ui/", http.StripPrefix("/swagger-ui/", http.FileServer(http.FS(fsys))))
	}

	server := &http.Server{
		Addr:              options.HTTPAddress,
		Handler:           mux,
		ReadHeaderTimeout: headerReadTimeout,
	}

	return server
}
