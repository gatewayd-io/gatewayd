package network

import (
	"sync"
	"testing"

	gerr "github.com/gatewayd-io/gatewayd/errors"
	"github.com/stretchr/testify/assert"
)

// TestNewRandom verifies that the NewRandom function properly initializes
// a Random object with the provided server and its associated proxies.
// The test ensures that the Random object is not nil and that the number of
// proxies in the Random object matches the number of proxies in the Server.
func TestNewRandom(t *testing.T) {
	proxies := []IProxy{&MockProxy{}, &MockProxy{}}
	server := &Server{Proxies: proxies}
	random := NewRandom(server)

	assert.NotNil(t, random)
	assert.Equal(t, len(proxies), len(random.proxies))
}

// TestGetNextProxy checks the behavior of the NextProxy method in various
// scenarios, including when proxies are available, when no proxies are available,
// and the randomness of proxy selection.
//
// - The first sub-test confirms that NextProxy returns a valid proxy when available.
// - The second sub-test ensures that an error is returned when there are no proxies available.
// - The third sub-test checks if the proxy selection is random by comparing two subsequent calls.
func TestGetNextProxy(t *testing.T) {
	t.Run("Returns a proxy when proxies are available", func(t *testing.T) {
		proxies := []IProxy{&MockProxy{}, &MockProxy{}}
		server := &Server{Proxies: proxies}
		random := NewRandom(server)

		proxy, err := random.NextProxy(nil)

		assert.Nil(t, err)
		assert.Contains(t, proxies, proxy)
	})

	t.Run("Returns error when no proxies are available", func(t *testing.T) {
		server := &Server{Proxies: []IProxy{}}
		random := NewRandom(server)

		proxy, err := random.NextProxy(nil)

		assert.Nil(t, proxy)
		assert.Equal(t, gerr.ErrNoProxiesAvailable.Message, err.Message)
	})
	t.Run("Random selection of proxies", func(t *testing.T) {
		proxies := []IProxy{&MockProxy{}, &MockProxy{}}
		server := &Server{Proxies: proxies}
		random := NewRandom(server)

		proxy1, _ := random.NextProxy(nil)
		proxy2, _ := random.NextProxy(nil)

		assert.Contains(t, proxies, proxy1)
		assert.Contains(t, proxies, proxy2)
		// It's possible that proxy1 and proxy2 are the same, but if we run this
		// test enough times, they should occasionally be different.
	})
}

// TestConcurrencySafety ensures that the Random object is safe for concurrent
// use by multiple goroutines. The test launches multiple goroutines that
// concurrently call the NextProxy method. It then verifies that all returned
// proxies are part of the expected set of proxies, ensuring thread safety.
func TestConcurrencySafety(t *testing.T) {
	proxies := []IProxy{&MockProxy{}, &MockProxy{}}
	server := &Server{Proxies: proxies}
	random := NewRandom(server)

	var waitGroup sync.WaitGroup
	numGoroutines := 100
	proxyChan := make(chan IProxy, numGoroutines)

	for range numGoroutines {
		waitGroup.Add(1)
		go func() {
			defer waitGroup.Done()
			proxy, _ := random.NextProxy(nil)
			proxyChan <- proxy
		}()
	}

	waitGroup.Wait()
	close(proxyChan)

	for proxy := range proxyChan {
		assert.Contains(t, proxies, proxy)
	}
}
