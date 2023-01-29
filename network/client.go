package network

import (
	"net"

	gerr "github.com/gatewayd-io/gatewayd/errors"
	"github.com/rs/zerolog"
)

const (
	DefaultSeed      = 1000
	DefaultChunkSize = 4096
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

	ID                string
	ReceiveBufferSize int
	Network           string // tcp/udp/unix
	Address           string
	// TODO: add read/write deadline and deal with timeouts
}

var _ ClientInterface = &Client{}

// TODO: implement a better connection management algorithm

func NewClient(network, address string, receiveBufferSize int, logger zerolog.Logger) *Client {
	var client Client

	client.logger = logger

	// Try to resolve the address and log an error if it can't be resolved
	addr, err := Resolve(network, address, logger)
	if err != nil {
		logger.Error().Err(err).Msg("Failed to resolve address")
	}

	// Create a resolved client
	client = Client{
		Network: network,
		Address: addr,
	}

	// Fall back to the original network and address if the address can't be resolved
	if client.Address == "" || client.Network == "" {
		client = Client{
			Network: network,
			Address: address,
		}
	}

	// Create a new connection
	conn, origErr := net.Dial(client.Network, client.Address)
	if origErr != nil {
		err := gerr.ErrClientConnectionFailed.Wrap(origErr)
		logger.Error().Err(err).Msg("Failed to create a new connection")
		return nil
	}

	client.Conn = conn
	if receiveBufferSize <= 0 {
		client.ReceiveBufferSize = DefaultBufferSize
	} else {
		client.ReceiveBufferSize = receiveBufferSize
	}

	logger.Debug().Msgf("New client created: %s", client.Address)
	client.ID = GetID(conn.LocalAddr().Network(), conn.LocalAddr().String(), DefaultSeed, logger)

	return &client
}

func (c *Client) Send(data []byte) (int, *gerr.GatewayDError) {
	sent, err := c.Conn.Write(data)
	if err != nil {
		c.logger.Error().Err(err).Msgf("Couldn't send data to the server: %s", err)
		return 0, gerr.ErrClientSendFailed.Wrap(err)
	}
	c.logger.Debug().Msgf("Sent %d bytes to %s", len(data), c.Address)
	return sent, nil
}

func (c *Client) Receive() (int, []byte, *gerr.GatewayDError) {
	var received int
	buffer := make([]byte, 0, c.ReceiveBufferSize)
	for {
		smallBuf := make([]byte, DefaultChunkSize)
		read, err := c.Conn.Read(smallBuf)
		switch {
		case read > 0 && err != nil:
			received += read
			buffer = append(buffer, smallBuf[:read]...)
			c.logger.Error().Err(err).Msg("Couldn't receive data from the server")
			return received, buffer, gerr.ErrClientReceiveFailed.Wrap(err)
		case err != nil:
			c.logger.Error().Err(err).Msg("Couldn't receive data from the server")
			return received, buffer, gerr.ErrClientReceiveFailed.Wrap(err)
		default:
			received += read
			buffer = append(buffer, smallBuf[:read]...)
		}

		if read == 0 || read < DefaultChunkSize {
			break
		}
	}
	return received, buffer, nil
}

func (c *Client) Close() {
	c.logger.Debug().Msgf("Closing connection to %s", c.Address)
	if c.Conn != nil {
		c.Conn.Close()
	}
	c.ID = ""
	c.Conn = nil
	c.Address = ""
	c.Network = ""
}

func (c *Client) IsConnected() bool {
	if c == nil {
		c.logger.Debug().Str(
			"reason", "client is nil").Msgf("Connection to %s is closed", c.Address)
		return false
	}

	if c != nil && c.Conn == nil || c.ID == "" {
		c.logger.Debug().Str(
			"reason", "connection is nil or invalid",
		).Msgf("Connection to %s is closed", c.Address)
		return false
	}

	if n, err := c.Read([]byte{}); n == 0 && err != nil {
		c.logger.Debug().Str(
			"reason", "read 0 bytes").Msgf("Connection to %s is closed, ", c.Address)
		return false
	}

	return true
}
