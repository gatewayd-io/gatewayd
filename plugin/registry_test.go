package plugin

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestPluginRegistry tests the PluginRegistry.
func TestPluginRegistry(t *testing.T) {
	hooksConfig := NewHookConfig()
	assert.NotNil(t, hooksConfig)
	reg := NewRegistry(hooksConfig)
	assert.NotNil(t, reg)
	assert.NotNil(t, reg.plugins)
	assert.NotNil(t, reg.hooksConfig)
	assert.Equal(t, 0, len(reg.List()))

	ident := Identifier{
		Name:      "test",
		Version:   "1.0.0",
		RemoteURL: "github.com/remote/test",
	}
	impl := &Impl{
		ID: ident,
	}
	reg.Add(impl)
	assert.Equal(t, 1, len(reg.List()))

	instance := reg.Get(ident)
	assert.Equal(t, instance, impl)

	reg.Remove(ident)
	assert.Equal(t, 0, len(reg.List()))

	reg.Shutdown()
}