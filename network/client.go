package network

import (
	"bytes"
	"context"
	"fmt"
	"net"
	"time"

	"github.com/gatewayd-io/gatewayd/config"
	gerr "github.com/gatewayd-io/gatewayd/errors"
	"github.com/gatewayd-io/gatewayd/metrics"
	"github.com/rs/zerolog"
	"go.opentelemetry.io/otel"
)

type IClient interface {
	Send(data []byte) (int, *gerr.GatewayDError)
	Receive() (int, []byte, *gerr.GatewayDError)
	Close()
	IsConnected() bool
	RemoteAddr() string
	LocalAddr() string
}

type Client struct {
	net.Conn

	logger zerolog.Logger
	ctx    context.Context //nolint:containedctx

	TCPKeepAlive       bool
	TCPKeepAlivePeriod time.Duration
	ReceiveChunkSize   int
	ReceiveDeadline    time.Duration
	SendDeadline       time.Duration
	ReceiveTimeout     time.Duration
	ID                 string
	Network            string // tcp/udp/unix
	Address            string
}

var _ IClient = &Client{}

// NewClient creates a new client.
func NewClient(ctx context.Context, clientConfig *config.Client, logger zerolog.Logger) *Client {
	clientCtx, span := otel.Tracer(config.TracerName).Start(ctx, "NewClient")
	defer span.End()

	var client Client

	if clientConfig == nil || clientConfig == (&config.Client{}) {
		return nil
	}

	client.logger = logger

	// Try to resolve the address and log an error if it can't be resolved.
	addr, err := Resolve(clientConfig.Network, clientConfig.Address, logger)
	if err != nil {
		logger.Error().Err(err).Msg("Failed to resolve address")
		span.RecordError(err)
	}

	// Create a resolved client.
	client = Client{
		ctx:     clientCtx,
		Network: clientConfig.Network,
		Address: addr,
	}

	// Fall back to the original network and address if the address can't be resolved.
	if client.Address == "" || client.Network == "" {
		client = Client{
			Network: clientConfig.Network,
			Address: clientConfig.Address,
		}
	}

	// Create a new connection.
	conn, origErr := net.Dial(client.Network, client.Address)
	if origErr != nil {
		err := gerr.ErrClientConnectionFailed.Wrap(origErr)
		logger.Error().Err(err).Msg("Failed to create a new connection")
		span.RecordError(err)
		return nil
	}

	client.Conn = conn

	// Set the TCP keep alive.
	client.TCPKeepAlive = clientConfig.TCPKeepAlive
	client.TCPKeepAlivePeriod = clientConfig.TCPKeepAlivePeriod

	if c, ok := client.Conn.(*net.TCPConn); ok {
		if err := c.SetKeepAlive(client.TCPKeepAlive); err != nil {
			logger.Error().Err(err).Msg("Failed to set keep alive")
			span.RecordError(err)
		} else {
			if err := c.SetKeepAlivePeriod(client.TCPKeepAlivePeriod); err != nil {
				logger.Error().Err(err).Msg("Failed to set keep alive period")
				span.RecordError(err)
			}
		}
	}

	// Set the receive deadline (timeout).
	client.ReceiveDeadline = clientConfig.ReceiveDeadline
	if client.ReceiveDeadline > 0 {
		if err := client.Conn.SetReadDeadline(time.Now().Add(client.ReceiveDeadline)); err != nil {
			logger.Error().Err(err).Msg("Failed to set receive deadline")
			span.RecordError(err)
		} else {
			logger.Debug().Str("duration", fmt.Sprint(client.ReceiveDeadline.String())).Msg(
				"Set receive deadline")
		}
	}

	// Set the send deadline (timeout).
	client.SendDeadline = clientConfig.SendDeadline
	if client.SendDeadline > 0 {
		if err := client.Conn.SetWriteDeadline(time.Now().Add(client.SendDeadline)); err != nil {
			logger.Error().Err(err).Msg("Failed to set send deadline")
			span.RecordError(err)
		} else {
			logger.Debug().Str("duration", fmt.Sprint(client.SendDeadline)).Msg(
				"Set send deadline")
		}
	}

	// Set the receive chunk size. This is the size of the buffer that is read from the connection
	// in chunks.
	client.ReceiveChunkSize = clientConfig.ReceiveChunkSize

	logger.Trace().Str("address", client.Address).Msg("New client created")
	client.ID = GetID(
		conn.LocalAddr().Network(), conn.LocalAddr().String(), config.DefaultSeed, logger)

	metrics.ServerConnections.Inc()

	return &client
}

