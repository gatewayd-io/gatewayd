package errors

import "fmt"

type ErrCode uint32

type GatewayDError struct {
	Code          ErrCode
	Message       string
	OriginalError error
}

func NewGatewayDError(code ErrCode, message string, err error) *GatewayDError {
	return &GatewayDError{
		Code:          code,
		Message:       message,
		OriginalError: err,
	}
}

func (e *GatewayDError) Error() string {
	if e.OriginalError == nil {
		return e.Message
	}
	return fmt.Sprintf("%s, OriginalError: %s", e.Message, e.OriginalError)
}

func (e *GatewayDError) Wrap(err error) *GatewayDError {
	e.OriginalError = err
	return e
}

func (e *GatewayDError) Unwrap() error {
	return e.OriginalError
}
