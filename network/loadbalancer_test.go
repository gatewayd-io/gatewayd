package network

import (
	"errors"
	"testing"

	"github.com/gatewayd-io/gatewayd/config"
	gerr "github.com/gatewayd-io/gatewayd/errors"
)

// TestNewLoadBalancerStrategy tests the NewLoadBalancerStrategy function to ensure it correctly
// initializes the load balancer strategy based on the strategy name provided in the server configuration.
// It covers both valid and invalid strategy names.
func TestNewLoadBalancerStrategy(t *testing.T) {
	serverValid := &Server{
		LoadbalancerStrategyName: config.RoundRobinStrategy,
		Proxies:                  []IProxy{MockProxy{}},
	}

	// Test case 1: Valid strategy name
	strategy, err := NewLoadBalancerStrategy(serverValid)
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	_, ok := strategy.(*RoundRobin)
	if !ok {
		t.Errorf("Expected strategy to be of type RoundRobin")
	}

	// Test case 2: InValid strategy name
	serverInvalid := &Server{
		LoadbalancerStrategyName: "InvalidStrategy",
		Proxies:                  []IProxy{MockProxy{}},
	}

	strategy, err = NewLoadBalancerStrategy(serverInvalid)
	if !errors.Is(err, gerr.ErrLoadBalancerStrategyNotFound) {
		t.Errorf("Expected ErrLoadBalancerStrategyNotFound, got %v", err)
	}
	if strategy != nil {
		t.Errorf("Expected strategy to be nil for invalid strategy name")
	}
}
