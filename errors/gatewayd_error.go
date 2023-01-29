package errors

import "fmt"

type ErrCode uint32

type GatewayDError struct {
	Code          ErrCode
	Message       string
	OriginalError error
}

// NewGatewayDError creates a new GatewayDError.
func NewGatewayDError(code ErrCode, message string, err error) *GatewayDError {
	return &GatewayDError{
		Code:          code,
		Message:       message,
		OriginalError: err,
	}
}

// Error returns the error message of the GatewayDError.
func (e *GatewayDError) Error() string {
	if e.OriginalError == nil {
		return e.Message
	}
	return fmt.Sprintf("%s, OriginalError: %s", e.Message, e.OriginalError)
}

// Wrap wraps the original error.
func (e *GatewayDError) Wrap(err error) *GatewayDError {
	e.OriginalError = err
	return e
}

// Unwrap returns the original error.
func (e *GatewayDError) Unwrap() error {
	return e.OriginalError
}