// Send sends data to the server.
func (c *Client) Send(data []byte) (int, *gerr.GatewayDError) {
	_, span := otel.Tracer(config.TracerName).Start(c.ctx, "Send")
	defer span.End()

	sent := 0
	received := len(data)
	for {
		if sent >= received {
			break
		}

		written, err := c.Conn.Write(data)
		if err != nil {
			c.logger.Error().Err(err).Msg("Couldn't send data to the server")
			span.RecordError(err)
			return 0, gerr.ErrClientSendFailed.Wrap(err)
		}

		sent += written
	}

	c.logger.Debug().Fields(
		map[string]interface{}{
			"length":  sent,
			"address": c.Address,
		},
	).Msg("Sent data to server")

	metrics.BytesSentToServer.Observe(float64(sent))

	return sent, nil
}

// Receive receives data from the server.
func (c *Client) Receive() (int, []byte, *gerr.GatewayDError) {
	_, span := otel.Tracer(config.TracerName).Start(c.ctx, "Receive")
	defer span.End()

	var ctx context.Context
	var cancel context.CancelFunc
	if c.ReceiveTimeout > 0 {
		ctx, cancel = context.WithTimeout(c.ctx, c.ReceiveTimeout)
		defer cancel()
	} else {
		ctx = context.Background()
	}

	var received int
	buffer := bytes.NewBuffer(nil)
	// Read the data in chunks.
	select { //nolint:gosimple
	case <-time.After(time.Millisecond):
		for ctx.Err() == nil {
			chunk := make([]byte, c.ReceiveChunkSize)
			read, err := c.Conn.Read(chunk)
			if err != nil {
				c.logger.Error().Err(err).Msg("Couldn't receive data from the server")
				span.RecordError(err)
				metrics.BytesReceivedFromServer.Observe(float64(received))
				return received, buffer.Bytes(), gerr.ErrClientReceiveFailed.Wrap(err)
			}
			received += read
			buffer.Write(chunk[:read])

			if read == 0 || read < c.ReceiveChunkSize {
				break
			}
		}
	}
	metrics.BytesReceivedFromServer.Observe(float64(received))
	return received, buffer.Bytes(), nil
}

// Close closes the connection to the server.
func (c *Client) Close() {
	_, span := otel.Tracer(config.TracerName).Start(c.ctx, "Close")
	defer span.End()

	c.logger.Debug().Str("address", c.Address).Msg("Closing connection to server")
	if c.Conn != nil {
		c.Conn.Close()
	}
	c.ID = ""
	c.Conn = nil
	c.Address = ""
	c.Network = ""

	metrics.ServerConnections.Dec()
}

// IsConnected checks if the client is still connected to the server.
func (c *Client) IsConnected() bool {
	if c != nil && c.ctx.Err() != nil {
		_, span := otel.Tracer(config.TracerName).Start(c.ctx, "IsConnected")
		defer span.End()
	}

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

// RemoteAddr returns the remote address of the client safely.
func (c *Client) RemoteAddr() string {
	if c.Conn != nil && c.Conn.RemoteAddr() != nil {
		return c.Conn.RemoteAddr().String()
	}

	return ""
}

// LocalAddr returns the local address of the client safely.
func (c *Client) LocalAddr() string {
	if c.Conn != nil && c.Conn.LocalAddr() != nil {
		return c.Conn.LocalAddr().String()
	}

	return ""
}
