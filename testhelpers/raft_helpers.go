package testhelpers

import (
	"fmt"
	"os"
	"path/filepath"
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
	// Create a temporary directory for Raft data
	tempDir, err := os.MkdirTemp("", "raft-test-*")
	if err != nil {
		return nil, fmt.Errorf("failed to create temp dir: %w", err)
	}

	// Setup test configuration
	nodeID := "test-node-1"
	raftAddr := "127.0.0.1:0" // Using port 0 lets the system assign a random available port

	// Create test logger
	logger := zerolog.New(zerolog.ConsoleWriter{Out: os.Stderr}).
		Level(zerolog.DebugLevel).
		With().Timestamp().Logger()

	// Create Raft configuration
	raftConfig := config.Raft{
		NodeID:   nodeID,
		Address:  raftAddr,
		LeaderID: nodeID,              // Make this node the leader for testing
		Peers:    []config.RaftPeer{}, // Empty peers for single-node testing
	}

	// Create new Raft node
	node, err := raft.NewRaftNode(logger, raftConfig)
	if err != nil {
		os.RemoveAll(tempDir)
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
		os.RemoveAll(tempDir)
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

	if err := os.RemoveAll(h.TempDir); err != nil {
		return fmt.Errorf("failed to remove temp dir: %w", err)
	}

	// Also clean up the default raft directory
	if err := os.RemoveAll(filepath.Join("raft", h.NodeID)); err != nil {
		return fmt.Errorf("failed to remove raft directory: %w", err)
	}

	return nil
}
