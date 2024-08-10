package network

import (
	"testing"

	"github.com/gatewayd-io/gatewayd/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewWeightedRoundRobin(t *testing.T) {
	proxies := []IProxy{
		MockProxy{name: "proxy1"},
		MockProxy{name: "proxy2"},
		MockProxy{name: "proxy3"},
	}

	t.Run("loadBalancingRule with all proxies", func(t *testing.T) {
		loadBalancingRule := config.LoadBalancingRule{
			Condition: config.DefaultLoadBalancerCondition,
			Distribution: []config.Distribution{
				{
					ProxyName: "proxy1",
					Weight:    40,
				},
				{
					ProxyName: "proxy2",
					Weight:    60,
				},
				{
					ProxyName: "proxy3",
					Weight:    30,
				},
			},
		}
		server := &Server{Proxies: proxies}
		weightedRR := NewWeightedRoundRobin(server, loadBalancingRule)

		assert.NotNil(t, weightedRR, "weightedRR should not be nil")
		assert.Equal(t, len(loadBalancingRule.Distribution), len(weightedRR.proxies), "proxies count mismatch")
	})

	t.Run("loadBalancingRule with a subset of proxies", func(t *testing.T) {
		loadBalancingRule := config.LoadBalancingRule{
			Condition: config.DefaultLoadBalancerCondition,
			Distribution: []config.Distribution{
				{
					ProxyName: "proxy1",
					Weight:    40,
				},
				{
					ProxyName: "proxy2",
					Weight:    60,
				},
			},
		}
		server := &Server{Proxies: proxies}
		weightedRR := NewWeightedRoundRobin(server, loadBalancingRule)

		assert.NotNil(t, weightedRR, "weightedRR should not be nil")
		assert.Equal(t, len(loadBalancingRule.Distribution), len(weightedRR.proxies), "proxies count mismatch")
	})

	t.Run("loadBalancingRule with missing proxy", func(t *testing.T) {
		loadBalancingRule := config.LoadBalancingRule{
			Condition: config.DefaultLoadBalancerCondition,
			Distribution: []config.Distribution{
				{
					ProxyName: "proxy1",
					Weight:    50,
				},
				{
					ProxyName: "missing_proxy",
					Weight:    50,
				},
			},
		}
		server := &Server{Proxies: proxies}
		weightedRR := NewWeightedRoundRobin(server, loadBalancingRule)

		assert.NotNil(t, weightedRR, "weightedRR should not be nil")
		assert.Equal(t, 1, len(weightedRR.proxies), "should ignore missing proxies and only include available ones")
	})

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

func TestWeightedRoundRobinNextProxy(t *testing.T) {
	proxies := []IProxy{
		MockProxy{name: "proxy1"},
		MockProxy{name: "proxy2"},
		MockProxy{name: "proxy3"},
	}
	loadBalancingRule := config.LoadBalancingRule{
		Condition: config.DefaultLoadBalancerCondition,
		Distribution: []config.Distribution{
			{
				ProxyName: "proxy1",
				Weight:    30,
			},
			{
				ProxyName: "proxy2",
				Weight:    60,
			},
			{
				ProxyName: "proxy3",
				Weight:    10,
			},
		},
	}
	server := &Server{Proxies: proxies}
	weightedRR := NewWeightedRoundRobin(server, loadBalancingRule)

	// Define the expected distribution percentages.
	expectedWeights := map[string]int{
		"proxy1": 30,
		"proxy2": 60,
		"proxy3": 10,
	}

	// Total number of iterations to simulate
	totalRequests := 1000
	counts := map[string]int{
		"proxy1": 0,
		"proxy2": 0,
		"proxy3": 0,
	}

	for i := 0; i < totalRequests; i++ {
		proxy, err := weightedRR.NextProxy()
		require.Nil(t, err)

		mockProxy, ok := proxy.(MockProxy)
		require.True(t, ok, "expected proxy of type MockProxy, got %T", proxy)

		counts[mockProxy.GetName()]++
	}

	// Check that the distribution is within an acceptable range
	for proxyName, expectedWeight := range expectedWeights {
		expectedCount := totalRequests * expectedWeight / 100
		actualCount := counts[proxyName]

		// Allow a small margin of error
		assert.InDeltaf(t, expectedCount, actualCount, 5,
			"proxy %s: expected approximately %d, but got %d", proxyName, expectedCount, actualCount)
	}
}
