package api

import (
	"net"

	v1 "github.com/gatewayd-io/gatewayd/api/v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

// StartGRPCAPI starts the gRPC API.
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
