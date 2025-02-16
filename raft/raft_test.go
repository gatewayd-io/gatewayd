package raft

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"io"
	"math/big"
	"net"
	"os"
	"path/filepath"
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

	err = node.Apply(context.Background(), data, time.Second)
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
		state, _ := node.GetState()
		if state == raft.Leader {
			leaderCount++
			leaderNode = node
		}
	}
	assert.Equal(t, 1, leaderCount, "Expected exactly one leader")
	require.NotNil(t, leaderNode, "Expected to find a leader")

	// Test 2: Verify that other nodes are followers
	for _, node := range nodes {
		if node != leaderNode {
			state, _ := node.GetState()
			assert.Equal(t, raft.Follower, state, "Expected non-leader nodes to be followers")
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
	err = leaderNode.Apply(context.Background(), data, 5*time.Second)
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
		state, _ := node.GetState()
		if node != leaderNode && state == raft.Leader {
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
	state, _ := node1.GetState()
	if state == raft.Leader {
		healthStatus := node1.GetHealthStatus()
		assert.True(t, healthStatus.IsHealthy, "Leader node should be healthy")
		assert.True(t, healthStatus.IsLeader, "Node should be the leader")
		assert.True(t, healthStatus.HasLeader, "Node should have a leader (itself)")
		assert.NoError(t, healthStatus.Error, "There should be no error in health status")
		assert.Equal(t, healthStatus.LastContact, time.Duration(0), "Last contact should be 0")
	}

	// Test 2: Check health status when node2 is a follower
	state, _ = node2.GetState()
	if state == raft.Follower {
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

func TestAddPeer(t *testing.T) {
	logger := setupTestLogger()

	tempDir := t.TempDir()
	nodeConfig1 := config.Raft{
		NodeID:      "testAddPeerNode1",
		Address:     "127.0.0.1:5679",
		IsBootstrap: true,
		Directory:   tempDir,
		GRPCAddress: "127.0.0.1:5680",
	}

	node1, err := NewRaftNode(logger, nodeConfig1)
	require.NoError(t, err)
	defer func() {
		_ = node1.Shutdown()
	}()

	time.Sleep(2 * time.Second)

	// Add node2 to the cluster when peer is leader
	tempDir2 := t.TempDir()
	nodeConfig2 := config.Raft{
		NodeID:      "testAddPeerNode2",
		Address:     "127.0.0.1:5689",
		IsBootstrap: false,
		Directory:   tempDir2,
		GRPCAddress: "127.0.0.1:5690",
		Peers: []config.RaftPeer{
			{ID: "testAddPeerNode1", Address: "127.0.0.1:5679", GRPCAddress: "127.0.0.1:5680"},
		},
	}

	node2, err := NewRaftNode(logger, nodeConfig2)
	require.NoError(t, err)
	defer func() {
		_ = node2.Shutdown()
	}()

	// Add node3 to the cluster when peer is follower
	tempDir3 := t.TempDir()
	nodeConfig3 := config.Raft{
		NodeID:      "testAddPeerNode3",
		Address:     "127.0.0.1:5699",
		IsBootstrap: false,
		Directory:   tempDir3,
		GRPCAddress: "127.0.0.1:5700",
		Peers: []config.RaftPeer{
			{ID: "testAddPeerNode1", Address: "127.0.0.1:5689", GRPCAddress: "127.0.0.1:5690"},
		},
	}

	node3, err := NewRaftNode(logger, nodeConfig3)
	require.NoError(t, err)
	defer func() {
		_ = node3.Shutdown()
	}()

	// Function to check if a node is in the cluster configuration
	checkNodeInCluster := func(nodeID string) bool {
		existingConfig := node1.raft.GetConfiguration().Configuration()
		for _, server := range existingConfig.Servers {
			if server.ID == raft.ServerID(nodeID) {
				return true
			}
		}
		return false
	}

	// Wait and verify that both nodes join the cluster
	require.Eventually(t, func() bool {
		return checkNodeInCluster(nodeConfig2.NodeID) && checkNodeInCluster(nodeConfig3.NodeID)
	}, 30*time.Second, 1*time.Second, "Nodes failed to join the cluster")

	// Verify the cluster has exactly 3 nodes
	existingConfig := node1.raft.GetConfiguration().Configuration()
	assert.Equal(t, 3, len(existingConfig.Servers), "Cluster should have exactly 3 nodes")

	// Verify that node2 and node3 recognize node1 as the leader
	time.Sleep(2 * time.Second) // Give some time for leader recognition

	_, leaderID := node2.raft.LeaderWithID()
	assert.Equal(t, raft.ServerID(nodeConfig1.NodeID), leaderID, "Node2 should recognize Node1 as leader")

	_, leaderID = node3.raft.LeaderWithID()
	assert.Equal(t, raft.ServerID(nodeConfig1.NodeID), leaderID, "Node3 should recognize Node1 as leader")
}

func TestRemovePeer(t *testing.T) {
	logger := setupTestLogger()
	tempDir := t.TempDir()

	// Configure three nodes
	nodeConfigs := []config.Raft{
		{
			NodeID:      "testRemovePeerNode1",
			Address:     "127.0.0.1:5779",
			IsBootstrap: true,
			Directory:   tempDir,
			GRPCAddress: "127.0.0.1:5780",
			Peers: []config.RaftPeer{
				{ID: "testRemovePeerNode2", Address: "127.0.0.1:5789", GRPCAddress: "127.0.0.1:5790"},
				{ID: "testRemovePeerNode3", Address: "127.0.0.1:5799", GRPCAddress: "127.0.0.1:5800"},
			},
		},
		{
			NodeID:      "testRemovePeerNode2",
			Address:     "127.0.0.1:5789",
			IsBootstrap: false,
			Directory:   tempDir,
			GRPCAddress: "127.0.0.1:5790",
			Peers: []config.RaftPeer{
				{ID: "testRemovePeerNode1", Address: "127.0.0.1:5779", GRPCAddress: "127.0.0.1:5780"},
				{ID: "testRemovePeerNode3", Address: "127.0.0.1:5799", GRPCAddress: "127.0.0.1:5800"},
			},
		},
		{
			NodeID:      "testRemovePeerNode3",
			Address:     "127.0.0.1:5799",
			IsBootstrap: false,
			Directory:   tempDir,
			GRPCAddress: "127.0.0.1:5800",
			Peers: []config.RaftPeer{
				{ID: "testRemovePeerNode1", Address: "127.0.0.1:5779", GRPCAddress: "127.0.0.1:5780"},
				{ID: "testRemovePeerNode2", Address: "127.0.0.1:5789", GRPCAddress: "127.0.0.1:5790"},
			},
		},
	}

	// Start all nodes
	nodes := make([]*Node, len(nodeConfigs))
	defer func() {
		for _, node := range nodes {
			if node != nil {
				_ = node.Shutdown()
			}
		}
	}()

	// Initialize nodes
	for i, cfg := range nodeConfigs {
		node, err := NewRaftNode(logger, cfg)
		require.NoError(t, err)
		nodes[i] = node
	}

	// Wait for cluster to stabilize and leader election
	require.Eventually(t, func() bool {
		leaderCount := 0
		for _, node := range nodes {
			state, _ := node.GetState()
			if state == raft.Leader {
				leaderCount++
			}
		}
		return leaderCount == 1
	}, 10*time.Second, 100*time.Millisecond, "Failed to elect a leader")

	// Find the leader node
	var leaderNode *Node
	for _, node := range nodes {
		state, _ := node.GetState()
		if state == raft.Leader {
			leaderNode = node
			break
		}
	}
	require.NotNil(t, leaderNode, "Leader node not found")

	// Verify initial cluster configuration
	initialConfig := leaderNode.raft.GetConfiguration().Configuration()
	require.Equal(t, 3, len(initialConfig.Servers), "Initial cluster should have 3 nodes")

	// also need to verify that the peers are in the FSM
	require.Eventually(t, func() bool {
		return len(leaderNode.Fsm.raftPeers) == 3
	}, 10*time.Second, 100*time.Millisecond, "Peers not found in FSM")

	// Test removing a follower node
	followerID := nodeConfigs[1].NodeID // Second node
	err := leaderNode.RemovePeer(context.Background(), followerID)
	require.NoError(t, err, "Failed to remove follower node")

	// Wait for the removal to take effect and verify
	require.Eventually(t, func() bool {
		config := leaderNode.raft.GetConfiguration().Configuration()
		return len(config.Servers) == 2
	}, 5*time.Second, 100*time.Millisecond, "Node removal not reflected in configuration")

	// Verify the removed node is not in the configuration
	currentConfig := leaderNode.raft.GetConfiguration().Configuration()
	for _, server := range currentConfig.Servers {
		require.NotEqual(t, raft.ServerID(followerID), server.ID, "Removed node should not be in configuration")
	}

	// Test removing the leader node
	leaderID := leaderNode.config.LocalID
	err = leaderNode.RemovePeer(context.Background(), string(leaderID))
	require.NoError(t, err, "Failed to remove leader node")

	// Wait for new leader election and verify cluster size
	require.Eventually(t, func() bool {
		// Check remaining node's configuration
		for _, node := range nodes {
			state, _ := node.GetState()
			if state == raft.Leader {
				config := node.raft.GetConfiguration().Configuration()
				return len(config.Servers) == 1
			}
		}
		return false
	}, 10*time.Second, 100*time.Millisecond, "Failed to elect new leader after removing old leader")

	// Verify the final state
	var newLeaderNode *Node
	for _, node := range nodes {
		state, _ := node.GetState()
		if state == raft.Leader {
			newLeaderNode = node
			break
		}
	}
	require.NotNil(t, newLeaderNode, "New leader should be elected")

	finalConfig := newLeaderNode.raft.GetConfiguration().Configuration()
	require.Equal(t, 1, len(finalConfig.Servers), "Final cluster should have 1 node")
	require.NotEqual(t, leaderID, finalConfig.Servers[0].ID,
		"Old leader should not be in final configuration")
}

func TestSecureGRPCConfiguration(t *testing.T) {
	logger := setupTestLogger()
	tempDir := t.TempDir()

	// Create temporary certificate files
	certFile := filepath.Join(tempDir, "cert.pem")
	keyFile := filepath.Join(tempDir, "key.pem")

	// Generate self-signed certificate for testing
	err := generateTestCertificate(certFile, keyFile)
	require.NoError(t, err, "Failed to generate test certificates")

	tests := []struct {
		name       string
		raftConfig config.Raft
		wantErr    bool
		errMsg     string
	}{
		{
			name: "valid secure configuration",
			raftConfig: config.Raft{
				NodeID:      "secureNode1",
				Address:     "127.0.0.1:0",
				GRPCAddress: "127.0.0.1:0",
				IsBootstrap: true,
				Directory:   tempDir,
				IsSecure:    true,
				CertFile:    certFile,
				KeyFile:     keyFile,
			},
			wantErr: false,
		},
		{
			name: "secure mode without cert files",
			raftConfig: config.Raft{
				NodeID:      "secureNode2",
				Address:     "127.0.0.1:0",
				GRPCAddress: "127.0.0.1:0",
				IsBootstrap: true,
				Directory:   tempDir,
				IsSecure:    true,
				CertFile:    "",
				KeyFile:     "",
			},
			wantErr: true,
			errMsg:  "TLS certificate and key files are required when secure mode is enabled",
		},
		{
			name: "secure mode with invalid cert file",
			raftConfig: config.Raft{
				NodeID:      "secureNode3",
				Address:     "127.0.0.1:0",
				GRPCAddress: "127.0.0.1:0",
				IsBootstrap: true,
				Directory:   tempDir,
				IsSecure:    true,
				CertFile:    "nonexistent.pem",
				KeyFile:     keyFile,
			},
			wantErr: true,
			errMsg:  "failed to load TLS credentials",
		},
		{
			name: "non-secure configuration",
			raftConfig: config.Raft{
				NodeID:      "secureNode4",
				Address:     "127.0.0.1:0",
				GRPCAddress: "127.0.0.1:0",
				IsBootstrap: true,
				Directory:   tempDir,
				IsSecure:    false,
			},
			wantErr: false,
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			node, err := NewRaftNode(logger, testCase.raftConfig)
			if testCase.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), testCase.errMsg)
				assert.Nil(t, node)
			} else {
				require.NoError(t, err)
				require.NotNil(t, node)
				assert.Equal(t, testCase.raftConfig.IsSecure, node.grpcIsSecure)

				// Cleanup
				err = node.Shutdown()
				assert.NoError(t, err)
			}
		})
	}
}

