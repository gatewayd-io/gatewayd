package network

import (
	"context"
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

func NewDistributionStrategy(ctx context.Context, ds string, server *Server) (IDistributionStrategy, *gerr.GatewayDError) {
	switch ds {
	case "ab-testing":
		return &ABTestingStrategy{DistributionStrategy{server}}, nil
	case "canary":
		return &CanaryStrategy{DistributionStrategy{server}}, nil
	case "write-read-split":
		return &WriteReadSplitStrategy{DistributionStrategy{server}}, nil
	case "round-robin":
		return NewRoundRobin(ctx, server), nil
	case "murmur-hash":
		return &MurMurHashStrategy{DistributionStrategy{server}}, nil
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

type MurMurHashStrategy struct {
	DistributionStrategy
}
