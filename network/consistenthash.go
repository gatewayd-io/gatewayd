package network

import (
	"net"
	"sync"

	gerr "github.com/gatewayd-io/gatewayd/errors"
	"github.com/spaolacci/murmur3"
)

type ConsistentHash struct {
	originalStrategy LoadBalancerStrategy
	useSourceIp      bool
	hashMap          map[uint64]IProxy
	hashMapMutex     sync.RWMutex
}

func NewConsistentHash(server *Server, originalStrategy LoadBalancerStrategy) *ConsistentHash {
	return &ConsistentHash{
		originalStrategy: originalStrategy,
		useSourceIp:      server.LoadbalancerConsistentHash.UseSourceIp,
		hashMap:          make(map[uint64]IProxy),
	}
}

func (ch *ConsistentHash) NextProxy(conn IConnWrapper) (IProxy, *gerr.GatewayDError) {
	var key string

	if ch.useSourceIp {
		sourceIP, err := extractIPFromConn(conn)
		if err != nil {
			return nil, gerr.ErrNoProxiesAvailable.Wrap(err)
		}
		key = sourceIP
	} else {
		key = conn.RemoteAddr().String() // Fallback to use full address as the key if `useSourceIp` is false
	}

	hash := hashKey(key)

	ch.hashMapMutex.RLock()
	proxy, exists := ch.hashMap[hash]
	ch.hashMapMutex.RUnlock()

	if exists {
		return proxy, nil
	}

	// If no hash exists, fallback to the original strategy
	proxy, err := ch.originalStrategy.NextProxy(conn)
	if err != nil {
		return nil, gerr.ErrNoProxiesAvailable.Wrap(err)
	}

	// Add the selected proxy to the hash map for future requests
	ch.hashMapMutex.Lock()
	ch.hashMap[hash] = proxy
	ch.hashMapMutex.Unlock()

	return proxy, nil
}

// hash function using MurmurHash3
func hashKey(key string) uint64 {
	return murmur3.Sum64([]byte(key))
}

// extractIPFromConn extracts only the IP address from the RemoteAddr
func extractIPFromConn(con IConnWrapper) (string, error) {
	addr := con.RemoteAddr().String()
	// addr will be in the format "IP:port"
	ip, _, err := net.SplitHostPort(addr)
	if err != nil {
		return "", err
	}
	return ip, nil
}
