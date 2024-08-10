package network

import (
	"errors"
	"sync/atomic"

	"github.com/gatewayd-io/gatewayd/config"
	gerr "github.com/gatewayd-io/gatewayd/errors"
)

type WeightedRoundRobin struct {
	proxies []IProxy
	next    atomic.Uint32
}

func NewWeightedRoundRobin(server *Server, loadbalancerRule config.LoadBalancingRule) *WeightedRoundRobin {
	var proxies []IProxy

	// Populate the cyclic list of proxies according to their weights.
	for _, distribution := range loadbalancerRule.Distribution {
		proxy := findProxyByName(distribution.ProxyName, server.Proxies)
		if proxy != nil {
			for i := 0; i < distribution.Weight; i++ {
				proxies = append(proxies, proxy)
			}
		}
	}

	return &WeightedRoundRobin{
		proxies: proxies,
	}
}

func (r *WeightedRoundRobin) NextProxy() (IProxy, *gerr.GatewayDError) {
	proxiesLen := uint32(len(r.proxies))
	if proxiesLen == 0 {
		return nil, gerr.ErrNoProxiesAvailable.Wrap(errors.New("proxy list is empty"))
	}
	nextIndex := r.next.Add(1)
	return r.proxies[nextIndex%proxiesLen], nil
}

func findProxyByName(name string, proxies []IProxy) IProxy {
	for _, proxy := range proxies {
		if proxy.GetName() == name {
			return proxy
		}
	}
	return nil
}
