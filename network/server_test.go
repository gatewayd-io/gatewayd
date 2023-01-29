package network

import (
	"context"
	"encoding/base64"
	"errors"
	"testing"

	embeddedpostgres "github.com/fergusstrange/embedded-postgres"
	"github.com/gatewayd-io/gatewayd/config"
	"github.com/gatewayd-io/gatewayd/logging"
	"github.com/gatewayd-io/gatewayd/plugin"
	"github.com/gatewayd-io/gatewayd/pool"
	"github.com/panjf2000/gnet/v2"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/structpb"
)

// TestRunServer tests an entire server run with a single client connection and hooks.
func TestRunServer(t *testing.T) {
	errs := make(chan error)

	postgres := embeddedpostgres.NewDatabase()
	if err := postgres.Start(); err != nil {
		errs <- err
	}

	logger := logging.NewLogger(logging.LoggerConfig{
		Output:     config.Console,
		TimeFormat: zerolog.TimeFormatUnix,
		Level:      zerolog.DebugLevel,
		NoColor:    true,
	})

	pluginRegistry := plugin.NewRegistry(config.Loose, config.PassDown, config.Accept, logger)

	onTrafficFromClient := func(
		ctx context.Context,
		params *structpb.Struct,
		opts ...grpc.CallOption,
	) (*structpb.Struct, error) {
		paramsMap := params.AsMap()
		if paramsMap["request"] == nil {
			errs <- errors.New("request is nil") //nolint:goerr113
		}

		logger.Info().Msg("Ingress traffic")
		// Decode the request.
		// The request is []byte, but it is base64-encoded as a string
		// via using the structpb.NewStruct function.
		if req, ok := paramsMap["request"].(string); ok {
			if request, err := base64.StdEncoding.DecodeString(req); err == nil {
				assert.Equal(t, CreatePgStartupPacket(), request)
			} else {
				errs <- err
			}
		} else {
			errs <- errors.New("request is not a []byte") //nolint:goerr113
		}
		assert.Empty(t, paramsMap["error"])
		return params, nil
	}
	pluginRegistry.AddHook(plugin.OnTrafficFromClient, 1, onTrafficFromClient)

	onTrafficToServer := func(
		ctx context.Context,
		params *structpb.Struct,
		opts ...grpc.CallOption,
	) (*structpb.Struct, error) {
		paramsMap := params.AsMap()
		if paramsMap["request"] == nil {
			errs <- errors.New("request is nil") //nolint:goerr113
		}

		logger.Info().Msg("Ingress traffic")
		// Decode the request.
		// The request is []byte, but it is base64-encoded as a string
		// via using the structpb.NewStruct function.
		if req, ok := paramsMap["request"].(string); ok {
			if request, err := base64.StdEncoding.DecodeString(req); err == nil {
				assert.Equal(t, CreatePgStartupPacket(), request)
			} else {
				errs <- err
			}
		} else {
			errs <- errors.New("request is not a []byte") //nolint:goerr113
		}
		assert.Empty(t, paramsMap["error"])
		return params, nil
	}
	pluginRegistry.AddHook(plugin.OnTrafficToServer, 1, onTrafficToServer)

	onTrafficFromServer := func(
		ctx context.Context,
		params *structpb.Struct,
		opts ...grpc.CallOption,
	) (*structpb.Struct, error) {
		paramsMap := params.AsMap()
		if paramsMap["response"] == nil {
			errs <- errors.New("response is nil") //nolint:goerr113
		}

		logger.Info().Msg("Egress traffic")
		if resp, ok := paramsMap["response"].(string); ok {
			if response, err := base64.StdEncoding.DecodeString(resp); err == nil {
				assert.Equal(t, CreatePostgreSQLPacket('R', []byte{0x0, 0x0, 0x0, 0x3}), response)
			} else {
				errs <- err
			}
		} else {
			errs <- errors.New("response is not a []byte") //nolint:goerr113
		}
		assert.Empty(t, paramsMap["error"])
		return params, nil
	}
	pluginRegistry.AddHook(plugin.OnTrafficFromServer, 1, onTrafficFromServer)

	onTrafficToClient := func(
		ctx context.Context,
		params *structpb.Struct,
		opts ...grpc.CallOption,
	) (*structpb.Struct, error) {
		paramsMap := params.AsMap()
		if paramsMap["response"] == nil {
			errs <- errors.New("response is nil") //nolint:goerr113
		}

		logger.Info().Msg("Egress traffic")
		if resp, ok := paramsMap["response"].(string); ok {
			if response, err := base64.StdEncoding.DecodeString(resp); err == nil {
				assert.Equal(t, CreatePostgreSQLPacket('R', []byte{0x0, 0x0, 0x0, 0x3}), response)
			} else {
				errs <- err
			}
		} else {
			errs <- errors.New("response is not a []byte") //nolint:goerr113
		}
		assert.Empty(t, paramsMap["error"])
		return params, nil
	}
	pluginRegistry.AddHook(plugin.OnTrafficToClient, 1, onTrafficToClient)

	clientConfig := config.Client{
		Network:            "tcp",
		Address:            "localhost:5432",
		ReceiveBufferSize:  config.DefaultBufferSize,
		ReceiveChunkSize:   config.DefaultChunkSize,
		ReceiveDeadline:    config.DefaultReceiveDeadline,
		SendDeadline:       config.DefaultSendDeadline,
		TCPKeepAlive:       false,
		TCPKeepAlivePeriod: config.DefaultTCPKeepAlivePeriod,
	}

	// Create a connection pool.
	pool := pool.NewPool(2)
	client1 := NewClient(&clientConfig, logger)
	err := pool.Put(client1.ID, client1)
	assert.Nil(t, err)
	client2 := NewClient(&clientConfig, logger)
	err = pool.Put(client2.ID, client2)
	assert.Nil(t, err)

	// Create a proxy with a fixed buffer pool.
	proxy := NewProxy(
		pool, pluginRegistry, false, false, config.DefaultHealthCheckPeriod, &clientConfig, logger)

	// Create a server.
	server := NewServer(
		"tcp",
		"127.0.0.1:15432",
		0,
		0,
		config.DefaultTickInterval,
		[]gnet.Option{
			gnet.WithMulticore(false),
			gnet.WithReuseAddr(true),
			gnet.WithReusePort(true),
		},
		proxy,
		logger,
		pluginRegistry,
	)
	assert.NotNil(t, server)

	go func(server *Server, errs chan error) {
		if err := server.Run(); err != nil {
			errs <- err
		}
		close(errs)
	}(server, errs)

	//nolint:thelper
	go func(t *testing.T, server *Server, proxy *Proxy, errs chan error) {
		for {
			if server.IsRunning() {
				client := NewClient(
					&config.Client{
						Network:            "tcp",
						Address:            "127.0.0.1:15432",
						ReceiveBufferSize:  config.DefaultBufferSize,
						ReceiveChunkSize:   config.DefaultChunkSize,
						ReceiveDeadline:    config.DefaultReceiveDeadline,
						SendDeadline:       config.DefaultSendDeadline,
						TCPKeepAlive:       false,
						TCPKeepAlivePeriod: config.DefaultTCPKeepAlivePeriod,
					},
					logger)

				assert.NotNil(t, client)
				sent, err := client.Send(CreatePgStartupPacket())
				assert.Nil(t, err)
				assert.Equal(t, len(CreatePgStartupPacket()), sent)

				// The server should respond with an 'R' packet.
				size, data, err := client.Receive()
				msg := []byte{0x0, 0x0, 0x0, 0x3}
				// This includes the message type, length and the message itself.
				assert.Equal(t, 9, size)
				assert.Equal(t, len(data[:size]), size)
				assert.Nil(t, err)
				packetSize := int(data[1])<<24 | int(data[2])<<16 | int(data[3])<<8 | int(data[4])
				assert.Equal(t, 8, packetSize)
				assert.NotEmpty(t, data[:size])
				assert.Equal(t, msg, data[5:size])
				// AuthenticationOk.
				assert.Equal(t, uint8(0x52), data[0])

				assert.Equal(t, 1, proxy.availableConnections.Size())
				assert.Equal(t, 1, proxy.busyConnections.Size())

				// Test Prometheus metrics.
				CollectAndComparePrometheusMetrics(t)

				// Clean up.
				server.Shutdown()
				client.Close()
				if pgErr := postgres.Stop(); pgErr != nil {
					errs <- err
				}
				break
			}
		}
	}(t, server, proxy, errs)

	for err := range errs {
		if err != nil {
			t.Fatal(err)
		}
	}
}
