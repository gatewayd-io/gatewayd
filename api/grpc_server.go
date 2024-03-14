package api

import (
	"context"
	"net"

	v1 "github.com/gatewayd-io/gatewayd/api/v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/reflection"
)

type GRPCServer struct {
	api        *API
	grpcServer *grpc.Server
	listener   net.Listener
}

// NewGRPCServer creates a new gRPC server.
func NewGRPCServer(api *API, healthchecker *HealthChecker) *GRPCServer {
	grpcServer, listener := createGRPCAPI(api, healthchecker)
	return &GRPCServer{
		api:        api,
		grpcServer: grpcServer,
		listener:   listener,
	}
}

// Start starts the gRPC server.
func (s *GRPCServer) Start() {
	s.start(s.api, s.grpcServer, s.listener)
}

// Shutdown shuts down the gRPC server.
func (s *GRPCServer) Shutdown(_ context.Context) {
	s.shutdown(s.grpcServer)
}

// createGRPCAPI creates a new gRPC API server and listener.
func createGRPCAPI(api *API, healthchecker *HealthChecker) (*grpc.Server, net.Listener) {
	listener, err := net.Listen(api.Options.GRPCNetwork, api.Options.GRPCAddress)
	if err != nil {
		api.Options.Logger.Err(err).Msg("failed to start gRPC API")
	}

	grpcServer := grpc.NewServer()
	reflection.Register(grpcServer)
	v1.RegisterGatewayDAdminAPIServiceServer(grpcServer, api)
	grpc_health_v1.RegisterHealthServer(grpcServer, healthchecker)

	return grpcServer, listener
}

// start starts the gRPC API.
func (s *GRPCServer) start(api *API, grpcServer *grpc.Server, listener net.Listener) {
	if err := grpcServer.Serve(listener); err != nil {
		api.Options.Logger.Err(err).Msg("failed to start gRPC API")
	}
}

// shutdown shuts down the gRPC API.
func (s *GRPCServer) shutdown(grpcServer *grpc.Server) {
	grpcServer.GracefulStop()
}
