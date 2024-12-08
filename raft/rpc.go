package raft

import (
	"context"
	"fmt"
	"time"

	pb "github.com/gatewayd-io/gatewayd/raft/proto"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type rpcServer struct {
	pb.UnimplementedRaftServiceServer
	node *Node
}

func (s *rpcServer) ForwardApply(_ context.Context, req *pb.ApplyRequest) (*pb.ApplyResponse, error) {
	timeout := time.Duration(req.GetTimeoutMs()) * time.Millisecond

	err := s.node.applyInternal(req.GetData(), timeout)
	if err != nil {
		return &pb.ApplyResponse{
			Success: false,
			Error:   err.Error(),
		}, err
	}

	return &pb.ApplyResponse{
		Success: true,
	}, nil
}

type rpcClient struct {
	clients map[string]pb.RaftServiceClient
	conns   map[string]*grpc.ClientConn
	node    *Node
}

func newRPCClient(node *Node) *rpcClient {
	return &rpcClient{
		clients: make(map[string]pb.RaftServiceClient),
		conns:   make(map[string]*grpc.ClientConn),
		node:    node,
	}
}

func (c *rpcClient) getClient(address string) (pb.RaftServiceClient, error) {
	if client, ok := c.clients[address]; ok {
		return client, nil
	}

	conn, err := grpc.NewClient(
		address,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to %s: %w", address, err)
	}

	client := pb.NewRaftServiceClient(conn)
	c.clients[address] = client
	c.conns[address] = conn
	return client, nil
}

func (c *rpcClient) close() {
	for _, conn := range c.conns {
		conn.Close()
	}
}
