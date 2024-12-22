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

func TestWeightedRROperations(t *testing.T) {
	fsm := NewFSM()

	// Test updating weighted round robin
	cmd := Command{
		Type: CommandUpdateWeightedRR,
		Payload: WeightedRRPayload{
			GroupName: "test-group",
			ProxyName: "proxy1",
			Weight: WeightedProxy{
				CurrentWeight:   10,
				EffectiveWeight: 20,
			},
		},
	}

	data, err := json.Marshal(cmd)
	require.NoError(t, err)

	// Apply the command
	result := fsm.Apply(&raft.Log{Data: data})
	assert.Nil(t, result)

	// Test retrieving the weight
	weight, exists := fsm.GetWeightedRRState("test-group", "proxy1")
	assert.True(t, exists)
	assert.Equal(t, WeightedProxy{CurrentWeight: 10, EffectiveWeight: 20}, weight)

	// Test batch update
	batchCmd := Command{
		Type: CommandUpdateWeightedRRBatch,
		Payload: WeightedRRBatchPayload{
			GroupName: "test-group",
			Updates: map[string]WeightedProxy{
				"proxy1": {CurrentWeight: 15, EffectiveWeight: 25},
				"proxy2": {CurrentWeight: 20, EffectiveWeight: 30},
			},
		},
	}

	data, err = json.Marshal(batchCmd)
	require.NoError(t, err)

	result = fsm.Apply(&raft.Log{Data: data})
	assert.Nil(t, result)

	// Verify batch updates
	weight, exists = fsm.GetWeightedRRState("test-group", "proxy1")
	assert.True(t, exists)
	assert.Equal(t, WeightedProxy{CurrentWeight: 15, EffectiveWeight: 25}, weight)

	weight, exists = fsm.GetWeightedRRState("test-group", "proxy2")
	assert.True(t, exists)
	assert.Equal(t, WeightedProxy{CurrentWeight: 20, EffectiveWeight: 30}, weight)
}

func TestRoundRobinOperations(t *testing.T) {
	fsm := NewFSM()

	// Test adding round robin index
	cmd := Command{
		Type: CommandAddRoundRobinNext,
		Payload: RoundRobinPayload{
			GroupName: "test-group",
			NextIndex: 5,
		},
	}

	data, err := json.Marshal(cmd)
	require.NoError(t, err)

	// Apply the command
	result := fsm.Apply(&raft.Log{Data: data})
	assert.Nil(t, result)

	// Test retrieving the index
	index := fsm.GetRoundRobinNext("test-group")
	assert.Equal(t, uint32(5), index)

	// Test non-existent group
	index = fsm.GetRoundRobinNext("non-existent")
	assert.Equal(t, uint32(0), index)
}

func TestFSMInvalidCommands(t *testing.T) {
	fsm := NewFSM()

	// Test invalid command type
	cmd := Command{
		Type:    "INVALID_COMMAND",
		Payload: nil,
	}

	data, err := json.Marshal(cmd)
	require.NoError(t, err)

	result := fsm.Apply(&raft.Log{Data: data})
	err, isError := result.(error)
	assert.True(t, isError, "expected result to be an error")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unknown command type")

	// Test invalid payload
	cmd = Command{
		Type:    CommandAddConsistentHashEntry,
		Payload: "invalid payload",
	}

	data, err = json.Marshal(cmd)
	require.NoError(t, err)

	result = fsm.Apply(&raft.Log{Data: data})
	err, isError = result.(error)
	assert.True(t, isError, "expected result to be an error")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to convert payload")
}

func TestFSMRestoreSnapshot(t *testing.T) {
	fsm := NewFSM()

	// Create test data
	testData := struct {
		HashToBlock     map[string]string                   `json:"hashToBlock"`
		RoundRobin      map[string]uint32                   `json:"roundRobin"`
		WeightedRRState map[string]map[string]WeightedProxy `json:"weightedRrState"`
	}{
		HashToBlock: map[string]string{"test-hash": "test-block"},
		RoundRobin:  map[string]uint32{"test-group": 5},
		WeightedRRState: map[string]map[string]WeightedProxy{
			"test-group": {
				"proxy1": {CurrentWeight: 10, EffectiveWeight: 20},
			},
		},
	}

	// Create a pipe to simulate snapshot reading
	reader, writer := io.Pipe()

	// Write test data in a goroutine
	go func() {
		encoder := json.NewEncoder(writer)
		err := encoder.Encode(testData)
		assert.NoError(t, err)
		writer.Close()
	}()

	// Restore from snapshot
	err := fsm.Restore(reader)
	assert.NoError(t, err)

	// Verify restored data
	blockName, exists := fsm.GetProxyBlock("test-hash")
	assert.True(t, exists)
	assert.Equal(t, "test-block", blockName)

	index := fsm.GetRoundRobinNext("test-group")
	assert.Equal(t, uint32(5), index)

	weight, exists := fsm.GetWeightedRRState("test-group", "proxy1")
	assert.True(t, exists)
	assert.Equal(t, WeightedProxy{CurrentWeight: 10, EffectiveWeight: 20}, weight)
}

