package raft

import (
	"encoding/json"
	"os"
	"testing"
	"time"

	"github.com/gatewayd-io/gatewayd/config"
	"github.com/hashicorp/raft"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupTestLogger() zerolog.Logger {
	return zerolog.New(os.Stdout).With().Timestamp().Logger()
}

func TestNewRaftNode(t *testing.T) {
	logger := setupTestLogger()

	tests := []struct {
		name       string
		raftConfig config.Raft
		wantErr    bool
	}{
		{
			name: "valid configuration",
			raftConfig: config.Raft{
				NodeID:   "node1",
				Address:  "127.0.0.1:1234",
				LeaderID: "node1",
				Peers: []config.RaftPeer{
					{ID: "node2", Address: "127.0.0.1:1235"},
				},
			},
			wantErr: false,
		},
		{
			name: "invalid address",
			raftConfig: config.Raft{
				NodeID:   "node1",
				Address:  "invalid:address:",
				LeaderID: "node1",
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
				_ = os.RemoveAll("raft")
			}
		})
	}
}

func TestFSMOperations(t *testing.T) {
	fsm := NewFSM()

	// Test adding a hash mapping
	cmd := HashMapCommand{
		Type:      CommandAddHashMapping,
		Hash:      12345,
		BlockName: "test-block",
	}

	data, err := json.Marshal(cmd)
	require.NoError(t, err)

	// Apply the command
	result := fsm.Apply(&raft.Log{Data: data})
	assert.Nil(t, result)

	// Test retrieving the mapping
	blockName, exists := fsm.GetProxyBlock(12345)
	assert.True(t, exists)
	assert.Equal(t, "test-block", blockName)

	// Test non-existent mapping
	blockName, exists = fsm.GetProxyBlock(99999)
	assert.False(t, exists)
	assert.Empty(t, blockName)
}

func TestFSMSnapshot(t *testing.T) {
	fsm := NewFSM()

	// Add some data
	cmd := HashMapCommand{
		Type:      CommandAddHashMapping,
		Hash:      12345,
		BlockName: "test-block",
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
	assert.Equal(t, "test-block", fsmSnapshot.hashMap[12345])
}

func TestRaftNodeApply(t *testing.T) {
	logger := setupTestLogger()
	config := config.Raft{
		NodeID:   "node1",
		Address:  "127.0.0.1:1234",
		LeaderID: "node1",
	}

	node, err := NewRaftNode(logger, config)
	require.NoError(t, err)
	defer func() {
		_ = node.Shutdown()
		_ = os.RemoveAll("raft")
	}()

	// Test applying data
	cmd := HashMapCommand{
		Type:      CommandAddHashMapping,
		Hash:      12345,
		BlockName: "test-block",
	}
	data, err := json.Marshal(cmd)
	require.NoError(t, err)

	err = node.Apply(data, time.Second)
	// Note: This will likely fail as the node isn't a leader
	assert.Error(t, err)
}

func TestRaftLeadershipAndFollowers(t *testing.T) {
	logger := setupTestLogger()

	// Create temporary directories for each node
	defer os.RemoveAll("raft")

	// Configure three nodes
	nodes := make([]*RaftNode, 3)
	nodeConfigs := []config.Raft{
		{
			NodeID:   "node1",
			Address:  "127.0.0.1:1234",
			LeaderID: "node1", // First node is the bootstrap node
			Peers: []config.RaftPeer{
				{ID: "node2", Address: "127.0.0.1:1235"},
				{ID: "node3", Address: "127.0.0.1:1236"},
			},
		},
		{
			NodeID:   "node2",
			Address:  "127.0.0.1:1235",
			LeaderID: "node1",
			Peers: []config.RaftPeer{
				{ID: "node1", Address: "127.0.0.1:1234"},
				{ID: "node3", Address: "127.0.0.1:1236"},
			},
		},
		{
			NodeID:   "node3",
			Address:  "127.0.0.1:1236",
			LeaderID: "node1",
			Peers: []config.RaftPeer{
				{ID: "node1", Address: "127.0.0.1:1234"},
				{ID: "node2", Address: "127.0.0.1:1235"},
			},
		},
	}

	// Start all nodes
	for i, cfg := range nodeConfigs {
		node, err := NewRaftNode(logger, cfg)
		require.NoError(t, err)
		nodes[i] = node
		defer node.Shutdown()
	}

	// Wait for leader election
	time.Sleep(3 * time.Second)

	// Test 1: Verify that exactly one leader is elected
	leaderCount := 0
	var leaderNode *RaftNode
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
	cmd := HashMapCommand{
		Type:      CommandAddHashMapping,
		Hash:      12345,
		BlockName: "test-block",
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
		blockName, exists := node.Fsm.GetProxyBlock(12345)
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
