package network

import (
	"context"
	"encoding/base64"
	"errors"
	"testing"

	v1 "github.com/gatewayd-io/gatewayd-plugin-sdk/plugin/v1"
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

	logger := logging.NewLogger(context.Background(), logging.LoggerConfig{
		Output:            []config.LogOutput{config.Console},
		TimeFormat:        zerolog.TimeFormatUnix,
		ConsoleTimeFormat: config.DefaultConsoleTimeFormat,
		Level:             zerolog.ErrorLevel,
		NoColor:           true,
	})

	pluginRegistry := plugin.NewRegistry(
		context.Background(),
		plugin.Registry{
			Compatibility: config.Loose,
			Verification:  config.PassDown,
			Acceptance:    config.Accept,
			Termination:   config.Stop,
			Logger:        logger,
		})

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
	pluginRegistry.AddHook(v1.HookName_HOOK_NAME_ON_TRAFFIC_FROM_CLIENT, 1, onTrafficFromClient)

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
	pluginRegistry.AddHook(v1.HookName_HOOK_NAME_ON_TRAFFIC_TO_SERVER, 1, onTrafficToServer)

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
				assert.Equal(t, CreatePostgreSQLPacket('R', []byte{
					0x0, 0x0, 0x0, 0xa, 0x53, 0x43, 0x52, 0x41, 0x4d, 0x2d, 0x53, 0x48, 0x41, 0x2d, 0x32, 0x35, 0x36, 0x0, 0x0,
				}), response)
			} else {
				errs <- err
			}
		} else {
			errs <- errors.New("response is not a []byte") //nolint:goerr113
		}
		assert.Empty(t, paramsMap["error"])
		return params, nil
	}
	pluginRegistry.AddHook(v1.HookName_HOOK_NAME_ON_TRAFFIC_FROM_SERVER, 1, onTrafficFromServer)

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
				assert.Equal(t, uint8(0x52), response[0])
			} else {
				errs <- err
			}
		} else {
			errs <- errors.New("response is not a []byte") //nolint:goerr113
		}
		assert.Empty(t, paramsMap["error"])
		return params, nil
	}
	pluginRegistry.AddHook(v1.HookName_HOOK_NAME_ON_TRAFFIC_TO_CLIENT, 1, onTrafficToClient)

	client := Client{
		Network:            "tcp",
		Address:            "localhost:5432",
		ReceiveChunkSize:   config.DefaultChunkSize,
		ReceiveDeadline:    config.DefaultReceiveDeadline,
		SendDeadline:       config.DefaultSendDeadline,
		TCPKeepAlive:       false,
		TCPKeepAlivePeriod: config.DefaultTCPKeepAlivePeriod,
		Logger:             logger,
	}

	// Create a connection pool.
	pool := pool.NewPool(context.Background(), 3)
	client1 := NewClient(context.Background(), &client)
	err := pool.Put(client1.ID, client1)
	assert.Nil(t, err)
	client2 := NewClient(context.Background(), &client)
	err = pool.Put(client2.ID, client2)
	assert.Nil(t, err)
	client3 := NewClient(context.Background(), &client)
	err = pool.Put(client3.ID, client3)
	assert.Nil(t, err)

	// Create a Proxy with a fixed buffer pool.
	proxy := NewProxy(
		context.Background(),
		Proxy{
			AvailableConnections: pool,
			PluginRegistry:       pluginRegistry,
			HealthCheckPeriod:    config.DefaultHealthCheckPeriod,
			Client:               &client,
			Logger:               logger,
			PluginTimeout:        config.DefaultPluginTimeout,
		},
	)

	// Create a server.
	server := NewServer(
		context.Background(),
		Server{
			Network:      "tcp",
			Address:      "127.0.0.1:15432",
			SoftLimit:    0,
			HardLimit:    0,
			TickInterval: config.DefaultTickInterval,
			Options: []gnet.Option{
				gnet.WithMulticore(false),
				gnet.WithReuseAddr(true),
				gnet.WithReusePort(true),
			},
			Proxy:          proxy,
			Logger:         logger,
			PluginRegistry: pluginRegistry,
			PluginTimeout:  config.DefaultPluginTimeout,
		})
	assert.NotNil(t, server)

	go func(server *Server, errs chan error) {
		if err := server.Run(); err != nil {
			errs <- err
		}
		close(errs)
	}(server, errs)

	//nolint:thelper
	go func(t *testing.T, server *Server, proxy *Proxy) {
		for {
			if server.IsRunning() {
				client := NewClient(
					context.Background(),
					&Client{
						Network:            "tcp",
						Address:            "127.0.0.1:15432",
						ReceiveChunkSize:   config.DefaultChunkSize,
						ReceiveDeadline:    config.DefaultReceiveDeadline,
						SendDeadline:       config.DefaultSendDeadline,
						TCPKeepAlive:       false,
						TCPKeepAlivePeriod: config.DefaultTCPKeepAlivePeriod,
						Logger:             logger,
					})

				assert.NotNil(t, client)
				sent, err := client.Send(CreatePgStartupPacket())
				assert.Nil(t, err)
				assert.Equal(t, len(CreatePgStartupPacket()), sent)

				// The server should respond with an 'R' packet.
				size, data, err := client.Receive()
				msg := []byte{
					0x0, 0x0, 0x0, 0xa, 0x53, 0x43, 0x52, 0x41, 0x4d, 0x2d,
					0x53, 0x48, 0x41, 0x2d, 0x32, 0x35, 0x36, 0x0, 0x0,
				}
				// This includes the message type, length and the message itself.
				assert.Equal(t, 24, size)
				assert.Equal(t, len(data[:size]), size)
				assert.Nil(t, err)
				packetSize := int(data[1])<<24 | int(data[2])<<16 | int(data[3])<<8 | int(data[4])
				assert.Equal(t, 23, packetSize)
				assert.NotEmpty(t, data[:size])
				assert.Equal(t, msg, data[5:size])
				// AuthenticationOk.
				assert.Equal(t, uint8(0x52), data[0])

				assert.Equal(t, 2, proxy.AvailableConnections.Size())
				assert.Equal(t, 1, proxy.busyConnections.Size())

				// Test Prometheus metrics.
				CollectAndComparePrometheusMetrics(t)

				// Clean up.
				server.Shutdown()
				client.Close()
				break
			}
		}
	}(t, server, proxy)

	for err := range errs {
		if err != nil {
			t.Fatal(err)
		}
	}
}
