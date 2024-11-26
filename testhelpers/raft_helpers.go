package testhelpers

import (
	"fmt"
	"net"
	"os"
	"testing"
	"time"

	"github.com/gatewayd-io/gatewayd/config"
	"github.com/gatewayd-io/gatewayd/raft"
	"github.com/rs/zerolog"
)

// TestRaftHelper contains utilities for testing Raft functionality
type TestRaftHelper struct {
	Node     *raft.RaftNode
	TempDir  string
	NodeID   string
	RaftAddr string
}

// NewTestRaftNode creates a Raft node for testing purposes
func NewTestRaftNode(t *testing.T) (*TestRaftHelper, error) {
	tempDir := t.TempDir()

	// Setup test configuration
	nodeID := fmt.Sprintf("test-node-%d", time.Now().UnixNano())
	// Get a random available port
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return nil, fmt.Errorf("failed to get random port: %w", err)
	}
	port := listener.Addr().(*net.TCPAddr).Port
	listener.Close()

	raftAddr := fmt.Sprintf("127.0.0.1:%d", port)

	// Create test logger
	logger := zerolog.New(zerolog.ConsoleWriter{Out: os.Stderr}).
		Level(zerolog.DebugLevel).
		With().Timestamp().Logger()

	// Create Raft configuration
	raftConfig := config.Raft{
		NodeID:    nodeID,
		Address:   raftAddr,
		LeaderID:  nodeID,              // Make this node the leader for testing
		Peers:     []config.RaftPeer{}, // Empty peers for single-node testing
		Directory: tempDir,
	}

	// Create new Raft node
	node, err := raft.NewRaftNode(logger, raftConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create raft node: %w", err)
	}

	// Wait for the node to become leader
	timeout := time.Now().Add(3 * time.Second)
	for time.Now().Before(timeout) {
		if node.GetState() == raft.RaftLeaderState {
			break
		}
		time.Sleep(100 * time.Millisecond)
	}

	if node.GetState() != raft.RaftLeaderState {
		return nil, fmt.Errorf("timeout waiting for node to become leader")
	}

	return &TestRaftHelper{
		Node:     node,
		TempDir:  tempDir,
		NodeID:   nodeID,
		RaftAddr: raftAddr,
	}, nil
}

// Cleanup removes temporary files and shuts down the Raft node
func (h *TestRaftHelper) Cleanup() error {
	if h.Node != nil {
		if err := h.Node.Shutdown(); err != nil {
			return fmt.Errorf("failed to shutdown raft node: %w", err)
		}
	}

	return nil
}
