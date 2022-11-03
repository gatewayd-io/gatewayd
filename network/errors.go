package network

import "errors"

var (
	ErrClientNotFound      = errors.New("client not found")
	ErrNetworkNotSupported = errors.New("network is not supported")
	ErrClientNotConnected  = errors.New("client is not connected")
	ErrPoolExhausted       = errors.New("pool is exhausted")
)
