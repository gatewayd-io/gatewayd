package raft

import (
	"context"
	"net"
	"strconv"
	"testing"
	"time"

	"github.com/gatewayd-io/gatewayd/config"
	pb "github.com/gatewayd-io/gatewayd/raft/proto"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// setupGRPCServer creates a gRPC server for testing on a random port.
func setupGRPCServer(t *testing.T, node *Node) (*grpc.Server, net.Listener) {
	t.Helper()
	lis, err := net.Listen("tcp", "localhost:0") // Bind to a random available port
	require.NoError(t, err, "Failed to create listener")

	server := grpc.NewServer()
	pb.RegisterRaftServiceServer(server, &rpcServer{node: node})

	go func() {
		if err := server.Serve(lis); err != nil {
			t.Errorf("Failed to serve: %v", err)
		}
	}()

	return server, lis
}

// getListenerAddr returns address for the listener.
func getListenerAddr(lis net.Listener) string {
	return lis.Addr().String()
}

func setupNodes(t *testing.T, logger zerolog.Logger, ports []int, tempDir string) []*Node {
	t.Helper()
	nodeConfigs := []config.Raft{
		{
			NodeID:      "testRaftLeadershipnode1",
			Address:     "127.0.0.1:" + strconv.Itoa(ports[0]),
			GRPCAddress: "127.0.0.1:" + strconv.Itoa(ports[0]+10),
			IsBootstrap: true,
			Peers: []config.RaftPeer{
				{
					ID:          "testRaftLeadershipnode2",
					Address:     "127.0.0.1:" + strconv.Itoa(ports[1]),
					GRPCAddress: "127.0.0.1:" + strconv.Itoa(ports[1]+10),
				},
				{
					ID:          "testRaftLeadershipnode3",
					Address:     "127.0.0.1:" + strconv.Itoa(ports[2]),
					GRPCAddress: "127.0.0.1:" + strconv.Itoa(ports[2]+10),
				},
			},
			Directory: tempDir,
		},
		{
			NodeID:      "testRaftLeadershipnode2",
			Address:     "127.0.0.1:" + strconv.Itoa(ports[1]),
			GRPCAddress: "127.0.0.1:" + strconv.Itoa(ports[1]+10),
			IsBootstrap: false,
			Peers: []config.RaftPeer{
				{
					ID:          "testRaftLeadershipnode1",
					Address:     "127.0.0.1:" + strconv.Itoa(ports[0]),
					GRPCAddress: "127.0.0.1:" + strconv.Itoa(ports[0]+10),
				},
				{
					ID:      "testRaftLeadershipnode3",
					Address: "127.0.0.1:" + strconv.Itoa(ports[2]), GRPCAddress: "127.0.0.1:" + strconv.Itoa(ports[2]+10),
				},
			},
			Directory: tempDir,
		},
		{
			NodeID:      "testRaftLeadershipnode3",
			Address:     "127.0.0.1:" + strconv.Itoa(ports[2]),
			GRPCAddress: "127.0.0.1:" + strconv.Itoa(ports[2]+10),
			IsBootstrap: false,
			Peers: []config.RaftPeer{
				{
					ID:          "testRaftLeadershipnode1",
					Address:     "127.0.0.1:" + strconv.Itoa(ports[0]),
					GRPCAddress: "127.0.0.1:" + strconv.Itoa(ports[0]+10),
				},
				{
					ID:      "testRaftLeadershipnode2",
					Address: "127.0.0.1:" + strconv.Itoa(ports[1]), GRPCAddress: "127.0.0.1:" + strconv.Itoa(ports[1]+10),
				},
			},
			Directory: tempDir,
		},
	}

	nodes := make([]*Node, len(nodeConfigs))
	for i, cfg := range nodeConfigs {
		node, err := NewRaftNode(logger, cfg)
		require.NoError(t, err, "Failed to create node")
		nodes[i] = node
		t.Cleanup(func() {
			err := node.Shutdown()
			if err != nil {
				t.Errorf("Failed to shutdown node: %v", err)
			}
		})
	}
	return nodes
}

func TestRPCServer_ForwardApply(t *testing.T) {
	tests := []struct {
		name        string
		data        []byte
		timeoutMs   int64
		wantSuccess bool
		wantErr     bool
	}{
		{
			name:        "successful apply",
			data:        []byte("test data"),
			timeoutMs:   1000,
			wantSuccess: true,
			wantErr:     false,
		},
	}
	for i, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			tempDir := t.TempDir()
			logger := setupTestLogger()
			ports := []int{6004 + i, 6005 + i, 6006 + i}

			nodes := setupNodes(t, logger, ports, tempDir)

			// Wait for leader election
			time.Sleep(3 * time.Second)

			server, lis := setupGRPCServer(t, nodes[0])
			defer server.Stop()

			conn, err := grpc.NewClient(
				getListenerAddr(lis),
				grpc.WithTransportCredentials(insecure.NewCredentials()),
			)
			require.NoError(t, err, "Failed to create gRPC client connection")
			defer conn.Close()

			client := pb.NewRaftServiceClient(conn)
			resp, err := client.ForwardApply(
				context.Background(),
				&pb.ForwardApplyRequest{
					Data:      testCase.data,
					TimeoutMs: testCase.timeoutMs,
				},
			)

			if testCase.wantErr {
				assert.Error(t, err)
				return
			}

			require.NoError(t, err)
			assert.Equal(t, testCase.wantSuccess, resp.GetSuccess())
		})
	}
}