// generateTestCertificate creates a self-signed certificate for testing.
func generateTestCertificate(certFile, keyFile string) error {
	// Generate private key
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return fmt.Errorf("failed to generate private key: %w", err)
	}

	// Create certificate template
	template := x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject: pkix.Name{
			Organization: []string{"Test Org"},
		},
		NotBefore: time.Now(),
		NotAfter:  time.Now().Add(time.Hour * 24),

		KeyUsage:              x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
		IPAddresses:           []net.IP{net.ParseIP("127.0.0.1")},
	}

	// Create certificate
	derBytes, err := x509.CreateCertificate(rand.Reader, &template, &template, &privateKey.PublicKey, privateKey)
	if err != nil {
		return fmt.Errorf("failed to create certificate: %w", err)
	}

	// Write certificate to file
	certOut, err := os.Create(certFile)
	if err != nil {
		return fmt.Errorf("failed to create cert file: %w", err)
	}
	defer certOut.Close()
	if err := pem.Encode(certOut, &pem.Block{Type: "CERTIFICATE", Bytes: derBytes}); err != nil {
		return fmt.Errorf("failed to encode certificate: %w", err)
	}

	// Write private key to file
	keyOut, err := os.Create(keyFile)
	if err != nil {
		return fmt.Errorf("failed to create key file: %w", err)
	}
	defer keyOut.Close()
	if err := pem.Encode(keyOut, &pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(privateKey),
	}); err != nil {
		return fmt.Errorf("failed to encode private key: %w", err)
	}

	return nil
}

