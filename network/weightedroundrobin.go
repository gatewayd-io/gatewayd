package network

import (
	"context"
	"encoding/json"
	"errors"
	"sync"

	"github.com/gatewayd-io/gatewayd/config"
	gerr "github.com/gatewayd-io/gatewayd/errors"
	"github.com/gatewayd-io/gatewayd/raft"
)

type WeightedRoundRobin struct {
	groupName      string
	proxies        []IProxy
	raftNode       *raft.Node
	mu             sync.Mutex
	initialWeights map[string]int
}

// NewWeightedRoundRobin creates a new WeightedRoundRobin load balancer.
// It initializes the weighted proxies based on the distribution rules defined in the load balancer configuration.
func NewWeightedRoundRobin(server *Server, loadbalancerRule config.LoadBalancingRule) *WeightedRoundRobin {
	var proxies []IProxy
	initialWeights := make(map[string]int)
	for _, distribution := range loadbalancerRule.Distribution {
		proxy := findProxyByName(distribution.ProxyName, server.Proxies)
		if proxy != nil {
			proxies = append(proxies, proxy)
			initialWeights[proxy.GetBlockName()] = distribution.Weight
		}
	}

	return &WeightedRoundRobin{
		groupName:      server.GroupName,
		proxies:        proxies,
		raftNode:       server.RaftNode,
		initialWeights: initialWeights,
	}
}

// NextProxy selects the next proxy based on the weighted round-robin algorithm.
//
// It adjusts the current weight of each proxy based on its effective weight and selects
// the proxy with the highest current weight. The selected proxy's current weight is then
// decreased by the total weight of all proxies to ensure balanced distribution over time.
func (r *WeightedRoundRobin) NextProxy(ctx context.Context, _ IConnWrapper) (IProxy, *gerr.GatewayDError) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if len(r.proxies) == 0 {
		return nil, gerr.ErrNoProxiesAvailable.Wrap(errors.New("proxy list is empty"))
	}

	var selected IProxy
	var maxWeight int
	totalWeight := 0

	// Prepare batch updates
	updates := make(map[string]raft.WeightedProxy)

	// Calculate weights and find the selected proxy
	for _, proxy := range r.proxies {
		proxyName := proxy.GetBlockName()
		weight, exists := r.raftNode.Fsm.GetWeightedRRState(r.groupName, proxyName)
		if !exists {
			weight = raft.WeightedProxy{
				EffectiveWeight: r.initialWeights[proxyName],
				CurrentWeight:   0,
			}
		}

		totalWeight += weight.EffectiveWeight
		newWeight := weight.CurrentWeight + weight.EffectiveWeight

		// Store the update
		updates[proxyName] = raft.WeightedProxy{
			CurrentWeight:   newWeight,
			EffectiveWeight: weight.EffectiveWeight,
		}

		if selected == nil || newWeight > maxWeight {
			selected = proxy
			maxWeight = newWeight
		}
	}

	if selected == nil {
		return nil, gerr.ErrNoProxiesAvailable.Wrap(errors.New("no proxy selected"))
	}

	// Update the selected proxy's weight
	selectedProxy := updates[selected.GetBlockName()]
	selectedProxy.CurrentWeight -= totalWeight
	updates[selected.GetBlockName()] = selectedProxy

	// Create and apply the batch update command
	cmd := raft.Command{
		Type: raft.CommandUpdateWeightedRRBatch,
		Payload: raft.WeightedRRBatchPayload{
			GroupName: r.groupName,
			Updates:   updates,
		},
	}

	data, err := json.Marshal(cmd)
	if err != nil {
		return nil, gerr.ErrNoProxiesAvailable.Wrap(err)
	}
	if err := r.raftNode.Apply(ctx, data, raft.ApplyTimeout); err != nil {
		return nil, gerr.ErrNoProxiesAvailable.Wrap(err)
	}

	return selected, nil
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