func TestNodeShutdown(t *testing.T) {
	logger := setupTestLogger()
	tempDir := t.TempDir()

	config := config.Raft{
		NodeID:      "shutdown-test-node",
		Address:     "127.0.0.1:0",
		IsBootstrap: true,
		Directory:   tempDir,
	}

	node, err := NewRaftNode(logger, config)
	require.NoError(t, err)

	// Test multiple shutdowns
	err = node.Shutdown()
	assert.NoError(t, err)

	err = node.Shutdown()
	assert.NoError(t, err) // Should not error on multiple shutdowns
}

func TestGetHealthStatus(t *testing.T) {
	logger := setupTestLogger()
	tempDir := t.TempDir()

	// Create a Raft node configuration for the first node
	nodeConfig1 := config.Raft{
		NodeID:      "testGetHealthStatusNode1",
		Address:     "127.0.0.1:0",
		IsBootstrap: true,
		Directory:   tempDir,
		Peers: []config.RaftPeer{
			{ID: "testGetHealthStatusNode2", Address: "127.0.0.1:5678"},
		},
	}

	// Create a Raft node configuration for the second node
	nodeConfig2 := config.Raft{
		NodeID:      "testGetHealthStatusNode2",
		Address:     "127.0.0.1:5678",
		IsBootstrap: false,
		Directory:   tempDir,
	}

	// Initialize the first Raft node
	node1, err := NewRaftNode(logger, nodeConfig1)
	require.NoError(t, err)
	defer func() {
		_ = node1.Shutdown()
	}()

	// Initialize the second Raft node
	node2, err := NewRaftNode(logger, nodeConfig2)
	require.NoError(t, err)
	defer func() {
		_ = node2.Shutdown()
	}()

	// Wait for leader election
	time.Sleep(3 * time.Second)

	// Test 1: Check health status when node1 is the leader
	if node1.GetState() == raft.Leader {
		healthStatus := node1.GetHealthStatus()
		assert.True(t, healthStatus.IsHealthy, "Leader node should be healthy")
		assert.True(t, healthStatus.IsLeader, "Node should be the leader")
		assert.True(t, healthStatus.HasLeader, "Node should have a leader (itself)")
		assert.NoError(t, healthStatus.Error, "There should be no error in health status")
		assert.Equal(t, healthStatus.LastContact, time.Duration(0), "Last contact should be 0")
	}

	// Test 2: Check health status when node2 is a follower
	if node2.GetState() == raft.Follower {
		healthStatus := node2.GetHealthStatus()
		assert.True(t, healthStatus.IsHealthy, "Follower node should be healthy")
		assert.False(t, healthStatus.IsLeader, "Node should not be the leader")
		assert.True(t, healthStatus.HasLeader, "Node should have a leader")
		assert.NoError(t, healthStatus.Error, "There should be no error in health status")
		assert.Greater(t, healthStatus.LastContact.Milliseconds(), int64(0), "Last contact should be greater than 0")
	}

	// Test 3: Check health status when no leader is available
	// Simulate no leader by not bootstrapping any node
	nodeConfig3 := config.Raft{
		NodeID:      "testGetHealthStatusNode3",
		Address:     "127.0.0.1:0",
		IsBootstrap: false,
		Directory:   tempDir,
	}
	node3, err := NewRaftNode(logger, nodeConfig3)
	require.NoError(t, err)
	defer func() {
		_ = node3.Shutdown()
	}()

	// Wait for the node to realize there's no leader
	time.Sleep(3 * time.Second)

	healthStatus := node3.GetHealthStatus()
	assert.False(t, healthStatus.IsHealthy, "Node should not be healthy without a leader")
	assert.False(t, healthStatus.IsLeader, "Node should not be the leader")
	assert.False(t, healthStatus.HasLeader, "Node should not have a leader")
	assert.Error(t, healthStatus.Error, "There should be an error indicating no leader")
	assert.Equal(t, healthStatus.LastContact, time.Duration(-1), "Last contact should be -1")
}
