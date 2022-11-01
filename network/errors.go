package network

import "errors"

var (
	ClientNotFound      = errors.New("Client not found")
	NetworkNotSupported = errors.New("Network is not supported")
	ClientNotConnected  = errors.New("Client is not connected")
	PoolExhausted       = errors.New("Pool is exhausted")
)
