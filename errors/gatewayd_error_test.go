package errors

import (
	"io"
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestNewGatewayDError tests the creation of a new GatewayDError.
func TestNewGatewayDError(t *testing.T) {
	err := NewGatewayDError(GatewayDError{
		Code:          ErrCodeUnknown,
		Message:       "test",
		OriginalError: nil,
	})
	assert.NotNil(t, err)
	assert.Equal(t, err.Code, ErrCodeUnknown)
	assert.Equal(t, err.Error(), "test")
	assert.Equal(t, err.Message, "test")
	assert.Nil(t, err.OriginalError)

	assert.NotNil(t, err.Wrap(io.EOF))
	assert.Equal(t, err.OriginalError, io.EOF)
	assert.Equal(t, io.EOF, err.Unwrap())
	assert.Equal(t, err.Error(), "test, OriginalError: EOF")
}
