package network

import (
	"context"
	"encoding/json"
	"net"
	"sync"
	"testing"
	"time"

	"github.com/gatewayd-io/gatewayd/config"
	"github.com/gatewayd-io/gatewayd/raft"
	"github.com/gatewayd-io/gatewayd/testhelpers"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestNewConsistentHash verifies that a new ConsistentHash instance is properly created.
// It checks that the original load balancing strategy is preserved, that the useSourceIp
// setting is correctly applied, and that the hashMap is initialized.
func TestNewConsistentHash(t *testing.T) {
	server := &Server{
		LoadbalancerConsistentHash: &config.ConsistentHash{UseSourceIP: true},
	}
	originalStrategy := NewRandom(server)
	consistentHash := NewConsistentHash(server, originalStrategy)

	assert.NotNil(t, consistentHash)
	assert.Equal(t, originalStrategy, consistentHash.originalStrategy)
	assert.True(t, consistentHash.useSourceIP)
}

// TestConsistentHashNextProxyUseSourceIpExists tests the consistent hash load balancer
// when useSourceIP is enabled. It verifies that:
// 1. A connection from a specific IP is correctly hashed
// 2. The hash mapping is properly stored in the Raft FSM
// 3. The correct proxy is returned based on the stored mapping
// The test uses a mock connection and Raft node to simulate a distributed environment.
func TestConsistentHashNextProxyUseSourceIpExists(t *testing.T) {
	proxies := []IProxy{
		MockProxy{name: "proxy1", groupName: "test-group"},
		MockProxy{name: "proxy2", groupName: "test-group"},
		MockProxy{name: "proxy3", groupName: "test-group"},
	}

	raftHelper, err := testhelpers.NewTestRaftNode(t)
	if err != nil {
		t.Fatalf("Failed to create test raft node: %v", err)
	}
	defer func() {
		if err := raftHelper.Cleanup(); err != nil {
			t.Errorf("Failed to cleanup raft: %v", err)
		}
	}()

	server := &Server{
		Proxies:                    proxies,
		LoadbalancerConsistentHash: &config.ConsistentHash{UseSourceIP: true},
		RaftNode:                   raftHelper.Node,
		GroupName:                  "test-group",
	}
	server.initializeProxies()
	originalStrategy := NewRandom(server)
	consistentHash := NewConsistentHash(server, originalStrategy)
	mockConn := new(MockConnWrapper)

	// Mock RemoteAddr to return a specific IP:port format
	mockAddr := &net.TCPAddr{IP: net.ParseIP("192.168.1.1"), Port: 1234}
	mockConn.On("RemoteAddr").Return(mockAddr)

	// Instead of setting hashMap directly, setup the FSM
	hash := hashKey("192.168.1.1" + server.GroupName)
	// Create and apply the command through Raft
	cmd := raft.Command{
		Type: raft.CommandAddConsistentHashEntry,
		Payload: raft.ConsistentHashPayload{
			Hash:      hash,
			BlockName: proxies[2].GetBlockName(),
		},
	}

	cmdBytes, marshalErr := json.Marshal(cmd)
	require.NoError(t, marshalErr)

	// Apply the command through Raft
	if err := server.RaftNode.Apply(context.Background(), cmdBytes, 10*time.Second); err != nil {
		require.NoError(t, err)
	}

	proxy, err := consistentHash.NextProxy(mockConn)
	assert.Nil(t, err)
	assert.Equal(t, proxies[2], proxy)

	// Clean up
	mockConn.AssertExpectations(t)
}

// TestConsistentHashNextProxyUseFullAddress verifies the behavior when useSourceIp is disabled.
// It ensures that the full connection address (IP:port) plus group name is used for hashing,
// and the correct proxy is returned. The test also verifies that the proxy mapping is properly
// stored in the Raft FSM for persistence and cluster-wide consistency.
func TestConsistentHashNextProxyUseFullAddress(t *testing.T) {
	mockConn := new(MockConnWrapper)
	proxies := []IProxy{
		MockProxy{name: "proxy1", groupName: "test-group"},
		MockProxy{name: "proxy2", groupName: "test-group"},
		MockProxy{name: "proxy3", groupName: "test-group"},
	}
	raftHelper, err := testhelpers.NewTestRaftNode(t)
	if err != nil {
		t.Fatalf("Failed to create test raft node: %v", err)
	}
	defer func() {
		if err := raftHelper.Cleanup(); err != nil {
			t.Errorf("Failed to cleanup raft: %v", err)
		}
	}()
	server := &Server{
		Proxies: proxies,
		LoadbalancerConsistentHash: &config.ConsistentHash{
			UseSourceIP: false,
		},
		RaftNode:  raftHelper.Node,
		GroupName: "test-group",
	}
	mockStrategy := NewRoundRobin(server)

	// Mock RemoteAddr to return full address
	mockAddr := &net.TCPAddr{IP: net.ParseIP("192.168.1.1"), Port: 1234}
	mockConn.On("RemoteAddr").Return(mockAddr)

	server.initializeProxies()

	consistentHash := NewConsistentHash(server, mockStrategy)

	proxy, err := consistentHash.NextProxy(mockConn)
	assert.Nil(t, err)
	assert.NotNil(t, proxy)
	assert.Equal(t, proxies[1], proxy)

	// Verify the hash was stored in Raft FSM
	hash := hashKey("192.168.1.1:1234" + server.GroupName)
	blockName, exists := server.RaftNode.Fsm.GetProxyBlock(hash)
	assert.True(t, exists)
	assert.Equal(t, proxies[1].GetBlockName(), blockName)

	// Clean up
	mockConn.AssertExpectations(t)
}

// TestConsistentHashNextProxyConcurrency tests the thread safety and consistency of the NextProxy method
// in a distributed environment. It verifies that:
// 1. Multiple concurrent requests from the same IP address consistently map to the same proxy
// 2. The mapping is stable across sequential calls
// 3. Different IP addresses map to different proxies
// 4. The Raft-based consistent hash remains thread-safe under high concurrency.
func TestConsistentHashNextProxyConcurrency(t *testing.T) {
	// Setup mocks
	conn1 := new(MockConnWrapper)
	conn2 := new(MockConnWrapper)
	proxies := []IProxy{
		MockProxy{name: "proxy1", groupName: "test-group"},
		MockProxy{name: "proxy2", groupName: "test-group"},
		MockProxy{name: "proxy3", groupName: "test-group"},
	}
	raftHelper, err := testhelpers.NewTestRaftNode(t)
	if err != nil {
		t.Fatalf("Failed to create test raft node: %v", err)
	}
	defer func() {
		if err := raftHelper.Cleanup(); err != nil {
			t.Errorf("Failed to cleanup raft: %v", err)
		}
	}()
	server := &Server{
		Proxies:                    proxies,
		LoadbalancerConsistentHash: &config.ConsistentHash{UseSourceIP: true},
		RaftNode:                   raftHelper.Node,
		GroupName:                  "test-group",
	}
	originalStrategy := NewRoundRobin(server)

	// Mock IP addresses
	mockAddr1 := &net.TCPAddr{IP: net.ParseIP("192.168.1.1"), Port: 1234}
	mockAddr2 := &net.TCPAddr{IP: net.ParseIP("192.168.1.2"), Port: 1234}
	conn1.On("RemoteAddr").Return(mockAddr1)
	conn2.On("RemoteAddr").Return(mockAddr2)

	server.initializeProxies()

	// Initialize the ConsistentHash
	consistentHash := NewConsistentHash(server, originalStrategy)

	// Run the test concurrently
	var waitGroup sync.WaitGroup
	const numGoroutines = 100

	for range numGoroutines {
		waitGroup.Add(1)
		go func() {
			defer waitGroup.Done()
			p, err := consistentHash.NextProxy(conn1)
			assert.Nil(t, err)
			assert.Equal(t, proxies[1], p)
		}()
	}

	waitGroup.Wait()

	// Ensure that the proxy is consistently the same
	proxy, err := consistentHash.NextProxy(conn1)
	assert.Nil(t, err)
	assert.Equal(t, proxies[1], proxy)

	// Ensure that connecting from a different address returns a different proxy
	proxy, err = consistentHash.NextProxy(conn2)
	assert.Nil(t, err)
	assert.Equal(t, proxies[2], proxy)
}
