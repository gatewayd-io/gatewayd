package network

import (
	"context"
	"testing"
	"time"

	"github.com/gatewayd-io/gatewayd/config"
	"github.com/gatewayd-io/gatewayd/logging"
	"github.com/gatewayd-io/gatewayd/testhelpers"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestNewClient tests the NewClient function.
func TestNewClient(t *testing.T) {
	ctx := context.Background()
	postgresHostIP, postgresMappedPort := testhelpers.SetupPostgreSQLTestContainer(ctx, t)
	client, _ := CreateNewClient(ctx, t, postgresHostIP, postgresMappedPort.Port(), nil)
	defer client.Close()

	assert.NotNil(t, client)
	assert.Equal(t, "tcp", client.Network)
	assert.Equal(t, "127.0.0.1:"+postgresMappedPort.Port(), client.Address)
	assert.NotEmpty(t, client.ID)
	assert.NotNil(t, client.conn)
}

// TestSend tests the Send function.
func TestSend(t *testing.T) {
	ctx := context.Background()
	postgresHostIP, postgresMappedPort := testhelpers.SetupPostgreSQLTestContainer(ctx, t)
	client, _ := CreateNewClient(ctx, t, postgresHostIP, postgresMappedPort.Port(), nil)
	defer client.Close()

	assert.NotNil(t, client)
	packet := CreatePostgreSQLPacket('Q', []byte("select 1;"))
	sent, err := client.Send(packet)
	assert.Nil(t, err)
	assert.Len(t, packet, sent)
}

// TestReceive tests the Receive function.
func TestReceive(t *testing.T) {
	ctx := context.Background()
	postgresHostIP, postgresMappedPort := testhelpers.SetupPostgreSQLTestContainer(ctx, t)
	client, _ := CreateNewClient(ctx, t, postgresHostIP, postgresMappedPort.Port(), nil)
	defer client.Close()

	assert.NotNil(t, client)
	packet := CreatePgStartupPacket()
	sent, err := client.Send(packet)
	assert.Nil(t, err)
	assert.Len(t, packet, sent)

	size, data, err := client.Receive()
	// AuthenticationSASL
	msg := "\x00\x00\x00\nSCRAM-SHA-256\x00\x00"
	assert.Equal(t, 24, size)
	assert.Len(t, data[:size], size)
	assert.Nil(t, err)
	assert.NotEmpty(t, data[:size])
	assert.Equal(t, msg, string(data[5:size]))
	// AuthenticationOk
	assert.Equal(t, uint8(0x52), data[0])
}

// TestClose tests the Close function.
func TestClose(t *testing.T) {
	ctx := context.Background()
	postgresHostIP, postgresMappedPort := testhelpers.SetupPostgreSQLTestContainer(ctx, t)
	client, _ := CreateNewClient(ctx, t, postgresHostIP, postgresMappedPort.Port(), nil)

	assert.NotNil(t, client)
	client.Close()
	assert.Equal(t, "", client.ID)
	assert.Equal(t, "", client.Network)
	assert.Equal(t, "", client.Address)
	assert.Nil(t, client.conn)
}

// TestIsConnected tests the IsConnected function.
func TestIsConnected(t *testing.T) {
	postgresHostIP, postgresMappedPort := testhelpers.SetupPostgreSQLTestContainer(context.Background(), t)
	client, _ := CreateNewClient(context.Background(), t, postgresHostIP, postgresMappedPort.Port(), nil)

	assert.True(t, client.IsConnected())
	client.Close()
	assert.False(t, client.IsConnected())
}

func TestReconnect(t *testing.T) {
	ctx := context.Background()
	postgresHostIP, postgresMappedPort := testhelpers.SetupPostgreSQLTestContainer(ctx, t)
	client, _ := CreateNewClient(ctx, t, postgresHostIP, postgresMappedPort.Port(), nil)
	defer client.Close()

	assert.NotNil(t, client)
	assert.NotNil(t, client.conn)
	assert.NotEmpty(t, client.ID)
	localAddr := client.LocalAddr()
	assert.NotEmpty(t, localAddr)

	require.NoError(t, client.Reconnect())
	assert.NotNil(t, client)
	assert.NotNil(t, client.conn)
	assert.NotEmpty(t, client.ID)
	assert.NotEmpty(t, client.LocalAddr())
	assert.NotEqual(t, localAddr, client.LocalAddr()) // This is a new connection.
}

func BenchmarkNewClient(b *testing.B) {
	cfg := logging.LoggerConfig{
		Output:            []config.LogOutput{config.Console},
		TimeFormat:        zerolog.TimeFormatUnix,
		ConsoleTimeFormat: time.RFC3339,
		Level:             zerolog.DebugLevel,
		NoColor:           true,
	}

	logger := logging.NewLogger(context.Background(), cfg)
	for i := 0; i < b.N; i++ {
		client := NewClient(context.Background(), &config.Client{
			Network:            "tcp",
			Address:            "localhost:5432",
			ReceiveChunkSize:   config.DefaultChunkSize,
			ReceiveDeadline:    config.DefaultReceiveDeadline,
			SendDeadline:       config.DefaultSendDeadline,
			DialTimeout:        config.DefaultDialTimeout,
			TCPKeepAlive:       false,
			TCPKeepAlivePeriod: config.DefaultTCPKeepAlivePeriod,
		}, logger, nil)
		client.Close()
	}
}

func BenchmarkSend(b *testing.B) {
	logger := logging.NewLogger(context.Background(), logging.LoggerConfig{
		Output:            []config.LogOutput{config.Console},
		TimeFormat:        zerolog.TimeFormatUnix,
		ConsoleTimeFormat: time.RFC3339,
		Level:             zerolog.DebugLevel,
		NoColor:           true,
	})

	client := NewClient(
		context.Background(),
		&config.Client{
			Network:            "tcp",
			Address:            "localhost:5432",
			ReceiveChunkSize:   config.DefaultChunkSize,
			ReceiveDeadline:    config.DefaultReceiveDeadline,
			SendDeadline:       config.DefaultSendDeadline,
			DialTimeout:        config.DefaultDialTimeout,
			TCPKeepAlive:       false,
			TCPKeepAlivePeriod: config.DefaultTCPKeepAlivePeriod,
		},
		logger, nil)
	defer client.Close()

	packet := CreatePgStartupPacket()
	for i := 0; i < b.N; i++ {
		client.Send(packet) //nolint:errcheck
	}
}

func BenchmarkReceive(b *testing.B) {
	logger := logging.NewLogger(context.Background(), logging.LoggerConfig{
		Output:            []config.LogOutput{config.Console},
		TimeFormat:        zerolog.TimeFormatUnix,
		ConsoleTimeFormat: time.RFC3339,
		Level:             zerolog.DebugLevel,
		NoColor:           true,
	})

	client := NewClient(
		context.Background(),
		&config.Client{
			Network:            "tcp",
			Address:            "localhost:5432",
			ReceiveChunkSize:   config.DefaultChunkSize,
			ReceiveDeadline:    config.DefaultReceiveDeadline,
			ReceiveTimeout:     1 * time.Millisecond,
			SendDeadline:       config.DefaultSendDeadline,
			DialTimeout:        config.DefaultDialTimeout,
			TCPKeepAlive:       false,
			TCPKeepAlivePeriod: config.DefaultTCPKeepAlivePeriod,
		},
		logger, nil)
	defer client.Close()

	packet := CreatePgStartupPacket()
	client.Send(packet) //nolint:errcheck
	for i := 0; i < b.N; i++ {
		client.Receive() //nolint:errcheck
	}
}

func BenchmarkIsConnected(b *testing.B) {
	logger := logging.NewLogger(context.Background(), logging.LoggerConfig{
		Output:            []config.LogOutput{config.Console},
		TimeFormat:        zerolog.TimeFormatUnix,
		ConsoleTimeFormat: time.RFC3339,
		Level:             zerolog.DebugLevel,
		NoColor:           true,
	})

	client := NewClient(
		context.Background(),
		&config.Client{
			Network:            "tcp",
			Address:            "localhost:5432",
			ReceiveChunkSize:   config.DefaultChunkSize,
			ReceiveDeadline:    config.DefaultReceiveDeadline,
			SendDeadline:       config.DefaultSendDeadline,
			DialTimeout:        config.DefaultDialTimeout,
			TCPKeepAlive:       false,
			TCPKeepAlivePeriod: config.DefaultTCPKeepAlivePeriod,
		},
		logger, nil)
	defer client.Close()

	for i := 0; i < b.N; i++ {
		client.IsConnected()
	}
}
