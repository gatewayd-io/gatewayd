package network

import (
	"errors"
	"sync"

	"github.com/gatewayd-io/gatewayd/config"
	gerr "github.com/gatewayd-io/gatewayd/errors"
)

type weightedProxy struct {
	proxy           IProxy
	currentWeight   int
	effectiveWeight int
}

type WeightedRoundRobin struct {
	proxies     []*weightedProxy
	totalWeight int
	mu          sync.Mutex
}

// NewWeightedRoundRobin creates a new WeightedRoundRobin load balancer.
// It initializes the weighted proxies based on the distribution rules defined in the load balancer configuration.
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
//
// It adjusts the current weight of each proxy based on its effective weight and selects
// the proxy with the highest current weight. The selected proxy's current weight is then
// decreased by the total weight of all proxies to ensure balanced distribution over time.
func (r *WeightedRoundRobin) NextProxy(_ IConnWrapper) (IProxy, *gerr.GatewayDError) {
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

// findProxyByName locates a proxy by its name in the provided list of proxies.
func findProxyByName(name string, proxies []IProxy) IProxy {
	for _, proxy := range proxies {
		if proxy.GetBlockName() == name {
			return proxy
		}
	}
	return nil
}
