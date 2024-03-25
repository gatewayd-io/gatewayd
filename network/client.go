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
}

type Client struct {
	net.Conn

	Logger zerolog.Logger
	ctx    context.Context //nolint:containedctx

	TCPKeepAlive       bool
	TCPKeepAlivePeriod time.Duration
	ReceiveChunkSize   int
	ReceiveDeadline    time.Duration
	SendDeadline       time.Duration
	ID                 string
	Network            string // tcp/udp/unix
	Address            string
}

var _ IClient = &Client{}

// ClientFromConfig create Client from config.Client
func ClientFromConfig(config *config.Client) *Client {
	return &Client{
		Network:            config.Network,
		Address:            config.Address,
		TCPKeepAlive:       config.TCPKeepAlive,
		TCPKeepAlivePeriod: config.GetTCPKeepAlivePeriod(),
		ReceiveChunkSize:   config.GetReceiveChunkSize(),
		ReceiveDeadline:    config.GetReceiveDeadline(),
		SendDeadline:       config.GetSendDeadline(),
	}
}

// NewClient creates a new client.
func NewClient(ctx context.Context, client *Client) *Client {
	clientCtx, span := otel.Tracer(config.TracerName).Start(ctx, "NewClient")
	defer span.End()

	if client == nil || client == (&Client{}) {
		return nil
	}

	// Try to resolve the address and log an error if it can't be resolved.
	addr, err := Resolve(client.Network, client.Address, client.Logger)
	if err != nil {
		client.Logger.Error().Err(err).Msg("Failed to resolve address")
		span.RecordError(err)
	}

	// Fall back to the original network and address if the address can't be resolved.
	if addr == "" || client.Network == "" {
		addr = client.Address
	}

	// Create a resolved client.
	clt := Client{
		ctx:                clientCtx,
		Address:            addr,
		Network:            client.Network,
		Logger:             client.Logger,
		TCPKeepAlive:       client.TCPKeepAlive,
		TCPKeepAlivePeriod: client.TCPKeepAlivePeriod,
		ReceiveDeadline:    client.ReceiveDeadline,
		SendDeadline:       client.SendDeadline,
		ReceiveChunkSize:   client.ReceiveChunkSize,
	}

	// Create a new connection.
	conn, origErr := net.Dial(clt.Network, clt.Address)
	if origErr != nil {
		err := gerr.ErrClientConnectionFailed.Wrap(origErr)
		clt.Logger.Error().Err(err).Msg("Failed to create a new connection")
		span.RecordError(err)
		return nil
	}

	clt.Conn = conn

	if c, ok := clt.Conn.(*net.TCPConn); ok {
		if err := c.SetKeepAlive(clt.TCPKeepAlive); err != nil {
			clt.Logger.Error().Err(err).Msg("Failed to set keep alive")
			span.RecordError(err)
		} else {
			if err := c.SetKeepAlivePeriod(clt.TCPKeepAlivePeriod); err != nil {
				clt.Logger.Error().Err(err).Msg("Failed to set keep alive period")
				span.RecordError(err)
			}
		}
	}

	// Set the receive deadline (timeout).
	if clt.ReceiveDeadline > 0 {
		if err := clt.Conn.SetReadDeadline(time.Now().Add(clt.ReceiveDeadline)); err != nil {
			clt.Logger.Error().Err(err).Msg("Failed to set receive deadline")
			span.RecordError(err)
		} else {
			clt.Logger.Debug().Str("duration", fmt.Sprint(clt.ReceiveDeadline.String())).Msg(
				"Set receive deadline")
		}
	}

	// Set the send deadline (timeout).
	if clt.SendDeadline > 0 {
		if err := clt.Conn.SetWriteDeadline(time.Now().Add(clt.SendDeadline)); err != nil {
			clt.Logger.Error().Err(err).Msg("Failed to set send deadline")
			span.RecordError(err)
		} else {
			clt.Logger.Debug().Str("duration", fmt.Sprint(clt.SendDeadline)).Msg(
				"Set send deadline")
		}
	}

	// Set the receive chunk size. This is the size of the buffer that is read from the connection
	// in chunks.
	clt.Logger.Trace().Str("address", clt.Address).Msg("New client created")
	clt.ID = GetID(
		conn.LocalAddr().Network(), conn.LocalAddr().String(), config.DefaultSeed, clt.Logger)

	metrics.ServerConnections.Inc()

	return &clt
}

// Send sends data to the server.
func (c *Client) Send(data []byte) (int, *gerr.GatewayDError) {
	_, span := otel.Tracer(config.TracerName).Start(c.ctx, "Send")
	defer span.End()

	sent, err := c.Conn.Write(data)
	if err != nil {
		c.Logger.Error().Err(err).Msg("Couldn't send data to the server")
		span.RecordError(err)
		return 0, gerr.ErrClientSendFailed.Wrap(err)
	}
	c.Logger.Debug().Fields(
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

	var received int
	buffer := bytes.NewBuffer(nil)
	// Read the data in chunks.
	select { //nolint:gosimple
	case <-time.After(time.Millisecond):
		for {
			chunk := make([]byte, c.ReceiveChunkSize)
			read, err := c.Conn.Read(chunk)
			if err != nil {
				c.Logger.Error().Err(err).Msg("Couldn't receive data from the server")
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

	c.Logger.Debug().Str("address", c.Address).Msg("Closing connection to server")
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
		c.Logger.Debug().Fields(
			map[string]interface{}{
				"address": c.Address,
				"reason":  "client is nil",
			}).Msg("Connection to server is closed")
		return false
	}

	if c != nil && c.Conn == nil || c.ID == "" {
		c.Logger.Debug().Fields(
			map[string]interface{}{
				"address": c.Address,
				"reason":  "connection is nil or invalid",
			}).Msg("Connection to server is closed")
		return false
	}

	if n, err := c.Read([]byte{}); n == 0 && err != nil {
		c.Logger.Debug().Fields(
			map[string]interface{}{
				"address": c.Address,
				"reason":  "read 0 bytes",
			}).Msg("Connection to server is closed")
		return false
	}

	return true
}
