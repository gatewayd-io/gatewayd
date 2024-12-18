package network

import (
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
func (r *WeightedRoundRobin) NextProxy(_ IConnWrapper) (IProxy, *gerr.GatewayDError) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if len(r.proxies) == 0 {
		return nil, gerr.ErrNoProxiesAvailable.Wrap(errors.New("proxy list is empty"))
	}

	var selected IProxy
	var maxWeight int
	totalWeight := 0

	// Calculate total weight and find proxy with highest current weight
	for _, proxy := range r.proxies {
		weight, exists := r.raftNode.Fsm.GetWeightedRRState(r.groupName, proxy.GetBlockName())
		if !exists {
			weight = raft.WeightedProxy{
				EffectiveWeight: r.initialWeights[proxy.GetBlockName()],
				CurrentWeight:   0,
			}
		}

		totalWeight += weight.EffectiveWeight

		// Update current weight in Raft
		newWeight := weight.CurrentWeight + weight.EffectiveWeight
		cmd := raft.Command{
			Type: raft.CommandUpdateWeightedRR,
			Payload: raft.WeightedRRPayload{
				GroupName: r.groupName,
				ProxyName: proxy.GetBlockName(),
				Weight: raft.WeightedProxy{
					CurrentWeight:   newWeight,
					EffectiveWeight: weight.EffectiveWeight,
				},
			},
		}
		data, _ := json.Marshal(cmd)
		r.raftNode.Apply(data, raft.LeaderElectionTimeout)

		if selected == nil || newWeight > maxWeight {
			selected = proxy
			maxWeight = newWeight
		}
	}

	if selected != nil {
		// Decrease the selected proxy's current weight
		weight, _ := r.raftNode.Fsm.GetWeightedRRState(r.groupName, selected.GetBlockName())
		cmd := raft.Command{
			Type: raft.CommandUpdateWeightedRR,
			Payload: raft.WeightedRRPayload{
				GroupName: r.groupName,
				ProxyName: selected.GetBlockName(),
				Weight: raft.WeightedProxy{
					CurrentWeight:   weight.CurrentWeight - totalWeight,
					EffectiveWeight: weight.EffectiveWeight,
				},
			},
		}
		data, _ := json.Marshal(cmd)
		r.raftNode.Apply(data, raft.LeaderElectionTimeout)

		return selected, nil
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
