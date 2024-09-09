package network

import (
	"crypto/rand"
	"errors"
	"fmt"
	"math/big"
	"sync"

	gerr "github.com/gatewayd-io/gatewayd/errors"
)

// Random is a struct that holds a list of proxies and a mutex for thread safety.
type Random struct {
	mu      sync.Mutex
	proxies []IProxy
}

// NewRandom creates a new Random instance with the given server's proxies.
func NewRandom(server *Server) *Random {
	return &Random{
		proxies: server.Proxies,
	}
}

// NextProxy returns a random proxy from the list.
func (r *Random) NextProxy(_ IConnWrapper) (IProxy, *gerr.GatewayDError) {
	r.mu.Lock()
	defer r.mu.Unlock()

	proxiesLen := len(r.proxies)
	if proxiesLen == 0 {
		return nil, gerr.ErrNoProxiesAvailable.Wrap(errors.New("proxy list is empty"))
	}

	randomIndex, err := randInt(proxiesLen)
	if err != nil {
		return nil, gerr.ErrNoProxiesAvailable.Wrap(err)
	}

	return r.proxies[randomIndex], nil
}

// randInt generates a random integer between 0 and max-1 using crypto/rand.
func randInt(maxValue int) (int, error) {
	// Generate a secure random number
	n, err := rand.Int(rand.Reader, big.NewInt(int64(maxValue)))
	if err != nil {
		return 0, fmt.Errorf("failed to generate random integer: %w", err)
	}
	return int(n.Int64()), nil
}
