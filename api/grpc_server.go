package api

import (
	"context"
	"errors"
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
func NewGRPCServer(ctx context.Context, server GRPCServer) *GRPCServer {
	grpcServer, listener := createGRPCAPI(server.API, server.HealthChecker)
	if grpcServer == nil || listener == nil {
		server.API.Options.Logger.Error().Msg("Failed to create gRPC API server and listener")
		return nil
	}

	if ctx != nil {
		server.API.ctx = ctx
	} else {
		server.API.ctx = context.Background()
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
	if err := s.grpcServer.Serve(s.listener); err != nil && !errors.Is(err, net.ErrClosed) {
		s.API.Options.Logger.Err(err).Msg("failed to start gRPC API")
	}
}

// Shutdown shuts down the gRPC server.
func (s *GRPCServer) Shutdown(_ context.Context) {
	s.listener.Close()
	s.grpcServer.Stop()
}

// createGRPCAPI creates a new gRPC API server and listener.
func createGRPCAPI(api *API, healthchecker *HealthChecker) (*grpc.Server, net.Listener) {
	listener, err := net.Listen(api.Options.GRPCNetwork, api.Options.GRPCAddress)
	if err != nil && !errors.Is(err, net.ErrClosed) {
		api.Options.Logger.Err(err).Msg("failed to start gRPC API")
		return nil, nil
	}

	grpcServer := grpc.NewServer()
	reflection.Register(grpcServer)
	v1.RegisterGatewayDAdminAPIServiceServer(grpcServer, api)
	grpc_health_v1.RegisterHealthServer(grpcServer, healthchecker)

	return grpcServer, listener
}
