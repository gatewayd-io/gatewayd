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
	API        *API
	grpcServer *grpc.Server
	listener   net.Listener
	*HealthChecker
}

// NewGRPCServer creates a new gRPC server.
func NewGRPCServer(server GRPCServer) *GRPCServer {
	grpcServer, listener := createGRPCAPI(server.API, server.HealthChecker)
	if grpcServer == nil || listener == nil {
		server.API.Options.Logger.Error().Msg("Failed to create gRPC API server and listener")
		return nil
	}

	return &GRPCServer{
		API:           server.API,
		grpcServer:    grpcServer,
		listener:      listener,
		HealthChecker: server.HealthChecker,
	}
}

// Start starts the gRPC server.
func (s *GRPCServer) Start() {
	s.start(s.API, s.grpcServer, s.listener)
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
		return nil, nil
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
