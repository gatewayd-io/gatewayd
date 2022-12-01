package network

import (
	"sync"
	"testing"

	embeddedpostgres "github.com/fergusstrange/embedded-postgres"
	"github.com/gatewayd-io/gatewayd/logging"
	"github.com/panjf2000/gnet/v2"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
)

//nolint:funlen
func TestRunServer(t *testing.T) {
	postgres := embeddedpostgres.NewDatabase()
	if err := postgres.Start(); err != nil {
		t.Fatal(err)
	}

	// Create a logger
	cfg := logging.LoggerConfig{
		Output:     nil,
		TimeFormat: zerolog.TimeFormatUnix,
		Level:      zerolog.DebugLevel,
		NoColor:    true,
	}

	logger := logging.NewLogger(cfg)

	hooksConfig := NewHookConfig()

	onIncomingTraffic := func(params Signature) Signature {
		if params["buffer"] == nil {
			t.Fatal("buffer is nil")
		}

		logger.Info().Msg("Incoming traffic")
		if buf, ok := params["buffer"].([]byte); ok {
			assert.Equal(t, CreatePgStartupPacket(), buf)
		} else {
			t.Fatal("buffer is not a []byte")
		}
		assert.Nil(t, params["error"])
		return nil
	}
	hooksConfig.Add(OnIncomingTraffic, 1, onIncomingTraffic)

	onOutgoingTraffic := func(params Signature) Signature {
		if params["buffer"] == nil {
			t.Fatal("buffer is nil")
		}

		logger.Info().Msg("Outgoing traffic")
		if buf, ok := params["buffer"].([]byte); ok {
			assert.Equal(
				t, CreatePostgreSQLPacket('R', []byte{0x0, 0x0, 0x0, 0x3}), buf)
		} else {
			t.Fatal("buffer is not a []byte")
		}
		assert.Nil(t, params["error"])
		return nil
	}
	hooksConfig.Add(OnOutgoingTraffic, 1, onOutgoingTraffic)

	// Create a connection pool
	pool := NewEmptyPool(logger)
	client1 := NewClient("tcp", "localhost:5432", DefaultBufferSize, logger)
	pool.Put(client1.ID, client1)
	client2 := NewClient("tcp", "localhost:5432", DefaultBufferSize, logger)
	pool.Put(client2.ID, client2)

	// Create a proxy with a fixed buffer pool
	proxy := NewProxy(pool, NewHookConfig(), false, false, &Client{
		Network:           "tcp",
		Address:           "localhost:5432",
		ReceiveBufferSize: DefaultBufferSize,
	}, logger)

	// Create a server
	server := NewServer(
		"tcp",
		"127.0.0.1:15432",
		0,
		0,
		DefaultTickInterval,
		[]gnet.Option{
			gnet.WithMulticore(true),
		},
		proxy,
		logger,
		hooksConfig,
	)
	assert.NotNil(t, server)

	var waitGroup sync.WaitGroup
	waitGroup.Add(2)

	go func(t *testing.T, server *Server) {
		t.Helper()
		defer waitGroup.Done()

		err := server.Run()
		assert.Nil(t, err)
	}(t, server)

	go func(t *testing.T, server *Server) {
		t.Helper()
		defer waitGroup.Done()

		for {
			if server.IsRunning() {
				client := NewClient("tcp", "127.0.0.1:15432", DefaultBufferSize, logger)
				defer client.Close()

				assert.NotNil(t, client)
				err := client.Send(CreatePgStartupPacket())
				assert.Nil(t, err)

				// The server should respond with a 'R' packet
				size, data, err := client.Receive()
				msg := []byte{0x0, 0x0, 0x0, 0x3}
				// This includes the message type, length and the message itself
				assert.Equal(t, 9, size)
				assert.Equal(t, len(data[:size]), size)
				assert.Nil(t, err)
				packetSize := int(data[1])<<24 | int(data[2])<<16 | int(data[3])<<8 | int(data[4])
				assert.Equal(t, 8, packetSize)
				assert.NotEmpty(t, data[:size])
				assert.Equal(t, msg, data[5:size])
				// AuthenticationOk
				assert.Equal(t, uint8(0x52), data[0])

				// Clean up
				server.Shutdown()
				assert.NoError(t, postgres.Stop())
				return
			}
		}
	}(t, server)

	waitGroup.Wait()
}
