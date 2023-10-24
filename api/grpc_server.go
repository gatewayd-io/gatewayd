package api

import (
	"net"

	v1 "github.com/gatewayd-io/gatewayd/api/v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/reflection"
)

// StartGRPCAPI starts the gRPC API.
func StartGRPCAPI(api *API, healthchecker *HealthChecker) {
	listener, err := net.Listen(api.Options.GRPCNetwork, api.Options.GRPCAddress)
	if err != nil {
		api.Options.Logger.Err(err).Msg("failed to start gRPC API")
	}

	grpcServer := grpc.NewServer()
	reflection.Register(grpcServer)
	v1.RegisterGatewayDAdminAPIServiceServer(grpcServer, api)
	grpc_health_v1.RegisterHealthServer(grpcServer, healthchecker)
	if err := grpcServer.Serve(listener); err != nil {
		api.Options.Logger.Err(err).Msg("failed to start gRPC API")
	}
}
