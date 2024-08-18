package network

import (
	"net"

	gerr "github.com/gatewayd-io/gatewayd/errors"
	"github.com/rs/zerolog"
	"github.com/spaolacci/murmur3"
)

type ConsistentHash struct {
	originalStrategy LoadBalancerStrategy
	useSourceIp      bool
	hashMap          map[uint64]IProxy
	Logger           zerolog.Logger
}

func NewConsistentHash(server *Server, originalStrategy LoadBalancerStrategy) *ConsistentHash {
	return &ConsistentHash{
		originalStrategy: originalStrategy,
		useSourceIp:      server.LoadbalancerConsistentHash.UseSourceIp,
		hashMap:          make(map[uint64]IProxy),
		Logger:           server.Logger,
	}
}

func (ch *ConsistentHash) NextProxy(conn net.Conn) (IProxy, *gerr.GatewayDError) {
	var key string
	if ch.useSourceIp {
		sourceIP, err := extractIPFromConn(conn)
		if err != nil {
			gerr.ErrNoProxiesAvailable.Wrap(err)
		}
		key = sourceIP
	}

	hash := hashKey(key)

	if proxy, exists := ch.hashMap[hash]; exists {
		return proxy, nil
	}

	// If no hash exists, fallback to the original strategy
	proxy, err := ch.originalStrategy.NextProxy(conn)
	if err != nil {
		return nil, err
	}

	// Optionally add the selected proxy to the hash map for future requests
	ch.hashMap[hash] = proxy
	return proxy, nil
}

// hash function using MurmurHash3
func hashKey(key string) uint64 {
	return murmur3.Sum64([]byte(key))
}

// extractIPFromConn extract only the IP address from the RemoteAddr
func extractIPFromConn(conn net.Conn) (string, error) {
	addr := conn.RemoteAddr().String()
	// addr will be in the format "IP:port"
	ip, _, err := net.SplitHostPort(addr)
	if err != nil {
		return "", err
	}
	return ip, nil
}
