package network

import (
	"context"
	"encoding/json"
	"math"
	"sync"

	gerr "github.com/gatewayd-io/gatewayd/errors"
	"github.com/gatewayd-io/gatewayd/raft"
)

type RoundRobin struct {
	proxies []IProxy
	server  *Server
	mu      sync.Mutex
}

// NewRoundRobin creates a new RoundRobin load balancer.
func NewRoundRobin(server *Server) *RoundRobin {
	return &RoundRobin{proxies: server.Proxies, server: server}
}

// NextProxy returns the next proxy in the round-robin sequence.
func (r *RoundRobin) NextProxy(_ IConnWrapper) (IProxy, *gerr.GatewayDError) {
	if len(r.proxies) > math.MaxUint32 {
		// This should never happen, but if it does, we fall back to the first proxy.
		return r.proxies[0], nil
	} else if len(r.proxies) == 0 {
		return nil, gerr.ErrNoProxiesAvailable
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	// Get current index from Raft FSM
	currentIndex := r.server.RaftNode.Fsm.GetRoundRobinNext(r.server.GroupName)
	nextIndex := currentIndex + 1

	// Create Raft command
	cmd := raft.Command{
		Type: raft.CommandAddRoundRobinNext,
		Payload: raft.RoundRobinPayload{
			NextIndex: nextIndex,
			GroupName: r.server.GroupName,
		},
	}

	// Convert command to JSON
	data, err := json.Marshal(cmd)
	if err != nil {
		return nil, gerr.ErrNoProxiesAvailable.Wrap(err)
	}

	// Apply through Raft
	if err := r.server.RaftNode.Apply(context.Background(), data, raft.ApplyTimeout); err != nil {
		return nil, gerr.ErrNoProxiesAvailable.Wrap(err)
	}

	return r.proxies[nextIndex%uint32(len(r.proxies))], nil //nolint:gosec
}
