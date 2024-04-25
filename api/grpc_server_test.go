package api

import (
	"context"
	"testing"

	v1 "github.com/gatewayd-io/gatewayd/api/v1"
	"github.com/gatewayd-io/gatewayd/config"
	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/protobuf/types/known/emptypb"
)

func Test_GRPC_Server(t *testing.T) {
	api := getAPIConfig()
	healthchecker := &HealthChecker{Servers: api.Servers}
	grpcServer := NewGRPCServer(GRPCServer{API: api, HealthChecker: healthchecker})
	assert.NotNil(t, grpcServer)

	go func(grpcServer *GRPCServer) {
		grpcServer.Start()
	}(grpcServer)

	grpcClient, err := grpc.NewClient(
		"localhost:19090", grpc.WithTransportCredentials(insecure.NewCredentials()))
	assert.Nil(t, err)
	defer grpcClient.Close()

	client := v1.NewGatewayDAdminAPIServiceClient(grpcClient)
	resp, err := client.Version(context.Background(), &emptypb.Empty{})
	assert.Nil(t, err)
	assert.NotNil(t, resp)
	assert.Equal(t, config.Version, resp.Version)
	assert.Equal(t, config.VersionInfo(), resp.VersionInfo)

	grpcServer.Shutdown(context.Background())
}
