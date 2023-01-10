package network

import (
	"testing"
	"time"

	embeddedpostgres "github.com/fergusstrange/embedded-postgres"
	"github.com/gatewayd-io/gatewayd/config"
	"github.com/gatewayd-io/gatewayd/logging"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestNewClient tests the NewClient function.
func TestNewClient(t *testing.T) {
	postgres := embeddedpostgres.NewDatabase()
	if err := postgres.Start(); err != nil {
		t.Fatal(err)
	}

	defer func() {
		if err := postgres.Stop(); err != nil {
			t.Fatal(err)
		}
	}()

	cfg := logging.LoggerConfig{
		Output:     config.Console,
		TimeFormat: zerolog.TimeFormatUnix,
		Level:      zerolog.DebugLevel,
		NoColor:    true,
	}

	logger := logging.NewLogger(cfg)

	keepAlive, err := time.ParseDuration(config.DefaultTCPKeepAlivePeriod)
	require.NoError(t, err)

	client := NewClient(
		&config.Client{
			Network:            "tcp",
			Address:            "localhost:5432",
			ReceiveBufferSize:  config.DefaultBufferSize,
			ReceiveChunkSize:   config.DefaultChunkSize,
			ReceiveDeadline:    config.DefaultReceiveDeadline,
			SendDeadline:       config.DefaultSendDeadline,
			TCPKeepAlive:       false,
			TCPKeepAlivePeriod: keepAlive,
		},
		logger)
	defer client.Close()

	assert.Equal(t, "tcp", client.Network)
	assert.Equal(t, "127.0.0.1:5432", client.Address)
	assert.Equal(t, config.DefaultBufferSize, client.ReceiveBufferSize)
	assert.NotEmpty(t, client.ID)
	assert.NotNil(t, client.Conn)
}

// TestSend tests the Send function.
func TestSend(t *testing.T) {
	postgres := embeddedpostgres.NewDatabase()
	if err := postgres.Start(); err != nil {
		t.Fatal(err)
	}

	defer func() {
		if err := postgres.Stop(); err != nil {
			t.Fatal(err)
		}
	}()

	cfg := logging.LoggerConfig{
		Output:     config.Console,
		TimeFormat: zerolog.TimeFormatUnix,
		Level:      zerolog.DebugLevel,
		NoColor:    true,
	}

	logger := logging.NewLogger(cfg)

	keepAlive, err := time.ParseDuration(config.DefaultTCPKeepAlivePeriod)
	require.NoError(t, err)

	client := NewClient(
		&config.Client{
			Network:            "tcp",
			Address:            "localhost:5432",
			ReceiveBufferSize:  config.DefaultBufferSize,
			ReceiveChunkSize:   config.DefaultChunkSize,
			ReceiveDeadline:    config.DefaultReceiveDeadline,
			SendDeadline:       config.DefaultSendDeadline,
			TCPKeepAlive:       false,
			TCPKeepAlivePeriod: keepAlive,
		},
		logger)
	defer client.Close()

	assert.NotNil(t, client)
	packet := CreatePostgreSQLPacket('Q', []byte("select 1;"))
	sent, err := client.Send(packet)
	assert.Nil(t, err)
	assert.Equal(t, len(packet), sent)
}

// TestReceive tests the Receive function.
func TestReceive(t *testing.T) {
	postgres := embeddedpostgres.NewDatabase()
	if err := postgres.Start(); err != nil {
		t.Fatal(err)
	}

	defer func() {
		if err := postgres.Stop(); err != nil {
			t.Fatal(err)
		}
	}()

	cfg := logging.LoggerConfig{
		Output:     config.Console,
		TimeFormat: zerolog.TimeFormatUnix,
		Level:      zerolog.DebugLevel,
		NoColor:    true,
	}

	logger := logging.NewLogger(cfg)

	keepAlive, err := time.ParseDuration(config.DefaultTCPKeepAlivePeriod)
	require.NoError(t, err)

	client := NewClient(
		&config.Client{
			Network:            "tcp",
			Address:            "localhost:5432",
			ReceiveBufferSize:  config.DefaultBufferSize,
			ReceiveChunkSize:   config.DefaultChunkSize,
			ReceiveDeadline:    config.DefaultReceiveDeadline,
			SendDeadline:       config.DefaultSendDeadline,
			TCPKeepAlive:       false,
			TCPKeepAlivePeriod: keepAlive,
		},
		logger)
	defer client.Close()

	assert.NotNil(t, client)
	packet := CreatePgStartupPacket()
	sent, err := client.Send(packet)
	assert.Nil(t, err)
	assert.Equal(t, len(packet), sent)

	size, data, err := client.Receive()
	msg := "\x00\x00\x00\x03"
	assert.Equal(t, 9, size)
	assert.Equal(t, len(data[:size]), size)
	assert.Nil(t, err)
	assert.NotEmpty(t, data[:size])
	assert.Equal(t, msg, string(data[5:size]))
	// AuthenticationOk
	assert.Equal(t, uint8(0x52), data[0])
}

// TestClose tests the Close function.
func TestClose(t *testing.T) {
	postgres := embeddedpostgres.NewDatabase()
	if err := postgres.Start(); err != nil {
		t.Fatal(err)
	}

	defer func() {
		if err := postgres.Stop(); err != nil {
			t.Fatal(err)
		}
	}()

	cfg := logging.LoggerConfig{
		Output:     config.Console,
		TimeFormat: zerolog.TimeFormatUnix,
		Level:      zerolog.DebugLevel,
		NoColor:    true,
	}

	logger := logging.NewLogger(cfg)

	keepAlive, err := time.ParseDuration(config.DefaultTCPKeepAlivePeriod)
	require.NoError(t, err)

	client := NewClient(
		&config.Client{
			Network:            "tcp",
			Address:            "localhost:5432",
			ReceiveBufferSize:  config.DefaultBufferSize,
			ReceiveChunkSize:   config.DefaultChunkSize,
			ReceiveDeadline:    config.DefaultReceiveDeadline,
			SendDeadline:       config.DefaultSendDeadline,
			TCPKeepAlive:       false,
			TCPKeepAlivePeriod: keepAlive,
		},
		logger)
	assert.NotNil(t, client)
	client.Close()
	assert.Equal(t, "", client.ID)
	assert.Equal(t, "", client.Network)
	assert.Equal(t, "", client.Address)
	assert.Nil(t, client.Conn)
	assert.Equal(t, config.DefaultBufferSize, client.ReceiveBufferSize)
}
