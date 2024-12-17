package raft

import (
	"encoding/json"
	"io"
	"testing"
	"time"

	"github.com/gatewayd-io/gatewayd/config"
	"github.com/hashicorp/raft"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupTestLogger() zerolog.Logger {
	return zerolog.New(io.Discard).With().Timestamp().Logger()
}

func TestNewRaftNode(t *testing.T) {
	logger := setupTestLogger()
	tempDir := t.TempDir()

	tests := []struct {
		name       string
		raftConfig config.Raft
		wantErr    bool
	}{
		{
			name: "valid configuration",
			raftConfig: config.Raft{
				NodeID:      "testRaftNodeValidConfigurationnode1",
				Address:     "127.0.0.1:6001",
				IsBootstrap: true,
				Peers: []config.RaftPeer{
					{ID: "testRaftNodeValidConfigurationnode2", Address: "127.0.0.1:6002"},
				},
				Directory: tempDir,
			},
			wantErr: false,
		},
		{
			name: "invalid address",
			raftConfig: config.Raft{
				NodeID:      "testRaftNodeInvalidAddressnode1",
				Address:     "invalid:address:",
				IsBootstrap: true,
				Directory:   tempDir,
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			node, err := NewRaftNode(logger, tt.raftConfig)
			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, node)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, node)
				// Cleanup
				_ = node.Shutdown()
			}
		})
	}
}

func TestFSMOperations(t *testing.T) {
	fsm := NewFSM()

	// Test adding a hash mapping
	cmd := Command{
		Type: CommandAddConsistentHashEntry,
		Payload: ConsistentHashPayload{
			Hash:      "12345",
			BlockName: "test-block",
		},
	}

	data, err := json.Marshal(cmd)
	require.NoError(t, err)

	// Apply the command
	result := fsm.Apply(&raft.Log{Data: data})
	assert.Nil(t, result)

	// Test retrieving the mapping
	blockName, exists := fsm.GetProxyBlock("12345")
	assert.True(t, exists)
	assert.Equal(t, "test-block", blockName)

	// Test non-existent mapping
	blockName, exists = fsm.GetProxyBlock("99999")
	assert.False(t, exists)
	assert.Empty(t, blockName)
}

func TestFSMSnapshot(t *testing.T) {
	fsm := NewFSM()

	// Add some data
	cmd := Command{
		Type: CommandAddConsistentHashEntry,
		Payload: ConsistentHashPayload{
			Hash:      "12345",
			BlockName: "test-block",
		},
	}
	data, err := json.Marshal(cmd)
	require.NoError(t, err)
	fsm.Apply(&raft.Log{Data: data})

	// Create snapshot
	snapshot, err := fsm.Snapshot()
	require.NoError(t, err)
	assert.NotNil(t, snapshot)

	// Verify snapshot data
	fsmSnapshot, ok := snapshot.(*FSMSnapshot)
	assert.True(t, ok)
	assert.Equal(t, "test-block", fsmSnapshot.lbHashToBlockName["12345"])
}

func TestRaftNodeApply(t *testing.T) {
	logger := setupTestLogger()
	tempDir := t.TempDir()
	config := config.Raft{
		NodeID:      "testRaftNodeApplynode1",
		Address:     "127.0.0.1:6003",
		IsBootstrap: true,
		Directory:   tempDir,
	}

	node, err := NewRaftNode(logger, config)
	require.NoError(t, err)
	defer func() {
		_ = node.Shutdown()
	}()

	// Test applying data
	cmd := Command{
		Type: CommandAddConsistentHashEntry,
		Payload: ConsistentHashPayload{
			Hash:      "12345",
			BlockName: "test-block",
		},
	}
	data, err := json.Marshal(cmd)
	require.NoError(t, err)

	err = node.Apply(data, time.Second)
	// Note: This will likely fail as the node isn't a leader
	assert.Error(t, err)
}

func TestRaftLeadershipAndFollowers(t *testing.T) {
	logger := setupTestLogger()
	tempDir := t.TempDir()

	// Configure three nodes with unique ports
	nodeConfigs := []config.Raft{
		{
			NodeID:      "testRaftLeadershipnode1",
			Address:     "127.0.0.1:6004",
			IsBootstrap: true,
			Peers: []config.RaftPeer{
				{ID: "testRaftLeadershipnode2", Address: "127.0.0.1:6005"},
				{ID: "testRaftLeadershipnode3", Address: "127.0.0.1:6006"},
			},
			Directory: tempDir,
		},
		{
			NodeID:      "testRaftLeadershipnode2",
			Address:     "127.0.0.1:6005",
			IsBootstrap: false,
			Peers: []config.RaftPeer{
				{ID: "testRaftLeadershipnode1", Address: "127.0.0.1:6004"},
				{ID: "testRaftLeadershipnode3", Address: "127.0.0.1:6006"},
			},
			Directory: tempDir,
		},
		{
			NodeID:      "testRaftLeadershipnode3",
			Address:     "127.0.0.1:6006",
			IsBootstrap: false,
			Peers: []config.RaftPeer{
				{ID: "testRaftLeadershipnode1", Address: "127.0.0.1:6004"},
				{ID: "testRaftLeadershipnode2", Address: "127.0.0.1:6005"},
			},
			Directory: tempDir,
		},
	}

	// Start all nodes
	nodes := make([]*Node, len(nodeConfigs))
	for i, cfg := range nodeConfigs {
		node, err := NewRaftNode(logger, cfg)
		require.NoError(t, err)
		nodes[i] = node
		defer func() {
			err := node.Shutdown()
			if err != nil {
				t.Errorf("Failed to shutdown node: %v", err)
			}
		}()
	}

	// Wait for leader election
	time.Sleep(3 * time.Second)

	// Test 1: Verify that exactly one leader is elected
	leaderCount := 0
	var leaderNode *Node
	for _, node := range nodes {
		if node.GetState() == raft.Leader {
			leaderCount++
			leaderNode = node
		}
	}
	assert.Equal(t, 1, leaderCount, "Expected exactly one leader")
	require.NotNil(t, leaderNode, "Expected to find a leader")

	// Test 2: Verify that other nodes are followers
	for _, node := range nodes {
		if node != leaderNode {
			assert.Equal(t, raft.Follower, node.GetState(), "Expected non-leader nodes to be followers")
		}
	}

	// Test 3: Test cluster functionality by applying a command
	cmd := Command{
		Type: CommandAddConsistentHashEntry,
		Payload: ConsistentHashPayload{
			Hash:      "12345",
			BlockName: "test-block",
		},
	}
	data, err := json.Marshal(cmd)
	require.NoError(t, err)

	// Apply command through leader
	err = leaderNode.Apply(data, 5*time.Second)
	require.NoError(t, err, "Failed to apply command through leader")

	// Wait for replication
	time.Sleep(1 * time.Second)

	// Verify that all nodes have the update
	for i, node := range nodes {
		blockName, exists := node.Fsm.GetProxyBlock("12345")
		assert.True(t, exists, "Node %d should have the replicated data", i+1)
		assert.Equal(t, "test-block", blockName, "Node %d has incorrect data", i+1)
	}

	// Test 4: Simulate leader shutdown and new leader election
	err = leaderNode.Shutdown()
	require.NoError(t, err)

	// Wait for new leader election
	time.Sleep(3 * time.Second)

	// Verify new leader is elected among remaining nodes
	newLeaderCount := 0
	for _, node := range nodes {
		if node != leaderNode && node.GetState() == raft.Leader {
			newLeaderCount++
		}
	}
	assert.Equal(t, 1, newLeaderCount, "Expected exactly one new leader after original leader shutdown")
}
