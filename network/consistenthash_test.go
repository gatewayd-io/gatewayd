package network

import (
	"net"
	"sync"
	"testing"

	"github.com/gatewayd-io/gatewayd/config"
	"github.com/stretchr/testify/assert"
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
	assert.NotNil(t, consistentHash.hashMap)
}

// TestConsistentHashNextProxyUseSourceIpExists ensures that when useSourceIp is enabled,
// and the hashed IP exists in the hashMap, the correct proxy is returned.
// It mocks a connection with a specific IP and verifies the proxy retrieval from the hashMap.
func TestConsistentHashNextProxyUseSourceIpExists(t *testing.T) {
	proxies := []IProxy{
		MockProxy{name: "proxy1"},
		MockProxy{name: "proxy2"},
		MockProxy{name: "proxy3"},
	}
	server := &Server{
		Proxies:                    proxies,
		LoadbalancerConsistentHash: &config.ConsistentHash{UseSourceIP: true},
	}
	originalStrategy := NewRandom(server)
	consistentHash := NewConsistentHash(server, originalStrategy)
	mockConn := new(MockConnWrapper)

	// Mock RemoteAddr to return a specific IP:port format
	mockAddr := &net.TCPAddr{IP: net.ParseIP("192.168.1.1"), Port: 1234}
	mockConn.On("RemoteAddr").Return(mockAddr)

	key := "192.168.1.1"
	hash := hashKey(key)

	consistentHash.hashMap[hash] = proxies[2]

	proxy, err := consistentHash.NextProxy(mockConn)
	assert.Nil(t, err)
	assert.Equal(t, proxies[2], proxy)

	// Clean up
	mockConn.AssertExpectations(t)
}

// TestConsistentHashNextProxyUseFullAddress verifies the behavior when useSourceIp is disabled.
// It ensures that the full connection address is used for hashing, and the correct proxy is returned
// and cached in the hashMap. The test also checks that the hash value is computed based on the full address.
func TestConsistentHashNextProxyUseFullAddress(t *testing.T) {
	mockConn := new(MockConnWrapper)
	proxies := []IProxy{
		MockProxy{name: "proxy1"},
		MockProxy{name: "proxy2"},
		MockProxy{name: "proxy3"},
	}
	server := &Server{
		Proxies: proxies,
		LoadbalancerConsistentHash: &config.ConsistentHash{
			UseSourceIP: false,
		},
	}
	mockStrategy := NewRoundRobin(server)

	// Mock RemoteAddr to return full address
	mockAddr := &net.TCPAddr{IP: net.ParseIP("192.168.1.1"), Port: 1234}
	mockConn.On("RemoteAddr").Return(mockAddr)

	consistentHash := NewConsistentHash(server, mockStrategy)

	proxy, err := consistentHash.NextProxy(mockConn)
	assert.Nil(t, err)
	assert.NotNil(t, proxy)
	assert.Equal(t, proxies[1], proxy)

	// Hash should be calculated using the full address and cached in hashMap
	hash := hashKey("192.168.1.1:1234")
	cachedProxy, exists := consistentHash.hashMap[hash]

	assert.True(t, exists)
	assert.Equal(t, proxies[1], cachedProxy)

	// Clean up
	mockConn.AssertExpectations(t)
}

// TestConsistentHashNextProxyConcurrency tests the concurrency safety of the NextProxy method
// in the ConsistentHash struct. It ensures that multiple goroutines can concurrently call
// NextProxy without causing race conditions or inconsistent behavior.
func TestConsistentHashNextProxyConcurrency(t *testing.T) {
	// Setup mocks
	conn1 := new(MockConnWrapper)
	conn2 := new(MockConnWrapper)
	proxies := []IProxy{
		MockProxy{name: "proxy1"},
		MockProxy{name: "proxy2"},
		MockProxy{name: "proxy3"},
	}
	server := &Server{
		Proxies:                    proxies,
		LoadbalancerConsistentHash: &config.ConsistentHash{UseSourceIP: true},
	}
	originalStrategy := NewRoundRobin(server)

	// Mock IP addresses
	mockAddr1 := &net.TCPAddr{IP: net.ParseIP("192.168.1.1"), Port: 1234}
	mockAddr2 := &net.TCPAddr{IP: net.ParseIP("192.168.1.2"), Port: 1234}
	conn1.On("RemoteAddr").Return(mockAddr1)
	conn2.On("RemoteAddr").Return(mockAddr2)

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
