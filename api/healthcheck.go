package api

import (
	"context"

	"github.com/gatewayd-io/gatewayd/network"
	"github.com/gatewayd-io/gatewayd/raft"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/status"
)

type HealthChecker struct {
	grpc_health_v1.UnimplementedHealthServer

	Servers  map[string]*network.Server
	raftNode *raft.Node
}

func (h *HealthChecker) Check(
	ctx context.Context, req *grpc_health_v1.HealthCheckRequest,
) (*grpc_health_v1.HealthCheckResponse, error) {
	// Check if all servers are running
	if liveness(h.Servers, h.raftNode) {
		return &grpc_health_v1.HealthCheckResponse{
			Status: grpc_health_v1.HealthCheckResponse_SERVING,
		}, nil
	}

	return &grpc_health_v1.HealthCheckResponse{
		Status: grpc_health_v1.HealthCheckResponse_NOT_SERVING,
	}, nil
}

func (h *HealthChecker) Watch(
	*grpc_health_v1.HealthCheckRequest,
	grpc_health_v1.Health_WatchServer,
) error {
	return status.Error(codes.Unimplemented, "not implemented") //nolint:wrapcheck
}