func TestRPCClient(t *testing.T) {
	t.Run("new client creation", func(t *testing.T) {
		node := &Node{}
		client := newRPCClient(node)
		assert.NotNil(t, client)
		assert.Empty(t, client.clients)
		assert.Empty(t, client.conns)
		assert.Equal(t, node, client.node)
	})

	t.Run("get client", func(t *testing.T) {
		tempDir := t.TempDir()
		logger := setupTestLogger()
		ports := []int{6014, 6015, 6016}

		nodes := setupNodes(t, logger, ports, tempDir)

		// Wait for leader election
		time.Sleep(3 * time.Second)
		client := newRPCClient(nodes[0])

		server, lis := setupGRPCServer(t, nodes[0])
		defer server.Stop()

		conn, err := grpc.NewClient(getListenerAddr(lis),
			grpc.WithTransportCredentials(insecure.NewCredentials()),
		)
		require.NoError(t, err, "Failed to create gRPC client connection")
		defer conn.Close()

		client.conns[getListenerAddr(lis)] = conn
		client.clients[getListenerAddr(lis)] = pb.NewRaftServiceClient(conn)

		existingClient, err := client.getClient(getListenerAddr(lis))
		assert.NoError(t, err)
		assert.NotNil(t, existingClient)
	})

	t.Run("close connections", func(t *testing.T) {
		node := &Node{}
		client := newRPCClient(node)

		server, lis := setupGRPCServer(t, node)
		defer server.Stop()

		conn, err := grpc.NewClient(getListenerAddr(lis),
			grpc.WithTransportCredentials(insecure.NewCredentials()),
		)
		require.NoError(t, err, "Failed to create gRPC client connection")
		defer conn.Close()

		client.conns[getListenerAddr(lis)] = conn
		client.close()

		assert.NotEqual(t, "READY", conn.GetState().String())
	})
}

func TestRPCServer_AddPeer(t *testing.T) {
	tests := []struct {
		name        string
		request     *pb.AddPeerRequest
		wantSuccess bool
		wantErr     bool
	}{
		{
			name: "successful peer addition",
			request: &pb.AddPeerRequest{
				PeerId:      "newPeer1",
				PeerAddress: "127.0.0.1:6100",
				GrpcAddress: "127.0.0.1:6101",
			},
			wantSuccess: true,
			wantErr:     false,
		},
		{
			name:        "nil request",
			request:     nil,
			wantSuccess: false,
			wantErr:     true,
		},
		{
			name: "missing peer ID",
			request: &pb.AddPeerRequest{
				PeerAddress: "127.0.0.1:6100",
				GrpcAddress: "127.0.0.1:6101",
			},
			wantSuccess: false,
			wantErr:     true,
		},
		{
			name: "missing peer address",
			request: &pb.AddPeerRequest{
				PeerId:      "newPeer2",
				GrpcAddress: "127.0.0.1:6101",
			},
			wantSuccess: false,
			wantErr:     true,
		},
		{
			name: "missing gRPC address",
			request: &pb.AddPeerRequest{
				PeerId:      "newPeer3",
				PeerAddress: "127.0.0.1:6100",
			},
			wantSuccess: false,
			wantErr:     true,
		},
	}

	for i, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			tempDir := t.TempDir()
			logger := setupTestLogger()
			ports := []int{7004 + i, 7005 + i, 7006 + i}

			nodes := setupNodes(t, logger, ports, tempDir)

			// Wait for leader election
			time.Sleep(3 * time.Second)

			server, lis := setupGRPCServer(t, nodes[0])
			defer server.Stop()

			conn, err := grpc.NewClient(
				getListenerAddr(lis),
				grpc.WithTransportCredentials(insecure.NewCredentials()),
			)
			require.NoError(t, err, "Failed to create gRPC client connection")
			defer conn.Close()

			client := pb.NewRaftServiceClient(conn)
			resp, err := client.AddPeer(context.Background(), testCase.request)

			if testCase.wantErr {
				assert.Error(t, err)
				if resp != nil {
					assert.False(t, resp.GetSuccess())
					assert.NotEmpty(t, resp.GetError())
				}
				return
			}

			require.NoError(t, err)
			assert.True(t, resp.GetSuccess())
			assert.Empty(t, resp.GetError())
		})
	}
}

