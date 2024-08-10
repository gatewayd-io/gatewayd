package network

import (
	"errors"
	"sync"

	"github.com/gatewayd-io/gatewayd/config"
	gerr "github.com/gatewayd-io/gatewayd/errors"
)

type weightedProxy struct {
	proxy           IProxy
	weight          int
	currentWeight   int
	effectiveWeight int
}

type WeightedRoundRobin struct {
	proxies     []*weightedProxy
	totalWeight int
	mu          sync.Mutex
}

func NewWeightedRoundRobin(server *Server, loadbalancerRule config.LoadBalancingRule) *WeightedRoundRobin {
	var proxies []*weightedProxy
	totalWeight := 0

	for _, distribution := range loadbalancerRule.Distribution {
		proxy := findProxyByName(distribution.ProxyName, server.Proxies)
		if proxy != nil {
			weight := distribution.Weight
			totalWeight += weight
			proxies = append(proxies, &weightedProxy{
				proxy:           proxy,
				weight:          weight,
				effectiveWeight: weight,
				currentWeight:   0,
			})
		}
	}

	return &WeightedRoundRobin{
		proxies:     proxies,
		totalWeight: totalWeight,
	}
}

// NextProxy selects the next proxy based on the weighted round-robin algorithm.
func (r *WeightedRoundRobin) NextProxy() (IProxy, *gerr.GatewayDError) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if len(r.proxies) == 0 {
		return nil, gerr.ErrNoProxiesAvailable.Wrap(errors.New("proxy list is empty"))
	}

	var selected *weightedProxy

	// Adjust weights and select the proxy with the highest current weight.
	for _, p := range r.proxies {
		p.currentWeight += p.effectiveWeight
		if selected == nil || p.currentWeight > selected.currentWeight {
			selected = p
		}
	}

	// Reduce the selected proxy's current weight by the total weight.
	if selected != nil {
		selected.currentWeight -= r.totalWeight
		return selected.proxy, nil
	}

	return nil, gerr.ErrNoProxiesAvailable.Wrap(errors.New("no proxy selected"))
}

// findProxyByName finds a proxy by name from the list of proxies.
func findProxyByName(name string, proxies []IProxy) IProxy {
	for _, proxy := range proxies {
		if proxy.GetName() == name {
			return proxy
		}
	}
	return nil
}
