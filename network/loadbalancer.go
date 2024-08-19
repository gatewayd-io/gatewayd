package network

import (
	"github.com/gatewayd-io/gatewayd/config"
	gerr "github.com/gatewayd-io/gatewayd/errors"
)

type LoadBalancerStrategy interface {
	NextProxy(conn IConnWrapper) (IProxy, *gerr.GatewayDError)
}

// NewLoadBalancerStrategy returns a LoadBalancerStrategy based on the server's load balancer strategy name.
// If the server's load balancer strategy is weighted round-robin,
// it selects a load balancer rule before returning the strategy.
// Returns an error if the strategy is not found or if there are no load balancer rules when required.
func NewLoadBalancerStrategy(server *Server) (LoadBalancerStrategy, *gerr.GatewayDError) {
	var strategy LoadBalancerStrategy
	switch server.LoadbalancerStrategyName {
	case config.RoundRobinStrategy:
		strategy = NewRoundRobin(server)
	case config.RANDOMStrategy:
		strategy = NewRandom(server)
	case config.WeightedRoundRobinStrategy:
		if server.LoadbalancerRules == nil {
			return nil, gerr.ErrNoLoadBalancerRules
		}
		loadbalancerRule := selectLoadBalancerRule(server.LoadbalancerRules)
		strategy = NewWeightedRoundRobin(server, loadbalancerRule)
	default:
		return nil, gerr.ErrLoadBalancerStrategyNotFound
	}

	// If consistent hashing is enabled, wrap the strategy
	if server.LoadbalancerConsistentHash != nil {
		strategy = NewConsistentHash(server, strategy)
	}

	return strategy, nil
}

// selectLoadBalancerRule selects and returns the first load balancer rule that matches the default condition.
// If no rule matches, it returns the first rule in the list as a fallback.
func selectLoadBalancerRule(rules []config.LoadBalancingRule) config.LoadBalancingRule {
	for _, rule := range rules {
		if rule.Condition == config.DefaultLoadBalancerCondition {
			return rule
		}
	}
	// Return the first rule as a fallback
	return rules[0]
}
