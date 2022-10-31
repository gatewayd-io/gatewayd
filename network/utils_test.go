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
	assert.Equal(t, "7b0a81105ebe3e390c617d9757feac3d5ea1a3d5", id)
}

func TestResolve(t *testing.T) {
	address, err := Resolve("udp", "localhost:53")
	assert.NoError(t, err)
	assert.Equal(t, "127.0.0.1:53", address)
}
