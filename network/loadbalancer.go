package network

import (
	"github.com/gatewayd-io/gatewayd/config"
	gerr "github.com/gatewayd-io/gatewayd/errors"
)

type LoadBalancerStrategy interface {
	NextProxy() (IProxy, *gerr.GatewayDError)
}

func NewLoadBalancerStrategy(server *Server) (LoadBalancerStrategy, *gerr.GatewayDError) {
	switch server.LoadbalancerStrategyName {
	case config.RoundRobinStrategy:
		return NewRoundRobin(server), nil
	case config.RANDOMStrategy:
		return NewRandom(server), nil
	case config.WeightedRoundRobinStrategy:
		if server.LoadbalancerRules == nil {
			return nil, gerr.ErrNoLoadBalancerRules
		}
		loadbalancerRule := selectLoadBalancerRule(server.LoadbalancerRules)
		return NewWeightedRoundRobin(server, loadbalancerRule), nil
	default:
		return nil, gerr.ErrLoadBalancerStrategyNotFound
	}
}

func selectLoadBalancerRule(loadbalancerRules []config.LoadBalancingRule) config.LoadBalancingRule {
	for _, loadbalancerRule := range loadbalancerRules {
		if loadbalancerRule.Condition == config.DefaultLoadBalancerCondition {
			return loadbalancerRule
		}
	}
	// Return the first rule as a fallback
	return loadbalancerRules[0]
}
