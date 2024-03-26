package network

import (
	"context"
	"testing"

	"github.com/gatewayd-io/gatewayd/config"
	"github.com/gatewayd-io/gatewayd/logging"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
)

// CreateNewClient creates a new client for testing.
func CreateNewClient(t *testing.T) *Client {
	t.Helper()

	logger := logging.NewLogger(context.Background(), logging.LoggerConfig{
		Output:            []config.LogOutput{config.Console},
		TimeFormat:        zerolog.TimeFormatUnix,
		ConsoleTimeFormat: config.DefaultConsoleTimeFormat,
		Level:             zerolog.DebugLevel,
		NoColor:           true,
	})

	client := NewClient(
		context.Background(),
		&Client{
			Network:            "tcp",
			Address:            "localhost:5432",
			ReceiveChunkSize:   config.DefaultChunkSize,
			ReceiveDeadline:    config.DefaultReceiveDeadline,
			SendDeadline:       config.DefaultSendDeadline,
			TCPKeepAlive:       false,
			TCPKeepAlivePeriod: config.DefaultTCPKeepAlivePeriod,
			Logger:             logger,
		})

	return client
}

// TestNewClient tests the NewClient function.
func TestNewClient(t *testing.T) {
	client := CreateNewClient(t)
	defer client.Close()

	assert.Equal(t, "tcp", client.Network)
	assert.Equal(t, "127.0.0.1:5432", client.Address)
	assert.NotEmpty(t, client.ID)
	assert.NotNil(t, client.Conn)
}

// TestSend tests the Send function.
func TestSend(t *testing.T) {
	client := CreateNewClient(t)
	defer client.Close()

	assert.NotNil(t, client)
	packet := CreatePostgreSQLPacket('Q', []byte("select 1;"))
	sent, err := client.Send(packet)
	assert.Nil(t, err)
	assert.Equal(t, len(packet), sent)
}

// TestReceive tests the Receive function.
func TestReceive(t *testing.T) {
	client := CreateNewClient(t)
	defer client.Close()

	assert.NotNil(t, client)
	packet := CreatePgStartupPacket()
	sent, err := client.Send(packet)
	assert.Nil(t, err)
	assert.Equal(t, len(packet), sent)

	size, data, err := client.Receive()
	// AuthenticationSASL
	msg := "\x00\x00\x00\nSCRAM-SHA-256\x00\x00"
	assert.Equal(t, 24, size)
	assert.Equal(t, len(data[:size]), size)
	assert.Nil(t, err)
	assert.NotEmpty(t, data[:size])
	assert.Equal(t, msg, string(data[5:size]))
	// AuthenticationOk
	assert.Equal(t, uint8(0x52), data[0])
}

// TestClose tests the Close function.
func TestClose(t *testing.T) {
	client := CreateNewClient(t)

	assert.NotNil(t, client)
	client.Close()
	assert.Equal(t, "", client.ID)
	assert.Equal(t, "", client.Network)
	assert.Equal(t, "", client.Address)
	assert.Nil(t, client.Conn)
}

// TestIsConnected tests the IsConnected function.
func TestIsConnected(t *testing.T) {
	client := CreateNewClient(t)

	assert.True(t, client.IsConnected())
	client.Close()
	assert.False(t, client.IsConnected())
}
