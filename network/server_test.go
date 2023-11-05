package network

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"io"
	"os"
	"sync"
	"testing"
	"time"

	v1 "github.com/gatewayd-io/gatewayd-plugin-sdk/plugin/v1"
	"github.com/gatewayd-io/gatewayd/config"
	"github.com/gatewayd-io/gatewayd/logging"
	"github.com/gatewayd-io/gatewayd/plugin"
	"github.com/gatewayd-io/gatewayd/pool"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
)

// TestRunServer tests an entire server run with a single client connection and hooks.
func TestRunServer(t *testing.T) {
	errs := make(chan error)

	// Reset prometheus metrics.
	prometheus.DefaultRegisterer = prometheus.NewRegistry()

	logger := logging.NewLogger(context.Background(), logging.LoggerConfig{
		Output: []config.LogOutput{
			config.File,
		},
		TimeFormat:        zerolog.TimeFormatUnix,
		ConsoleTimeFormat: time.RFC3339,
		Level:             zerolog.DebugLevel,
		NoColor:           true,
		FileName:          "server_test.log",
	})

	pluginRegistry := plugin.NewRegistry(
		context.Background(),
		config.Loose,
		config.PassDown,
		config.Accept,
		config.Stop,
		logger,
		false,
	)

	onTrafficFromClient := func(
		ctx context.Context,
		params *v1.Struct,
		opts ...grpc.CallOption,
	) (*v1.Struct, error) {
		paramsMap := params.AsMap()
		if paramsMap["request"] == nil {
			errs <- errors.New("request is nil") //nolint:goerr113
		}

		if req, ok := paramsMap["request"].([]byte); ok {
			if !bytes.Equal(req, CreatePgStartupPacket()) {
				errs <- errors.New("request does not match") //nolint:goerr113
			}
		} else {
			errs <- errors.New("request is not a []byte") //nolint:goerr113
		}
		assert.Empty(t, paramsMap["error"], "The error MUST be empty.")

		return params, nil
	}
	pluginRegistry.AddHook(v1.HookName_HOOK_NAME_ON_TRAFFIC_FROM_CLIENT, 1, onTrafficFromClient)

	onTrafficToServer := func(
		ctx context.Context,
		params *v1.Struct,
		opts ...grpc.CallOption,
	) (*v1.Struct, error) {
		paramsMap := params.AsMap()
		if paramsMap["request"] == nil {
			errs <- errors.New("request is nil") //nolint:goerr113
		}

		logger.Info().Msg("Ingress traffic")
		if req, ok := paramsMap["request"].([]byte); ok {
			if !bytes.Equal(req, CreatePgStartupPacket()) {
				errs <- errors.New("request does not match") //nolint:goerr113
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
		params *v1.Struct,
		opts ...grpc.CallOption,
	) (*v1.Struct, error) {
		paramsMap := params.AsMap()
		if paramsMap["response"] == nil {
			errs <- errors.New("response is nil") //nolint:goerr113
		}

		logger.Info().Msg("Egress traffic")
		if resp, ok := paramsMap["response"].([]byte); ok {
			assert.Equal(t, CreatePostgreSQLPacket('R', []byte{
				0x0, 0x0, 0x0, 0xa, 0x53, 0x43, 0x52, 0x41, 0x4d, 0x2d, 0x53, 0x48, 0x41, 0x2d, 0x32, 0x35, 0x36, 0x0, 0x0,
			}), resp)
		} else {
			errs <- errors.New("response is not a []byte") //nolint:goerr113
		}
		assert.Empty(t, paramsMap["error"])
		return params, nil
	}
	pluginRegistry.AddHook(v1.HookName_HOOK_NAME_ON_TRAFFIC_FROM_SERVER, 1, onTrafficFromServer)

	onTrafficToClient := func(
		ctx context.Context,
		params *v1.Struct,
		opts ...grpc.CallOption,
	) (*v1.Struct, error) {
		paramsMap := params.AsMap()
		if paramsMap["response"] == nil {
			errs <- errors.New("response is nil") //nolint:goerr113
		}

		logger.Info().Msg("Egress traffic")
		if resp, ok := paramsMap["response"].([]byte); ok {
			assert.Equal(t, uint8(0x52), resp[0])
		} else {
			errs <- errors.New("response is not a []byte") //nolint:goerr113
		}
		assert.Empty(t, paramsMap["error"])
		return params, nil
	}
	pluginRegistry.AddHook(v1.HookName_HOOK_NAME_ON_TRAFFIC_TO_CLIENT, 1, onTrafficToClient)

	clientConfig := config.Client{
		Network:            "tcp",
		Address:            "localhost:5432",
		ReceiveChunkSize:   config.DefaultChunkSize,
		ReceiveDeadline:    config.DefaultReceiveDeadline,
		SendDeadline:       config.DefaultSendDeadline,
		TCPKeepAlive:       false,
		TCPKeepAlivePeriod: config.DefaultTCPKeepAlivePeriod,
	}

	// Create a connection newPool.
	newPool := pool.NewPool(context.Background(), 3)
	client1 := NewClient(context.Background(), &clientConfig, logger)
	err := newPool.Put(client1.ID, client1)
	assert.Nil(t, err)
	client2 := NewClient(context.Background(), &clientConfig, logger)
	err = newPool.Put(client2.ID, client2)
	assert.Nil(t, err)
	client3 := NewClient(context.Background(), &clientConfig, logger)
	err = newPool.Put(client3.ID, client3)
	assert.Nil(t, err)

	// Create a proxy with a fixed buffer newPool.
	proxy := NewProxy(
		context.Background(),
		newPool,
		pluginRegistry,
		false,
		false,
		config.DefaultHealthCheckPeriod,
		&clientConfig,
		logger,
		config.DefaultPluginTimeout)

	// Create a server.
	server := NewServer(
		context.Background(),
		"tcp",
		"127.0.0.1:15432",
		config.DefaultTickInterval,
		Option{
			EnableTicker: true,
		},
		proxy,
		logger,
		pluginRegistry,
		config.DefaultPluginTimeout,
		false,
		"",
		"",
		config.DefaultHandshakeTimeout,
	)
	assert.NotNil(t, server)

	stop := make(chan struct{})

	var waitGroup sync.WaitGroup

	waitGroup.Add(1)
	go func(t *testing.T, server *Server, pluginRegistry *plugin.Registry, stop chan struct{}, waitGroup *sync.WaitGroup) {
		t.Helper()
		for {
			select {
			case <-stop:
				server.Shutdown()
				pluginRegistry.Shutdown()

				// Wait for the server to stop.
				time.Sleep(100 * time.Millisecond)

				// Read the log file and check if the log file contains the expected log messages.
				if _, err := os.Stat("server_test.log"); err == nil {
					logFile, err := os.Open("server_test.log")
					assert.Nil(t, err)

					reader := bufio.NewReader(logFile)
					assert.NotNil(t, reader)

					buffer, err := io.ReadAll(reader)
					assert.Nil(t, err)
					assert.NotEmpty(t, buffer) // The log file should not be empty.
					require.NoError(t, logFile.Close())

					logLines := string(buffer)
					assert.Contains(t, logLines, "GatewayD is running", "GatewayD should be running")
					assert.Contains(t, logLines, "GatewayD is ticking...", "GatewayD should be ticking")
					assert.Contains(t, logLines, "Ingress traffic", "Ingress traffic should be logged")
					assert.Contains(t, logLines, "Egress traffic", "Egress traffic should be logged")
					assert.Contains(t, logLines, "GatewayD is shutting down", "GatewayD should be shutting down")

					require.NoError(t, os.Remove("server_test.log"))
				}
				waitGroup.Done()
				return
			case <-errs:
				server.Shutdown()
				pluginRegistry.Shutdown()
				waitGroup.Done()
				return
			default: //nolint:staticcheck
			}
		}
	}(t, server, pluginRegistry, stop, &waitGroup)

	waitGroup.Add(1)
	go func(t *testing.T, server *Server, errs chan error, waitGroup *sync.WaitGroup) {
		t.Helper()
		if err := server.Run(); err != nil {
			errs <- err
			t.Fail()
		}
		waitGroup.Done()
	}(t, server, errs, &waitGroup)

	waitGroup.Add(1)
	go func(t *testing.T, server *Server, proxy *Proxy, stop chan struct{}, waitGroup *sync.WaitGroup) {
		t.Helper()
		// Pause for a while to allow the server to start.
		time.Sleep(500 * time.Millisecond)

		for {
			if server.IsRunning() {
				client := NewClient(
					context.Background(),
					&config.Client{
						Network:            "tcp",
						Address:            "127.0.0.1:15432",
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
				assert.Len(t, CreatePgStartupPacket(), sent)

				// The server should respond with an 'R' packet.
				size, data, err := client.Receive()
				msg := []byte{
					0x0, 0x0, 0x0, 0xa, 0x53, 0x43, 0x52, 0x41, 0x4d, 0x2d,
					0x53, 0x48, 0x41, 0x2d, 0x32, 0x35, 0x36, 0x0, 0x0,
				}
				// This includes the message type, length and the message itself.
				assert.Equal(t, 24, size)
				assert.Len(t, data[:size], size)
				assert.Nil(t, err)
				packetSize := int(data[1])<<24 | int(data[2])<<16 | int(data[3])<<8 | int(data[4])
				assert.Equal(t, 23, packetSize)
				assert.NotEmpty(t, data[:size])
				assert.Equal(t, msg, data[5:size])
				// AuthenticationOk.
				assert.Equal(t, uint8(0x52), data[0])

				assert.Equal(t, 2, proxy.availableConnections.Size())
				assert.Equal(t, 1, proxy.busyConnections.Size())

				// Test Prometheus metrics.
				CollectAndComparePrometheusMetrics(t)

				client.Close()
				break
			}
			time.Sleep(100 * time.Millisecond)
		}
		stop <- struct{}{}
		close(stop)
		waitGroup.Done()
	}(t, server, proxy, stop, &waitGroup)

	waitGroup.Wait()
}
