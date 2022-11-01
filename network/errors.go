package network

import "errors"

var (
	ClientNotFound      = errors.New("client not found")
	NetworkNotSupported = errors.New("network is not supported")
	ClientNotConnected  = errors.New("client is not connected")
	PoolExhausted       = errors.New("pool is exhausted")
)