func TestFSMPeerOperations(t *testing.T) {
	logger := setupTestLogger()
	tempDir := t.TempDir()

	nodeConfigs := []config.Raft{
		{
			NodeID:      "testFSMPeerNode1",
			Address:     "127.0.0.1:6779",
			IsBootstrap: true,
			Directory:   tempDir,
			GRPCAddress: "127.0.0.1:6780",
			Peers:       []config.RaftPeer{},
		},
		{
			NodeID:      "testFSMPeerNode2",
			Address:     "127.0.0.1:6789",
			IsBootstrap: false,
			Directory:   tempDir,
			GRPCAddress: "127.0.0.1:6790",
			Peers: []config.RaftPeer{
				{ID: "testFSMPeerNode1", Address: "127.0.0.1:6779", GRPCAddress: "127.0.0.1:6780"},
			},
		},
		{
			NodeID:      "testFSMPeerNode3",
			Address:     "127.0.0.1:6799",
			IsBootstrap: false,
			Directory:   tempDir,
			GRPCAddress: "127.0.0.1:6800",
			Peers: []config.RaftPeer{
				{ID: "testFSMPeerNode1", Address: "127.0.0.1:6779", GRPCAddress: "127.0.0.1:6780"},
			},
		},
	}

	// Create nodes and set up deferred cleanup
	nodes := make([]*Node, 0, len(nodeConfigs))
	defer func() {
		for _, node := range nodes {
			if node != nil {
				_ = node.Shutdown()
			}
		}
	}()

	// Initialize node1 (bootstrap node)
	node1, err := NewRaftNode(logger, nodeConfigs[0])
	require.NoError(t, err)
	nodes = append(nodes, node1)

	// Wait for node1 to become leader
	require.Eventually(t, func() bool {
		state, _ := node1.GetState()
		return state == raft.Leader
	}, 10*time.Second, 100*time.Millisecond, "Node 1 failed to become leader")

	// Initialize node2
	node2, err := NewRaftNode(logger, nodeConfigs[1])
	require.NoError(t, err)
	nodes = append(nodes, node2)

	// Wait for node2 to join the cluster
	require.Eventually(t, func() bool {
		return node2.raft.Leader() != "" && len(node2.Fsm.raftPeers) >= 2
	}, 10*time.Second, 100*time.Millisecond, "Node 2 failed to join the cluster")

	// Initialize node3
	node3, err := NewRaftNode(logger, nodeConfigs[2])
	require.NoError(t, err)
	nodes = append(nodes, node3)

	// Wait for node3 to join the cluster
	require.Eventually(t, func() bool {
		return node3.raft.Leader() != "" && len(node3.Fsm.raftPeers) >= 3
	}, 10*time.Second, 100*time.Millisecond, "Node 3 failed to join the cluster")

	// Wait for FSM state synchronization
	require.Eventually(t, func() bool {
		return len(node1.Fsm.raftPeers) == 3 &&
			len(node2.Fsm.raftPeers) == 3 &&
			len(node3.Fsm.raftPeers) == 3
	}, 10*time.Second, 100*time.Millisecond, "Failed to synchronize FSM state across nodes")

	// Verify peer information is consistent across all nodes
	for _, node := range nodes {
		peers := node.Fsm.raftPeers
		require.Equal(t, 3, len(peers), "Node should have exactly 3 peers")

		// Verify each peer has the correct information
		expectedPeers := map[string]bool{
			"testFSMPeerNode1": false,
			"testFSMPeerNode2": false,
			"testFSMPeerNode3": false,
		}

		for peerID, peer := range peers {
			require.Contains(t, expectedPeers, peerID, "Unexpected peer ID found")
			require.NotEmpty(t, peer.Address, "Peer address should not be empty")
			require.NotEmpty(t, peer.GRPCAddress, "Peer gRPC address should not be empty")
			expectedPeers[peerID] = true
		}

		// Verify all expected peers were found
		for peerID, found := range expectedPeers {
			require.True(t, found, "Expected peer %s not found in FSM state", peerID)
		}
	}

	// Verify leader is consistent across all nodes
	leaderAddr := node1.raft.Leader()
	require.NotEmpty(t, leaderAddr, "Leader address should not be empty")
	for _, node := range nodes[1:] {
		require.Equal(t, leaderAddr, node.raft.Leader(),
			"Leader address should be consistent across all nodes")
	}
}

