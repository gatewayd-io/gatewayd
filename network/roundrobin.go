package network

import (
	"math"
	"sync/atomic"

	gerr "github.com/gatewayd-io/gatewayd/errors"
)

type RoundRobin struct {
	proxies []IProxy
	next    atomic.Uint32
}

func NewRoundRobin(server *Server) *RoundRobin {
	return &RoundRobin{proxies: server.Proxies}
}

func (r *RoundRobin) NextProxy(_ IConnWrapper) (IProxy, *gerr.GatewayDError) {
	if len(r.proxies) > math.MaxUint32 {
		// This should never happen, but if it does, we fall back to the first proxy.
		return r.proxies[0], nil
	} else if len(r.proxies) == 0 {
		return nil, gerr.ErrNoProxiesAvailable
	}

	nextIndex := r.next.Add(1)
	return r.proxies[nextIndex%uint32(len(r.proxies))], nil //nolint:gosec
}
