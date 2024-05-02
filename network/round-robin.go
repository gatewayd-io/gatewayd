package network

import (
	"container/ring"
	"context"
)

type RoundRobinStrategy struct {
	// Atomic circular linked list used to store server's proxies and getting the next proxy on each connections
	DistributionStrategy
	proxies *ring.Ring
}

func NewRoundRobin(ctx context.Context, server *Server) *RoundRobinStrategy {
	// Create a circular linked list of proxies length
	proxies := ring.New(len(server.Proxies))

	// Initialize the ring with server's proxies
	for i := 0; i < proxies.Len(); i++ {
		proxies.Value = i
		proxies = proxies.Next()
	}
	return &RoundRobinStrategy{proxies: proxies}
}

func (ds RoundRobinStrategy) Distribute(wrapper *ConnWrapper) (IProxy, error) {
	// Set proxy the next one
	defer ds.proxies.Next()

	return ds.proxies.Value.(IProxy), nil
}
