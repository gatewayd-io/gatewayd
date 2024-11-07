package network

import (
	"fmt"
	"net"
	"sync"
	"time"

	"encoding/json"

	gerr "github.com/gatewayd-io/gatewayd/errors"
	"github.com/gatewayd-io/gatewayd/raft"
	"github.com/spaolacci/murmur3"
)

// ConsistentHash implements a load balancing strategy based on consistent hashing.
// It routes client connections to specific proxies by hashing the client's IP address or the full connection address.
type ConsistentHash struct {
	originalStrategy LoadBalancerStrategy
	useSourceIP      bool
	mu               sync.Mutex
	raftNode         *raft.RaftNode
	server           *Server
}

// NewConsistentHash creates a new ConsistentHash instance. It requires a server configuration and an original
// load balancing strategy. The consistent hash can use either the source IP or the full connection address
// as the key for hashing.
func NewConsistentHash(server *Server, originalStrategy LoadBalancerStrategy, raftNode *raft.RaftNode) *ConsistentHash {
	return &ConsistentHash{
		originalStrategy: originalStrategy,
		useSourceIP:      server.LoadbalancerConsistentHash.UseSourceIP,
		raftNode:         raftNode,
		server:           server,
	}
}

// NextProxy selects the appropriate proxy for a given client connection. It first tries to find an existing
// proxy in the hash map based on the hashed key (either the source IP or the full address). If no match is found,
// it falls back to the original load balancing strategy, adds the selected proxy to the hash map, and returns it.
func (ch *ConsistentHash) NextProxy(conn IConnWrapper) (IProxy, *gerr.GatewayDError) {
	ch.mu.Lock()
	defer ch.mu.Unlock()

	var key string

	if ch.useSourceIP {
		sourceIP, err := extractIPFromConn(conn)
		if err != nil {
			return nil, gerr.ErrNoProxiesAvailable.Wrap(err)
		}
		key = sourceIP
	} else {
		// Fallback to using the full remote address (IP:port) as the key if `useSourceIP` is false.
		// This effectively disables consistent hashing, as the remote address has a random port each time.
		key = conn.RemoteAddr().String()
	}

	hash := hashKey(key)

	proxyID, exists := ch.raftNode.Fsm.GetProxyID(hash)
	if exists {
		if proxy, ok := ch.server.GetProxyByID(proxyID); ok {
			return proxy, nil
		}
	}

	// If no hash exists, fallback to the original strategy
	proxy, err := ch.originalStrategy.NextProxy(conn)
	if err != nil {
		return nil, gerr.ErrNoProxiesAvailable.Wrap(err)
	}

	// Create and apply the command through Raft
	cmd := raft.HashMapCommand{
		Type:    raft.CommandAddHashMapping,
		Hash:    hash,
		ProxyID: proxy.GetID(),
	}

	cmdBytes, marshalErr := json.Marshal(cmd)
	if marshalErr != nil {
		return nil, gerr.ErrNoProxiesAvailable.Wrap(marshalErr)
	}

	// Apply the command through Raft
	if err := ch.raftNode.Apply(cmdBytes, 10*time.Second); err != nil {
		return nil, gerr.ErrNoProxiesAvailable.Wrap(err)
	}

	return proxy, nil
}

// hashKey hashes a given key using the MurmurHash3 algorithm. It is used to generate consistent hash values
// for IP addresses or connection strings.
func hashKey(key string) uint64 {
	return murmur3.Sum64([]byte(key))
}

// extractIPFromConn extracts the IP address from the connection's remote address. It splits the address
// into IP and port components and returns the IP part. This is useful for hashing based on the source IP.
func extractIPFromConn(con IConnWrapper) (string, error) {
	addr := con.RemoteAddr().String() // RemoteAddr is the address of the request, LocalAddress is the gateway address.
	// addr will be in the format "IP:port"
	ip, _, err := net.SplitHostPort(addr)
	if err != nil {
		return "", fmt.Errorf("failed to split host and port from address %s: %w", addr, err)
	}
	return ip, nil
}
