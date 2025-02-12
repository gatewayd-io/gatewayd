package network

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"strconv"
	"sync"

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
	server           *Server
}

// NewConsistentHash creates a new ConsistentHash instance. It requires a server configuration and an original
// load balancing strategy. The consistent hash can use either the source IP or the full connection address
// as the key for hashing.
func NewConsistentHash(server *Server, originalStrategy LoadBalancerStrategy) *ConsistentHash {
	return &ConsistentHash{
		originalStrategy: originalStrategy,
		useSourceIP:      server.LoadbalancerConsistentHash.UseSourceIP,
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

	hash := hashKey(key + ch.server.GroupName)

	// Get block name for this hash
	blockName, exists := ch.server.RaftNode.Fsm.GetProxyBlock(hash)
	if exists {
		if proxy, ok := ch.server.ProxyByBlock[blockName]; ok {
			return proxy, nil
		}
	}

	// If no hash exists or no matching proxy found, fallback to the original strategy
	proxy, err := ch.originalStrategy.NextProxy(conn)
	if err != nil {
		return nil, gerr.ErrNoProxiesAvailable.Wrap(err)
	}

	// Create and apply the command through Raft
	cmd := raft.Command{
		Type: raft.CommandAddConsistentHashEntry,
		Payload: raft.ConsistentHashPayload{
			Hash:      hash,
			BlockName: proxy.GetBlockName(),
		},
	}

	cmdBytes, marshalErr := json.Marshal(cmd)
	if marshalErr != nil {
		return nil, gerr.ErrNoProxiesAvailable.Wrap(marshalErr)
	}

	// Apply the command through Raft
	if err := ch.server.RaftNode.Apply(context.Background(), cmdBytes, raft.ApplyTimeout); err != nil {
		return nil, gerr.ErrNoProxiesAvailable.Wrap(err)
	}

	return proxy, nil
}

// hashKey hashes a given key using the MurmurHash3 algorithm and returns it as a string. It is used to generate
// consistent hash values for IP addresses or connection strings.
func hashKey(key string) string {
	hash := murmur3.Sum64([]byte(key))
	return strconv.FormatUint(hash, 10)
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