func TestLeaveCluster(t *testing.T) {
	logger := setupTestLogger()
	tempDir := t.TempDir()

	// Test cases for different cluster scenarios
	tests := []struct {
		name       string
		setupNodes func() []*Node
		testNode   int  // index of node that will leave
		wantErr    bool // whether we expect an error
	}{
		{
			name: "single node cluster",
			setupNodes: func() []*Node {
				node, err := NewRaftNode(logger, config.Raft{
					NodeID:      "singleNode",
					Address:     "127.0.0.1:0",
					GRPCAddress: "127.0.0.1:0",
					IsBootstrap: true,
					Directory:   tempDir,
				})
				require.NoError(t, err)
				return []*Node{node}
			},
			testNode: 0,
			wantErr:  false,
		},
		{
			name: "follower leaves multi-node cluster",
			setupNodes: func() []*Node {
				// Create a 3-node cluster
				nodes := make([]*Node, 0, 3)

				// Leader node
				node1, err := NewRaftNode(logger, config.Raft{
					NodeID:      "multiNode1",
					Address:     "127.0.0.1:7011",
					GRPCAddress: "127.0.0.1:7021",
					IsBootstrap: true,
					Directory:   tempDir,
					Peers: []config.RaftPeer{
						{ID: "multiNode2", Address: "127.0.0.1:7012", GRPCAddress: "127.0.0.1:7022"},
						{ID: "multiNode3", Address: "127.0.0.1:7013", GRPCAddress: "127.0.0.1:7023"},
					},
				})
				require.NoError(t, err)
				nodes = append(nodes, node1)

				// Follower nodes
				node2, err := NewRaftNode(logger, config.Raft{
					NodeID:      "multiNode2",
					Address:     "127.0.0.1:7012",
					GRPCAddress: "127.0.0.1:7022",
					IsBootstrap: false,
					Directory:   tempDir,
					Peers: []config.RaftPeer{
						{ID: "multiNode1", Address: "127.0.0.1:7011", GRPCAddress: "127.0.0.1:7021"},
						{ID: "multiNode3", Address: "127.0.0.1:7013", GRPCAddress: "127.0.0.1:7023"},
					},
				})
				require.NoError(t, err)
				nodes = append(nodes, node2)

				node3, err := NewRaftNode(logger, config.Raft{
					NodeID:      "multiNode3",
					Address:     "127.0.0.1:7013",
					GRPCAddress: "127.0.0.1:7023",
					IsBootstrap: false,
					Directory:   tempDir,
					Peers: []config.RaftPeer{
						{ID: "multiNode1", Address: "127.0.0.1:7011", GRPCAddress: "127.0.0.1:7021"},
						{ID: "multiNode2", Address: "127.0.0.1:7012", GRPCAddress: "127.0.0.1:7022"},
					},
				})
				require.NoError(t, err)
				nodes = append(nodes, node3)

				// Wait for cluster to stabilize
				time.Sleep(3 * time.Second)
				return nodes
			},
			testNode: 2, // Test with the last follower
			wantErr:  false,
		},
		{
			name: "leader leaves multi-node cluster",
			setupNodes: func() []*Node {
				// Similar setup to previous test case
				nodes := make([]*Node, 0, 3)

				node1, err := NewRaftNode(logger, config.Raft{
					NodeID:      "leaderNode1",
					Address:     "127.0.0.1:7021",
					GRPCAddress: "127.0.0.1:7031",
					IsBootstrap: true,
					Directory:   tempDir,
					Peers: []config.RaftPeer{
						{ID: "leaderNode2", Address: "127.0.0.1:7022", GRPCAddress: "127.0.0.1:7032"},
						{ID: "leaderNode3", Address: "127.0.0.1:7023", GRPCAddress: "127.0.0.1:7033"},
					},
				})
				require.NoError(t, err)
				nodes = append(nodes, node1)

				node2, err := NewRaftNode(logger, config.Raft{
					NodeID:      "leaderNode2",
					Address:     "127.0.0.1:7022",
					GRPCAddress: "127.0.0.1:7032",
					IsBootstrap: false,
					Directory:   tempDir,
					Peers: []config.RaftPeer{
						{ID: "leaderNode1", Address: "127.0.0.1:7021", GRPCAddress: "127.0.0.1:7031"},
						{ID: "leaderNode3", Address: "127.0.0.1:7023", GRPCAddress: "127.0.0.1:7033"},
					},
				})
				require.NoError(t, err)
				nodes = append(nodes, node2)

				node3, err := NewRaftNode(logger, config.Raft{
					NodeID:      "leaderNode3",
					Address:     "127.0.0.1:7023",
					GRPCAddress: "127.0.0.1:7033",
					IsBootstrap: false,
					Directory:   tempDir,
					Peers: []config.RaftPeer{
						{ID: "leaderNode1", Address: "127.0.0.1:7021", GRPCAddress: "127.0.0.1:7031"},
						{ID: "leaderNode2", Address: "127.0.0.1:7022", GRPCAddress: "127.0.0.1:7032"},
					},
				})
				require.NoError(t, err)
				nodes = append(nodes, node3)

				// Wait for cluster to stabilize
				time.Sleep(3 * time.Second)
				return nodes
			},
			testNode: 0, // Test with the leader
			wantErr:  false,
		},
		{
			name: "nil node",
			setupNodes: func() []*Node {
				return []*Node{nil}
			},
			testNode: 0,
			wantErr:  true,
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			nodes := testCase.setupNodes()
			defer func() {
				for _, node := range nodes {
					if node != nil {
						_ = node.Shutdown()
					}
				}
			}()

			// For multi-node clusters, verify the initial state
			if len(nodes) > 1 {
				require.Eventually(t, func() bool {
					leaderCount := 0
					followerCount := 0
					for _, node := range nodes {
						if node != nil {
							state, _ := node.GetState()
							if state == raft.Leader {
								leaderCount++
							} else if state == raft.Follower {
								followerCount++
							}
						}
					}
					return leaderCount == 1 && followerCount == len(nodes)-1
				}, 10*time.Second, 100*time.Millisecond, "Failed to establish initial cluster state")
			}

			// Execute the leave operation
			err := nodes[testCase.testNode].LeaveCluster(context.Background())

			// Verify the result
			if testCase.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)

				// For multi-node clusters, verify the remaining cluster state
				if len(nodes) > 1 {
					// Verify the node is no longer in the cluster configuration
					for i, node := range nodes {
						if i != testCase.testNode && node != nil {
							config := node.GetPeers()
							for _, server := range config {
								assert.NotEqual(t, nodes[testCase.testNode].config.LocalID, server.ID,
									"Departed node should not be in cluster configuration")
							}
						}
					}
				}
			}
		})
	}
}
