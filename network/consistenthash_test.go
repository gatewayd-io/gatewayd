package network

import (
	"net"
	"testing"

	"github.com/gatewayd-io/gatewayd/config"
	"github.com/stretchr/testify/assert"
)

func TestNewConsistentHash(t *testing.T) {
	server := &Server{
		LoadbalancerConsistentHash: &config.ConsistentHash{UseSourceIp: true},
	}
	originalStrategy := NewRandom(server)
	ch := NewConsistentHash(server, originalStrategy)

	assert.NotNil(t, ch)
	assert.Equal(t, originalStrategy, ch.originalStrategy)
	assert.True(t, ch.useSourceIp)
	assert.NotNil(t, ch.hashMap)
}

func TestConsistentHashNextProxyUseSourceIpExists(t *testing.T) {
	proxies := []IProxy{
		MockProxy{name: "proxy1"},
		MockProxy{name: "proxy2"},
		MockProxy{name: "proxy3"},
	}
	server := &Server{
		Proxies:                    proxies,
		LoadbalancerConsistentHash: &config.ConsistentHash{UseSourceIp: true},
	}
	originalStrategy := NewRandom(server)
	ch := NewConsistentHash(server, originalStrategy)
	mockConn := new(MockConnWrapper)

	// Mock RemoteAddr to return a specific IP:port format
	mockAddr := &net.TCPAddr{IP: net.ParseIP("192.168.1.1"), Port: 1234}
	mockConn.On("RemoteAddr").Return(mockAddr)

	key := "192.168.1.1"
	hash := hashKey(key)

	ch.hashMap[hash] = proxies[2]

	proxy, err := ch.NextProxy(mockConn)
	assert.Nil(t, err)
	assert.Equal(t, proxies[2], proxy)

	// Clean up
	mockConn.AssertExpectations(t)
}

// Test case for full address-based hashing
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
			UseSourceIp: false,
		},
	}
	mockStrategy := NewRoundRobin(server)

	// Mock RemoteAddr to return full address
	mockAddr := &net.TCPAddr{IP: net.ParseIP("192.168.1.1"), Port: 1234}
	mockConn.On("RemoteAddr").Return(mockAddr)

	ch := NewConsistentHash(server, mockStrategy)

	proxy, err := ch.NextProxy(mockConn)
	assert.Nil(t, err)
	assert.NotNil(t, proxy)
	assert.Equal(t, proxies[1], proxy)

	// Hash should be calculated using the full address and cached in hashMap
	hash := hashKey("192.168.1.1:1234")
	ch.hashMapMutex.RLock()
	cachedProxy, exists := ch.hashMap[hash]
	ch.hashMapMutex.RUnlock()

	assert.True(t, exists)
	assert.Equal(t, proxies[1], cachedProxy)

	// Clean up
	mockConn.AssertExpectations(t)
}
