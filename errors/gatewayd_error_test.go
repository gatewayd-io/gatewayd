package errors

import (
	"io"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestNewGatewayDError tests the creation of a new GatewayDError.
func TestNewGatewayDError(t *testing.T) {
	err := NewGatewayDError(ErrCodeUnknown, "test", nil)
	assert.NotNil(t, err)
	assert.Equal(t, ErrCodeUnknown, err.Code)
	assert.Equal(t, "test", err.Error())
	assert.Equal(t, "test", err.Message)
	require.NoError(t, err.OriginalError)

	assert.NotNil(t, err.Wrap(io.EOF))
	assert.Equal(t, io.EOF, err.OriginalError)
	assert.Equal(t, io.EOF, err.Unwrap())
	assert.Equal(t, "test, OriginalError: EOF", err.Error())
}
