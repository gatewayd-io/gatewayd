package network

import (
	"bytes"
	"context"
	"crypto/tls"
	"fmt"
	"net"
	"sync"
	"sync/atomic"
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
	Reconnect() error
	Close()
	IsConnected() bool
	RemoteAddr() string
	LocalAddr() string
}

type Client struct {
	conn      *ConnWrapper
	logger    zerolog.Logger
	ctx       context.Context //nolint:containedctx
	connected atomic.Bool
	mu        sync.Mutex

	TCPKeepAlive       bool
	TCPKeepAlivePeriod time.Duration
	ReceiveChunkSize   int
	ReceiveDeadline    time.Duration
	SendDeadline       time.Duration
	ReceiveTimeout     time.Duration
	ID                 string
	Network            string // tcp/udp/unix
	Address            string

	// TLS config
	EnableTLS        bool
	CertFile         string
	KeyFile          string
	HandshakeTimeout time.Duration
}

var _ IClient = (*Client)(nil)

// NewClient creates a new client.
func NewClient(ctx context.Context, clientConfig *config.Client, logger zerolog.Logger) *Client {
	clientCtx, span := otel.Tracer(config.TracerName).Start(ctx, "NewClient")
	defer span.End()

	var client Client

	if clientConfig == nil || clientConfig == (&config.Client{}) {
		return nil
	}

	client.connected.Store(false)
	client.logger = logger

	// Try to resolve the address and log an error if it can't be resolved.
	addr, err := Resolve(clientConfig.Network, clientConfig.Address, logger)
	if err != nil {
		logger.Error().Err(err).Msg("Failed to resolve address")
		span.RecordError(err)
	}

	// Create a resolved client.
	client = Client{
		ctx:              clientCtx,
		mu:               sync.Mutex{},
		logger:           logger,
		Network:          clientConfig.Network,
		Address:          addr,
		EnableTLS:        clientConfig.EnableTLS,
		CertFile:         clientConfig.CertFile,
		KeyFile:          clientConfig.KeyFile,
		HandshakeTimeout: clientConfig.HandshakeTimeout,
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

	// Set the TLS config if enabled.
	var tlsConfig *tls.Config
	if client.EnableTLS {
		tlsConfig, origErr = CreateTLSConfig(client.CertFile, client.KeyFile)
		if origErr != nil {
			client.logger.Error().Err(origErr).Msg("Failed to create TLS config")
		}
		client.logger.Info().Msg("TLS is enabled")
	} else {
		client.logger.Debug().Msg("TLS is disabled")
	}
	client.logger.Debug().Bool("enabled", client.EnableTLS).Msg("Created a new connection")

	client.conn = NewConnWrapper(conn, tlsConfig, client.HandshakeTimeout)

	if client.EnableTLS {
		if err := client.conn.UpgradeToTLS(
			config.ConnectionTypeClient, upgraderFunction); err != nil {
			logger.Error().Err(err).Msg("Failed to upgrade to TLS")
			span.RecordError(err)
			return nil
		}
	}

	client.connected.Store(true)

	// Set the TCP keep alive.
	client.TCPKeepAlive = clientConfig.TCPKeepAlive
	client.TCPKeepAlivePeriod = clientConfig.TCPKeepAlivePeriod

	if c, ok := client.conn.Conn().(*net.TCPConn); ok {
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
		if err := client.conn.SetReadDeadline(time.Now().Add(client.ReceiveDeadline)); err != nil {
			logger.Error().Err(err).Msg("Failed to set receive deadline")
			span.RecordError(err)
		} else {
			logger.Debug().Str("duration", client.ReceiveDeadline.String()).Msg(
				"Set receive deadline")
		}
	}

	// Set the send deadline (timeout).
	client.SendDeadline = clientConfig.SendDeadline
	if client.SendDeadline > 0 {
		if err := client.conn.SetWriteDeadline(time.Now().Add(client.SendDeadline)); err != nil {
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

	if !c.connected.Load() {
		span.RecordError(gerr.ErrClientNotConnected)
		return 0, gerr.ErrClientNotConnected
	}

	sent := 0
	received := len(data)
	for {
		if sent >= received {
			break
		}

		written, err := c.conn.Write(data)
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

	return sent, nil
}

// Receive receives data from the server.
func (c *Client) Receive() (int, []byte, *gerr.GatewayDError) {
	_, span := otel.Tracer(config.TracerName).Start(c.ctx, "Receive")
	defer span.End()

	if !c.connected.Load() {
		span.RecordError(gerr.ErrClientNotConnected)
		return 0, nil, gerr.ErrClientNotConnected
	}

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
	for ctx.Err() == nil {
		chunk := make([]byte, c.ReceiveChunkSize)
		read, err := c.conn.Read(chunk)
		if err != nil {
			c.logger.Error().Err(err).Msg("Couldn't receive data from the server")
			span.RecordError(err)
			return received, buffer.Bytes(), gerr.ErrClientReceiveFailed.Wrap(err)
		}
		received += read
		buffer.Write(chunk[:read])

		if read == 0 || read < c.ReceiveChunkSize {
			break
		}
	}
	return received, buffer.Bytes(), nil
}

// Reconnect reconnects to the server.
func (c *Client) Reconnect() error {
	_, span := otel.Tracer(config.TracerName).Start(c.ctx, "Reconnect")
	defer span.End()

	// Save the current address and network.
	address := c.Address
	network := c.Network

	if c.conn != nil {
		c.Close()
	} else {
		metrics.ServerConnections.Dec()
	}
	c.connected.Store(false)

	// Restore the address and network.
	c.Address = address
	c.Network = network

	conn, err := net.Dial(c.Network, c.Address)
	if err != nil {
		c.logger.Error().Err(err).Msg("Failed to reconnect")
		span.RecordError(err)
		return gerr.ErrClientConnectionFailed.Wrap(err)
	}

	// Set the TLS config if enabled.
	var tlsConfig *tls.Config
	var origErr error
	if c.EnableTLS {
		tlsConfig, origErr = CreateTLSConfig(c.CertFile, c.KeyFile)
		if origErr != nil {
			c.logger.Error().Err(origErr).Msg("Failed to create TLS config")
		}
		c.logger.Info().Msg("TLS is enabled")
	} else {
		c.logger.Debug().Msg("TLS is disabled")
	}

	c.conn = NewConnWrapper(conn, tlsConfig, c.HandshakeTimeout)

	if c.EnableTLS {
		if err := c.conn.UpgradeToTLS(config.ConnectionTypeClient, upgraderFunction); err != nil {
			c.logger.Error().Err(err).Msg("Failed to upgrade to TLS")
			span.RecordError(err)
			return err
		}
	}

	c.ID = GetID(
		conn.LocalAddr().Network(), conn.LocalAddr().String(), config.DefaultSeed, c.logger)
	c.connected.Store(true)
	c.logger.Debug().Str("address", c.Address).Msg("Reconnected to server")
	metrics.ServerConnections.Inc()

	return nil
}

// Close closes the connection to the server.
func (c *Client) Close() {
	_, span := otel.Tracer(config.TracerName).Start(c.ctx, "Close")
	defer span.End()

	c.mu.Lock()
	defer c.mu.Unlock()

	// Set the deadline to now so that the connection is closed immediately.
	// This will stop all the Conn.Read() and Conn.Write() calls.
	// Ref: https://groups.google.com/g/golang-nuts/c/VPVWFrpIEyo
	if c.conn != nil {
		if err := c.conn.SetDeadline(time.Now()); err != nil {
			c.logger.Error().Err(err).Msg("Failed to set deadline")
			span.RecordError(err)
		}
	}

	c.connected.Store(false)
	c.logger.Debug().Str("address", c.Address).Msg("Closing connection to server")
	if c.conn != nil {
		if err := c.conn.Close(); err != nil {
			c.logger.Error().Err(err).Msg("Failed to close connection")
			span.RecordError(err)
		}
	}
	c.ID = ""
	c.conn = nil
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

	if c != nil && c.conn == nil || c.ID == "" {
		c.logger.Debug().Fields(
			map[string]interface{}{
				"address": c.Address,
				"reason":  "connection is nil or invalid",
			}).Msg("Connection to server is closed")
		return false
	}

	return c.connected.Load()
}

// RemoteAddr returns the remote address of the client safely.
func (c *Client) RemoteAddr() string {
	if !c.connected.Load() {
		return ""
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	if c.conn != nil && c.conn.RemoteAddr() != nil {
		return c.conn.RemoteAddr().String()
	}

	return ""
}

// LocalAddr returns the local address of the client safely.
func (c *Client) LocalAddr() string {
	if !c.connected.Load() {
		return ""
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	if c.conn != nil && c.conn.LocalAddr() != nil {
		return c.conn.LocalAddr().String()
	}

	return ""
}

// upgraderFunction is the function that is used to upgrade the connection to TLS.
// For example, in Postgres, this function can be used to send the SSLRequest message
// and wait for the server to respond with a 'S' message to indicate that it supports
// TLS. The client then upgrades the connection to TLS.
func upgraderFunction(c net.Conn) error {
	// Send the SSLRequest message.
	// The SSLRequest message is sent by the client to request an SSL connection.
	// The server responds with a 'S' message to indicate that it supports TLS.
	// The client then upgrades the connection to TLS.
	// Ref: https://www.postgresql.org/docs/current/protocol-flow.html
	// Ref: https://www.postgresql.org/docs/current/protocol-message-formats.html

	sslRequest := []byte{0x00, 0x00, 0x00, 0x08, 0x04, 0xd2, 0x16, 0x2f}
	sent, err := c.Write(sslRequest)
	if err != nil || sent != len(sslRequest) {
		return err
	}

	// Read the response from the server.
	serverResponse := make([]byte, 1)
	if read, err := c.Read(serverResponse); err != nil || read != 1 {
		return err
	}

	// Check if the server supports TLS.
	if serverResponse[0] != 'S' {
		return fmt.Errorf("server doesn't support TLS")
	}

	return nil
}
