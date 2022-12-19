package errors

import "errors"

var (
	ErrClientNotFound      = errors.New("client not found")
	ErrNetworkNotSupported = errors.New("network is not supported")
	ErrClientNotConnected  = errors.New("client is not connected")
	ErrPoolExhausted       = errors.New("pool is exhausted")

	ErrPluginNotFound = errors.New("plugin not found")
	ErrPluginNotReady = errors.New("plugin is not ready")

	ErrClientReceiveFailed = errors.New("couldn't receive data from the server")
	ErrClientSendFailed    = errors.New("couldn't send data to the server")

	ErrPutFailed = errors.New("failed to put in pool")

	ErrCastFailed = errors.New("failed to cast")
)

const (
	FailedToLoadPluginConfig = 1
	FailedToLoadGlobalConfig = 2
)
