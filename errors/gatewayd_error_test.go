package errors

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestNewGatewayDError tests the creation of a new GatewayDError.
func TestNewGatewayDError(t *testing.T) {
	err := NewGatewayDError(ErrCodeUnknown, "test", nil)
	assert.NotNil(t, err)
	assert.Equal(t, err.Code, ErrCodeUnknown)
	assert.Equal(t, err.Error(), "test")
	assert.Equal(t, err.Message, "test")
	assert.Nil(t, err.OriginalError)

	origErr := errors.New("original error") //nolint:goerr113
	assert.NotNil(t, err.Wrap(origErr))
	assert.Equal(t, err.OriginalError, origErr)
	assert.Equal(t, origErr, err.Unwrap())
	assert.Equal(t, err.Error(), "test, OriginalError: original error")
}
