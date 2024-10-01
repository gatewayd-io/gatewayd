package network

import (
	"math"
	"sync"
	"testing"
)

// TestNewRoundRobin tests the NewRoundRobin function to ensure that it correctly initializes
// the round-robin load balancer with the expected number of proxies.
func TestNewRoundRobin(t *testing.T) {
	proxies := []IProxy{
		MockProxy{name: "proxy1"},
		MockProxy{name: "proxy2"},
		MockProxy{name: "proxy3"},
	}
	server := &Server{Proxies: proxies}
	rr := NewRoundRobin(server)

	if len(rr.proxies) != len(proxies) {
		t.Errorf("expected %d proxies, got %d", len(proxies), len(rr.proxies))
	}
}

// TestRoundRobin_NextProxy tests the NextProxy method of the round-robin load balancer to ensure
// that it returns proxies in the expected order.
func TestRoundRobin_NextProxy(t *testing.T) {
	proxies := []IProxy{
		MockProxy{name: "proxy1"},
		MockProxy{name: "proxy2"},
		MockProxy{name: "proxy3"},
	}
	server := &Server{Proxies: proxies}
	roundRobin := NewRoundRobin(server)

	expectedOrder := []string{"proxy2", "proxy3", "proxy1", "proxy2", "proxy3"}

	for testIndex, expected := range expectedOrder {
		proxy, err := roundRobin.NextProxy(nil)
		if err != nil {
			t.Fatalf("test %d: unexpected error from NextProxy: %v", testIndex, err)
		}
		mockProxy, ok := proxy.(MockProxy)
		if !ok {
			t.Fatalf("test %d: expected proxy of type MockProxy, got %T", testIndex, proxy)
		}
		if mockProxy.GetBlockName() != expected {
			t.Errorf("test %d: expected proxy name %s, got %s", testIndex, expected, mockProxy.GetBlockName())
		}
	}
}

// TestRoundRobin_ConcurrentAccess tests the thread safety of the NextProxy method in the round-robin load balancer
// by invoking it concurrently from multiple goroutines and ensuring that the internal state is updated correctly.
func TestRoundRobin_ConcurrentAccess(t *testing.T) {
	proxies := []IProxy{
		MockProxy{name: "proxy1"},
		MockProxy{name: "proxy2"},
		MockProxy{name: "proxy3"},
	}
	server := &Server{Proxies: proxies}
	roundRobin := NewRoundRobin(server)

	var waitGroup sync.WaitGroup
	numGoroutines := 100
	waitGroup.Add(numGoroutines)

	for range numGoroutines {
		go func() {
			defer waitGroup.Done()
			_, _ = roundRobin.NextProxy(nil)
		}()
	}

	waitGroup.Wait()
	nextIndex := roundRobin.next.Load()
	if nextIndex != uint32(numGoroutines) { //nolint:gosec
		t.Errorf("expected next index to be %d, got %d", numGoroutines, nextIndex)
	}
}

// TestNextProxyOverflow verifies that the round-robin proxy selection correctly handles
// the overflow of the internal counter. It sets the counter to a value close to the maximum
// uint32 value and ensures that the proxy selection wraps around as expected when the
// counter overflows.
func TestNextProxyOverflow(t *testing.T) {
	// Create a server with a few mock proxies
	server := &Server{
		Proxies: []IProxy{
			&MockProxy{},
			&MockProxy{},
			&MockProxy{},
		},
	}
	roundRobin := NewRoundRobin(server)

	// Set the next value to near the max uint32 value to force an overflow
	roundRobin.next.Store(math.MaxUint32 - 1)

	// Call NextProxy multiple times to trigger the overflow
	for range 4 {
		proxy, err := roundRobin.NextProxy(nil)
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}
		if proxy == nil {
			t.Fatal("Expected a proxy, got nil")
		}
	}

	// After overflow, next value should wrap around
	expectedNextValue := uint32(2) // (MaxUint32 - 1 + 4) % ProxiesLen = 2
	actualNextValue := roundRobin.next.Load()
	if actualNextValue != expectedNextValue {
		t.Fatalf("Expected next value to be %v, got %v", expectedNextValue, actualNextValue)
	}
}
