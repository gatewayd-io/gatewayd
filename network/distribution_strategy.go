package network

import (
	gerr "github.com/gatewayd-io/gatewayd/errors"
)

type IDistributionStrategy interface {
	Distribute(*ConnWrapper) (IProxy, error)
}

type DistributionStrategy struct {
	server *Server
}

func (ds DistributionStrategy) Distribute(wrapper *ConnWrapper) (IProxy, error) {
	// the place to implement the default behaviour
	// Return the first proxy
	return ds.server.Proxies[0], nil
}

var _ IDistributionStrategy = DistributionStrategy{}

func NewDistributionStrategy(ds string, s *Server) (IDistributionStrategy, *gerr.GatewayDError) {
	switch ds {
	case "ab-testing":
		return &ABTestingStrategy{DistributionStrategy{s}}, nil
	case "canary":
		return &CanaryStrategy{DistributionStrategy{s}}, nil
	case "write-read-split":
		return &WriteReadSplitStrategy{DistributionStrategy{s}}, nil
	case "round-robin":
		return &RoundRobinStrategy{DistributionStrategy{s}}, nil
	case "murmur-hash":
		return &MurMurHashStrategy{DistributionStrategy{s}}, nil
	default:
		return nil, gerr.ErrDistributionStrategyNotFound
	}
}

type ABTestingStrategy struct {
	DistributionStrategy
}

type WriteReadSplitStrategy struct {
	DistributionStrategy
}

type CanaryStrategy struct {
	DistributionStrategy
}

type RoundRobinStrategy struct {
	DistributionStrategy
}

type MurMurHashStrategy struct {
	DistributionStrategy
}
