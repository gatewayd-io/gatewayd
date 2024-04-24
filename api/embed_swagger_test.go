package api

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_IsSwaggerEmbedded(t *testing.T) {
	assert.False(t, IsSwaggerEmbedded())
}
