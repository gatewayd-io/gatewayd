package network

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetRlimit(t *testing.T) {
	rlimit := GetRLimit()
	assert.Greater(t, rlimit.Cur, uint64(1))
	assert.Greater(t, rlimit.Max, uint64(1))
}

func TestGetID(t *testing.T) {
	id := GetID("tcp", "localhost:5432", 1)
	assert.Equal(t, "0cf47ee4e436ecb40dbd1d2d9a47179d1f6d98e2ea18d6fbd1cdfa85d3cec94f", id)
}

func TestResolve(t *testing.T) {
	address, err := Resolve("udp", "localhost:53")
	assert.NoError(t, err)
	assert.Equal(t, "127.0.0.1:53", address)
}
