package network

import (
	"sync"
	"testing"

	"github.com/gatewayd-io/gatewayd/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestNewWeightedRoundRobin tests the creation and initialization of the
// NewWeightedRoundRobin function with various load balancing rules.
func TestNewWeightedRoundRobin(t *testing.T) {
	proxies := []IProxy{
		MockProxy{name: "proxy1"},
		MockProxy{name: "proxy2"},
		MockProxy{name: "proxy3"},
	}

	// Test with a load balancing rule that includes all proxies.
	t.Run("loadBalancingRule with all proxies", func(t *testing.T) {
		loadBalancingRule := config.LoadBalancingRule{
			Condition: config.DefaultLoadBalancerCondition,
			Distribution: []config.Distribution{
				{ProxyName: "proxy1", Weight: 40},
				{ProxyName: "proxy2", Weight: 60},
				{ProxyName: "proxy3", Weight: 30},
			},
		}
		server := &Server{Proxies: proxies}
		weightedRR := NewWeightedRoundRobin(server, loadBalancingRule)

		assert.NotNil(t, weightedRR, "weightedRR should not be nil")
		assert.Equal(t, len(loadBalancingRule.Distribution), len(weightedRR.proxies), "proxies count mismatch")
	})

	// Test with a load balancing rule that includes a subset of proxies.
	t.Run("loadBalancingRule with a subset of proxies", func(t *testing.T) {
		loadBalancingRule := config.LoadBalancingRule{
			Condition: config.DefaultLoadBalancerCondition,
			Distribution: []config.Distribution{
				{ProxyName: "proxy1", Weight: 40},
				{ProxyName: "proxy2", Weight: 60},
			},
		}
		server := &Server{Proxies: proxies}
		weightedRR := NewWeightedRoundRobin(server, loadBalancingRule)

		assert.NotNil(t, weightedRR, "weightedRR should not be nil")
		assert.Equal(t, len(loadBalancingRule.Distribution), len(weightedRR.proxies), "proxies count mismatch")
	})

	// Test with a load balancing rule that references a missing proxy.
	t.Run("loadBalancingRule with missing proxy", func(t *testing.T) {
		loadBalancingRule := config.LoadBalancingRule{
			Condition: config.DefaultLoadBalancerCondition,
			Distribution: []config.Distribution{
				{ProxyName: "proxy1", Weight: 50},
				{ProxyName: "missing_proxy", Weight: 50},
			},
		}
		server := &Server{Proxies: proxies}
		weightedRR := NewWeightedRoundRobin(server, loadBalancingRule)

		assert.NotNil(t, weightedRR, "weightedRR should not be nil")
		assert.Equal(t, 1, len(weightedRR.proxies), "should ignore missing proxies and only include available ones")
	})

	// Test with an empty distribution list in the load balancing rule.
	t.Run("loadBalancingRule with empty distribution", func(t *testing.T) {
		loadBalancingRule := config.LoadBalancingRule{
			Condition:    config.DefaultLoadBalancerCondition,
			Distribution: []config.Distribution{}, // empty distribution
		}
		server := &Server{Proxies: proxies}
		weightedRR := NewWeightedRoundRobin(server, loadBalancingRule)

		assert.NotNil(t, weightedRR, "weightedRR should not be nil")
		assert.Equal(t, 0, len(weightedRR.proxies), "no proxies should be included")
	})

	// Test with a nil distribution list in the load balancing rule.
	t.Run("loadBalancingRule with nil distribution", func(t *testing.T) {
		loadBalancingRule := config.LoadBalancingRule{
			Condition:    config.DefaultLoadBalancerCondition,
			Distribution: nil, // nil distribution
		}
		server := &Server{Proxies: proxies}
		weightedRR := NewWeightedRoundRobin(server, loadBalancingRule)

		assert.NotNil(t, weightedRR, "weightedRR should not be nil")
		assert.Equal(t, 0, len(weightedRR.proxies), "no proxies should be included")
	})
}

// TestWeightedRoundRobinNextProxy verifies that the WeightedRoundRobin algorithm
// correctly distributes requests among proxies according to their weights.
func TestWeightedRoundRobinNextProxy(t *testing.T) {
	proxies := []IProxy{
		MockProxy{name: "proxy1"},
		MockProxy{name: "proxy2"},
		MockProxy{name: "proxy3"},
	}

	loadBalancingRule := config.LoadBalancingRule{
		Condition: config.DefaultLoadBalancerCondition,
		Distribution: []config.Distribution{
			{ProxyName: "proxy1", Weight: 30},
			{ProxyName: "proxy2", Weight: 60},
			{ProxyName: "proxy3", Weight: 10},
		},
	}

	server := &Server{Proxies: proxies}
	weightedRR := NewWeightedRoundRobin(server, loadBalancingRule)

	// Expected distribution of requests among proxies based on the weights.
	expectedWeights := map[string]int{
		"proxy1": 30,
		"proxy2": 60,
		"proxy3": 10,
	}

	// Total number of simulated requests to distribute among proxies.
	totalRequests := 1000
	counts := map[string]int{
		"proxy1": 0,
		"proxy2": 0,
		"proxy3": 0,
	}

	// Simulate the distribution of requests using the WeightedRoundRobin algorithm.
	for range totalRequests {
		proxy, err := weightedRR.NextProxy(nil)
		require.Nil(t, err)

		mockProxy, ok := proxy.(MockProxy)
		require.True(t, ok, "expected proxy of type MockProxy, got %T", proxy)

		counts[mockProxy.GetBlockName()]++
	}

	// Validate that the actual distribution of requests closely matches the expected distribution.
	for proxyName, expectedWeight := range expectedWeights {
		expectedCount := totalRequests * expectedWeight / 100
		actualCount := counts[proxyName]

		// Allow a small margin of error (delta of 5) in the actual count.
		assert.InDeltaf(t, expectedCount, actualCount, 5,
			"proxy %s: expected approximately %d, but got %d", proxyName, expectedCount, actualCount)
	}
}

// TestWeightedRoundRobinConcurrentAccess tests the thread-safety of the
// WeightedRoundRobin algorithm by simulating concurrent access through
// multiple goroutines and ensuring the expected proxy distribution.
func TestWeightedRoundRobinConcurrentAccess(t *testing.T) {
	proxies := []IProxy{
		MockProxy{name: "proxy1"},
		MockProxy{name: "proxy2"},
		MockProxy{name: "proxy3"},
	}

	loadBalancingRule := config.LoadBalancingRule{
		Condition: config.DefaultLoadBalancerCondition,
		Distribution: []config.Distribution{
			{ProxyName: "proxy1", Weight: 3},
			{ProxyName: "proxy2", Weight: 2},
			{ProxyName: "proxy3", Weight: 1},
		},
	}

	server := &Server{Proxies: proxies}
	weightedRR := NewWeightedRoundRobin(server, loadBalancingRule)

	// WaitGroup to synchronize the completion of all goroutines.
	var waitGroup sync.WaitGroup
	numGoroutines := 100
	proxySelection := make(map[string]int)
	var mux sync.Mutex

	// Run multiple goroutines to simulate concurrent access to the NextProxy method.
	for range numGoroutines {
		waitGroup.Add(1)
		go func() {
			defer waitGroup.Done()

			// Retrieve the next proxy using the WeightedRoundRobin algorithm.
			proxy, err := weightedRR.NextProxy(nil)
			if assert.Nil(t, err, "No error expected when getting a proxy") {
				// Safely update the proxy selection count using a mutex.
				mux.Lock()
				proxySelection[proxy.GetBlockName()]++
				mux.Unlock()
			}
		}()
	}

	// Wait for all goroutines to finish.
	waitGroup.Wait()

	// Expected number of selections for each proxy based on their weights.
	expectedSelections := map[string]int{
		"proxy1": 50, // proxy1 should be selected 50 times (3/6 of 100)
		"proxy2": 33, // proxy2 should be selected 33 times (2/6 of 100)
		"proxy3": 17, // proxy3 should be selected 17 times (1/6 of 100)
	}

	// Validate that the actual selection counts are close to the expected counts.
	for name, expectedCount := range expectedSelections {
		actualCount, exists := proxySelection[name]
		assert.True(t, exists, "Expected proxy %s to be selected", name)
		assert.InDelta(t, expectedCount, actualCount, 5, "Proxy %s selection count should be within expected range", name)
	}
}
