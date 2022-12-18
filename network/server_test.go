package network

import (
	"context"
	"encoding/base64"
	"sync"
	"testing"

	embeddedpostgres "github.com/fergusstrange/embedded-postgres"
	"github.com/gatewayd-io/gatewayd/logging"
	"github.com/gatewayd-io/gatewayd/plugin"
	"github.com/gatewayd-io/gatewayd/pool"
	"github.com/panjf2000/gnet/v2"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/structpb"
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

	hooksConfig := plugin.NewHookConfig()

	onIngressTraffic := func(
		ctx context.Context,
		params *structpb.Struct,
		opts ...grpc.CallOption,
	) (*structpb.Struct, error) {
		paramsMap := params.AsMap()
		if paramsMap["buffer"] == nil {
			t.Fatal("buffer is nil")
		}

		logger.Info().Msg("Ingress traffic")
		// Decode the buffer
		// The buffer is []byte, but it is base64-encoded as a string
		// via using the structpb.NewStruct function
		if buf, ok := paramsMap["buffer"].(string); ok {
			if buffer, err := base64.StdEncoding.DecodeString(buf); err == nil {
				assert.Equal(t, CreatePgStartupPacket(), buffer)
			} else {
				t.Fatal(err)
			}
		} else {
			t.Fatal("buffer is not a []byte")
		}
		assert.Empty(t, paramsMap["error"])
		return params, nil
	}
	hooksConfig.Add(plugin.OnIngressTraffic, 1, onIngressTraffic)

	onEgressTraffic := func(
		ctx context.Context,
		params *structpb.Struct,
		opts ...grpc.CallOption,
	) (*structpb.Struct, error) {
		paramsMap := params.AsMap()
		if paramsMap["response"] == nil {
			t.Fatal("response is nil")
		}

		logger.Info().Msg("Egress traffic")
		if buf, ok := paramsMap["response"].(string); ok {
			if buffer, err := base64.StdEncoding.DecodeString(buf); err == nil {
				assert.Equal(t, CreatePostgreSQLPacket('R', []byte{0x0, 0x0, 0x0, 0x3}), buffer)
			} else {
				t.Fatal(err)
			}
		} else {
			t.Fatal("response is not a []byte")
		}
		assert.Empty(t, paramsMap["error"])
		return params, nil
	}
	hooksConfig.Add(plugin.OnEgressTraffic, 1, onEgressTraffic)

	// Create a connection pool
	pool := pool.NewPool(2)
	client1 := NewClient("tcp", "localhost:5432", DefaultBufferSize, logger)
	pool.Put(client1.ID, client1)
	client2 := NewClient("tcp", "localhost:5432", DefaultBufferSize, logger)
	pool.Put(client2.ID, client2)

	// Create a proxy with a fixed buffer pool
	proxy := NewProxy(pool, hooksConfig, false, false, &Client{
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
				sent, err := client.Send(CreatePgStartupPacket())
				assert.Nil(t, err)
				assert.Equal(t, len(CreatePgStartupPacket()), sent)

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