func TestRPCServer_RemovePeer(t *testing.T) {
	tests := []struct {
		name        string
		request     *pb.RemovePeerRequest
		wantSuccess bool
		wantErr     bool
	}{
		{
			name: "successful peer removal",
			request: &pb.RemovePeerRequest{
				PeerId: "testRaftLeadershipnode2",
			},
			wantSuccess: true,
			wantErr:     false,
		},
		{
			name:        "nil request",
			request:     nil,
			wantSuccess: false,
			wantErr:     true,
		},
		{
			name: "missing peer ID",
			request: &pb.RemovePeerRequest{
				PeerId: "",
			},
			wantSuccess: false,
			wantErr:     true,
		},
	}

	for i, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			tempDir := t.TempDir()
			logger := setupTestLogger()
			ports := []int{8004 + i, 8005 + i, 8006 + i}

			nodes := setupNodes(t, logger, ports, tempDir)

			// Wait for leader election
			time.Sleep(3 * time.Second)

			server, lis := setupGRPCServer(t, nodes[0])
			defer server.Stop()

			conn, err := grpc.NewClient(
				getListenerAddr(lis),
				grpc.WithTransportCredentials(insecure.NewCredentials()),
			)
			require.NoError(t, err, "Failed to create gRPC client connection")
			defer conn.Close()

			client := pb.NewRaftServiceClient(conn)
			resp, err := client.RemovePeer(context.Background(), testCase.request)

			if testCase.wantErr {
				assert.Error(t, err)
				if resp != nil {
					assert.False(t, resp.GetSuccess())
					assert.NotEmpty(t, resp.GetError())
				}
				return
			}

			require.NoError(t, err)
			assert.True(t, resp.GetSuccess())
			assert.Empty(t, resp.GetError())
		})
	}
}

func TestRPCServer_GetPeerInfo(t *testing.T) {
	tests := []struct {
		name    string
		request *pb.GetPeerInfoRequest
		want    *pb.GetPeerInfoResponse
		wantErr bool
	}{
		{
			name: "existing peer",
			request: &pb.GetPeerInfoRequest{
				PeerId: "testRaftLeadershipnode2",
			},
			want: &pb.GetPeerInfoResponse{
				Exists:      true,
				GrpcAddress: "127.0.0.1:9015",
			},
			wantErr: false,
		},
		{
			name: "non-existent peer",
			request: &pb.GetPeerInfoRequest{
				PeerId: "nonexistentPeer",
			},
			want: &pb.GetPeerInfoResponse{
				Exists: false,
			},
			wantErr: false,
		},
		{
			name:    "nil request",
			request: nil,
			want:    nil,
			wantErr: true,
		},
		{
			name: "empty peer ID",
			request: &pb.GetPeerInfoRequest{
				PeerId: "",
			},
			want:    nil,
			wantErr: true,
		},
	}

	for i, testCase := range tests[0:1] {
		t.Run(testCase.name, func(t *testing.T) {
			tempDir := t.TempDir()
			logger := setupTestLogger()
			ports := []int{9004 + i, 9005 + i, 9006 + i}

			nodes := setupNodes(t, logger, ports, tempDir)

			// Wait for leader election
			time.Sleep(3 * time.Second)

			server, lis := setupGRPCServer(t, nodes[0])
			defer server.Stop()

			conn, err := grpc.NewClient(
				getListenerAddr(lis),
				grpc.WithTransportCredentials(insecure.NewCredentials()),
			)
			require.NoError(t, err, "Failed to create gRPC client connection")
			defer conn.Close()

			client := pb.NewRaftServiceClient(conn)
			resp, err := client.GetPeerInfo(context.Background(), testCase.request)

			if testCase.wantErr {
				assert.Error(t, err)
				return
			}

			require.NoError(t, err)
			assert.Equal(t, testCase.want.GetExists(), resp.GetExists())
			if testCase.want.GetExists() {
				assert.Equal(t, testCase.want.GetGrpcAddress(), resp.GetGrpcAddress())
			}
		})
	}
}
