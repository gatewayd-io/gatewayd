package raft

import (
	"context"
	"fmt"
	"time"

	pb "github.com/gatewayd-io/gatewayd/raft/proto"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// rpcServer implements the RaftServiceServer interface and handles incoming RPC requests.
type rpcServer struct {
	pb.UnimplementedRaftServiceServer
	node *Node
}

// ForwardApply processes an ApplyRequest by applying the data to the node with a specified timeout.
func (s *rpcServer) ForwardApply(
	ctx context.Context,
	req *pb.ForwardApplyRequest,
) (*pb.ForwardApplyResponse, error) {
	timeout := time.Duration(req.GetTimeoutMs()) * time.Millisecond

	err := s.node.applyInternal(ctx, req.GetData(), timeout)
	if err != nil {
		return &pb.ForwardApplyResponse{
			Success: false,
			Error:   err.Error(),
		}, err
	}

	return &pb.ForwardApplyResponse{
		Success: true,
	}, nil
}

// rpcClient manages gRPC clients and connections for communicating with other nodes.
type rpcClient struct {
	clients map[string]pb.RaftServiceClient
	conns   map[string]*grpc.ClientConn
	node    *Node
}

// newRPCClient creates a new rpcClient for the given node.
func newRPCClient(node *Node) *rpcClient {
	return &rpcClient{
		clients: make(map[string]pb.RaftServiceClient),
		conns:   make(map[string]*grpc.ClientConn),
		node:    node,
	}
}

// getClient retrieves or establishes a gRPC client connection to the specified address.
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

// close terminates all gRPC client connections managed by the rpcClient.
func (c *rpcClient) close() {
	for _, conn := range c.conns {
		conn.Close()
	}
}

// AddPeer handles the AddPeer gRPC request.
func (s *rpcServer) AddPeer(ctx context.Context, req *pb.AddPeerRequest) (*pb.AddPeerResponse, error) {
	err := s.node.AddPeer(ctx, req.GetPeerId(), req.GetPeerAddress(), req.GetGrpcAddress())
	if err != nil {
		return &pb.AddPeerResponse{
			Success: false,
			Error:   err.Error(),
		}, err
	}
	return &pb.AddPeerResponse{
		Success: true,
	}, nil
}

// RemovePeer handles the RemovePeer gRPC request.
func (s *rpcServer) RemovePeer(ctx context.Context, req *pb.RemovePeerRequest) (*pb.RemovePeerResponse, error) {
	err := s.node.RemovePeer(ctx, req.GetPeerId())
	if err != nil {
		return &pb.RemovePeerResponse{
			Success: false,
			Error:   err.Error(),
		}, err
	}
	return &pb.RemovePeerResponse{
		Success: true,
	}, nil
}

// GetPeerInfo handles the GetPeerInfo gRPC request.
func (s *rpcServer) GetPeerInfo(_ context.Context, req *pb.GetPeerInfoRequest) (*pb.GetPeerInfoResponse, error) {
	s.node.Fsm.mu.RLock()
	peer, exists := s.node.Fsm.raftPeers[req.GetPeerId()]
	s.node.Fsm.mu.RUnlock()

	return &pb.GetPeerInfoResponse{
		Exists:      exists,
		GrpcAddress: peer.GRPCAddress,
	}, nil
}
