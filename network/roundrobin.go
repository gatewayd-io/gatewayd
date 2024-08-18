package network

import (
	"errors"
	"net"
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

func (r *RoundRobin) NextProxy(conn net.Conn) (IProxy, *gerr.GatewayDError) {
	proxiesLen := uint32(len(r.proxies))
	if proxiesLen == 0 {
		return nil, gerr.ErrNoProxiesAvailable.Wrap(errors.New("proxy list is empty"))
	}
	nextIndex := r.next.Add(1)
	return r.proxies[nextIndex%proxiesLen], nil
}
