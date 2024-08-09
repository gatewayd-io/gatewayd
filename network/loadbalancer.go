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
	default:
		return nil, gerr.ErrLoadBalancerStrategyNotFound
	}
}
