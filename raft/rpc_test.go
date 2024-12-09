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
			IsBootstrap: true,
			Peers: []config.RaftPeer{
				{ID: "testRaftLeadershipnode2", Address: "127.0.0.1:" + strconv.Itoa(ports[1])},
				{ID: "testRaftLeadershipnode3", Address: "127.0.0.1:" + strconv.Itoa(ports[2])},
			},
			Directory: tempDir,
		},
		{
			NodeID:      "testRaftLeadershipnode2",
			Address:     "127.0.0.1:" + strconv.Itoa(ports[1]),
			IsBootstrap: false,
			Peers: []config.RaftPeer{
				{ID: "testRaftLeadershipnode1", Address: "127.0.0.1:" + strconv.Itoa(ports[0])},
				{ID: "testRaftLeadershipnode3", Address: "127.0.0.1:" + strconv.Itoa(ports[2])},
			},
			Directory: tempDir,
		},
		{
			NodeID:      "testRaftLeadershipnode3",
			Address:     "127.0.0.1:" + strconv.Itoa(ports[2]),
			IsBootstrap: false,
			Peers: []config.RaftPeer{
				{ID: "testRaftLeadershipnode1", Address: "127.0.0.1:" + strconv.Itoa(ports[0])},
				{ID: "testRaftLeadershipnode2", Address: "127.0.0.1:" + strconv.Itoa(ports[1])},
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
			resp, err := client.ForwardApply(context.Background(), &pb.ApplyRequest{
				Data:      testCase.data,
				TimeoutMs: testCase.timeoutMs,
			})

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
