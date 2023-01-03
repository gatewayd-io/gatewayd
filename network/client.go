package network

import (
	"fmt"
	"net"
	"time"

	gerr "github.com/gatewayd-io/gatewayd/errors"
	"github.com/rs/zerolog"
)

const (
	DefaultSeed               = 1000
	DefaultChunkSize          = 4096
	DefaultReceiveDeadline    = 0 // 0 means no deadline (timeout)
	DefaultSendDeadline       = 0
	DefaultTCPKeepAlivePeriod = 30 * time.Second
)

type ClientInterface interface {
	Send(data []byte) (int, *gerr.GatewayDError)
	Receive() (int, []byte, *gerr.GatewayDError)
	Close()
	IsConnected() bool
}

type Client struct {
	net.Conn

	logger zerolog.Logger

	TCPKeepAlive       bool
	TCPKeepAlivePeriod time.Duration
	ReceiveBufferSize  int
	ReceiveChunkSize   int
	ReceiveDeadline    time.Duration
	SendDeadline       time.Duration
	ID                 string
	Network            string // tcp/udp/unix
	Address            string
}

var _ ClientInterface = &Client{}

// TODO: implement a better connection management algorithm

// NewClient creates a new client.
//
//nolint:funlen
func NewClient(
	network, address string,
	receiveBufferSize, receiveChunkSize int,
	receiveDeadline, sendDeadline time.Duration,
	tcpKeepAlive bool, tcpKeepAlivePeriod time.Duration,
	logger zerolog.Logger,
) *Client {
	var client Client

	client.logger = logger

	// Try to resolve the address and log an error if it can't be resolved.
	addr, err := Resolve(network, address, logger)
	if err != nil {
		logger.Error().Err(err).Msg("Failed to resolve address")
	}

	// Create a resolved client.
	client = Client{
		Network: network,
		Address: addr,
	}

	// Fall back to the original network and address if the address can't be resolved.
	if client.Address == "" || client.Network == "" {
		client = Client{
			Network: network,
			Address: address,
		}
	}

	// Create a new connection.
	conn, origErr := net.Dial(client.Network, client.Address)
	if origErr != nil {
		err := gerr.ErrClientConnectionFailed.Wrap(origErr)
		logger.Error().Err(err).Msg("Failed to create a new connection")
		return nil
	}

	client.Conn = conn

	// Set the TCP keep alive.
	client.TCPKeepAlive = tcpKeepAlive
	if tcpKeepAlivePeriod <= 0 {
		client.TCPKeepAlivePeriod = DefaultTCPKeepAlivePeriod
	} else {
		client.TCPKeepAlivePeriod = tcpKeepAlivePeriod
	}

	if c, ok := client.Conn.(*net.TCPConn); ok {
		if err := c.SetKeepAlive(client.TCPKeepAlive); err != nil {
			logger.Error().Err(err).Msg("Failed to set keep alive")
		} else {
			if err := c.SetKeepAlivePeriod(client.TCPKeepAlivePeriod); err != nil {
				logger.Error().Err(err).Msg("Failed to set keep alive period")
			}
		}
	}

	// Set the receive deadline (timeout).
	if receiveDeadline <= 0 {
		client.ReceiveDeadline = DefaultReceiveDeadline
	} else {
		client.ReceiveDeadline = receiveDeadline
		if err := client.Conn.SetReadDeadline(time.Now().Add(client.ReceiveDeadline)); err != nil {
			logger.Error().Err(err).Msg("Failed to set receive deadline")
		} else {
			logger.Debug().Str("duration", fmt.Sprint(client.ReceiveDeadline.Seconds())).Msg(
				"Set receive deadline")
		}
	}

	// Set the send deadline (timeout).
	if sendDeadline <= 0 {
		client.SendDeadline = DefaultSendDeadline
	} else {
		client.SendDeadline = sendDeadline
		if err := client.Conn.SetWriteDeadline(time.Now().Add(client.SendDeadline)); err != nil {
			logger.Error().Err(err).Msg("Failed to set send deadline")
		} else {
			logger.Debug().Str("duration", fmt.Sprint(client.SendDeadline)).Msg(
				"Set send deadline")
		}
	}

	// Set the receive buffer size. This is the maximum size of the buffer.
	if receiveBufferSize <= 0 {
		client.ReceiveBufferSize = DefaultBufferSize
	} else {
		client.ReceiveBufferSize = receiveBufferSize
	}

	// Set the receive chunk size. This is the size of the buffer that is read from the connection
	// in chunks.
	if receiveChunkSize <= 0 {
		client.ReceiveChunkSize = DefaultChunkSize
	} else {
		client.ReceiveChunkSize = receiveChunkSize
	}

	logger.Debug().Str("address", client.Address).Msg("New client created")
	client.ID = GetID(conn.LocalAddr().Network(), conn.LocalAddr().String(), DefaultSeed, logger)

	return &client
}

// Send sends data to the server.
func (c *Client) Send(data []byte) (int, *gerr.GatewayDError) {
	sent, err := c.Conn.Write(data)
	if err != nil {
		c.logger.Error().Err(err).Msg("Couldn't send data to the server")
		return 0, gerr.ErrClientSendFailed.Wrap(err)
	}
	c.logger.Debug().Fields(
		map[string]interface{}{
			"length":  sent,
			"address": c.Address,
		},
	).Msg("Sent data to server")
	return sent, nil
}

// Receive receives data from the server.
func (c *Client) Receive() (int, []byte, *gerr.GatewayDError) {
	var received int
	buffer := make([]byte, 0, c.ReceiveBufferSize)
	// Read the data in chunks.
	for {
		chunk := make([]byte, c.ReceiveChunkSize)
		read, err := c.Conn.Read(chunk)
		switch {
		case read > 0 && err != nil:
			received += read
			buffer = append(buffer, chunk[:read]...)
			c.logger.Error().Err(err).Msg("Couldn't receive data from the server")
			return received, buffer, gerr.ErrClientReceiveFailed.Wrap(err)
		case err != nil:
			c.logger.Error().Err(err).Msg("Couldn't receive data from the server")
			return received, buffer, gerr.ErrClientReceiveFailed.Wrap(err)
		default:
			received += read
			buffer = append(buffer, chunk[:read]...)
		}

		if read == 0 || read < c.ReceiveChunkSize {
			break
		}
	}
	return received, buffer, nil
}

// Close closes the connection to the server.
func (c *Client) Close() {
	c.logger.Debug().Str("address", c.Address).Msg("Closing connection to server")
	if c.Conn != nil {
		c.Conn.Close()
	}
	c.ID = ""
	c.Conn = nil
	c.Address = ""
	c.Network = ""
}

// IsConnected checks if the client is still connected to the server.
func (c *Client) IsConnected() bool {
	if c == nil {
		c.logger.Debug().Fields(
			map[string]interface{}{
				"address": c.Address,
				"reason":  "client is nil",
			}).Msg("Connection to server is closed")
		return false
	}

	if c != nil && c.Conn == nil || c.ID == "" {
		c.logger.Debug().Fields(
			map[string]interface{}{
				"address": c.Address,
				"reason":  "connection is nil or invalid",
			}).Msg("Connection to server is closed")
		return false
	}

	if n, err := c.Read([]byte{}); n == 0 && err != nil {
		c.logger.Debug().Fields(
			map[string]interface{}{
				"address": c.Address,
				"reason":  "read 0 bytes",
			}).Msg("Connection to server is closed")
		return false
	}

	return true
}
