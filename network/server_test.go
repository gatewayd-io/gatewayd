package network

import (
	"bufio"
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

	pluginRegistry.AddHook(v1.HookName_HOOK_NAME_ON_TRAFFIC_FROM_CLIENT, 1, onIncomingTraffic)
	pluginRegistry.AddHook(v1.HookName_HOOK_NAME_ON_TRAFFIC_TO_SERVER, 1, onIncomingTraffic)
	pluginRegistry.AddHook(v1.HookName_HOOK_NAME_ON_TRAFFIC_FROM_SERVER, 1, onOutgoingTraffic)
	pluginRegistry.AddHook(v1.HookName_HOOK_NAME_ON_TRAFFIC_TO_CLIENT, 1, onOutgoingTraffic)

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
	client1 := NewClient(context.Background(), &clientConfig, logger, nil)
	err := newPool.Put(client1.ID, client1)
	assert.Nil(t, err)
	client2 := NewClient(context.Background(), &clientConfig, logger, nil)
	err = newPool.Put(client2.ID, client2)
	assert.Nil(t, err)
	client3 := NewClient(context.Background(), &clientConfig, logger, nil)
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
	assert.Zero(t, server.connections)
	assert.Zero(t, server.CountConnections())
	assert.Empty(t, server.host)
	assert.Empty(t, server.port)
	assert.False(t, server.running.Load())

	var waitGroup sync.WaitGroup
	waitGroup.Add(2)

	go func(t *testing.T, server *Server, waitGroup *sync.WaitGroup) {
		t.Helper()

		if err := server.Run(); err != nil {
			t.Errorf("server.Run() error = %v", err)
		}

		waitGroup.Done()
	}(t, server, &waitGroup)

	go func(t *testing.T, server *Server, pluginRegistry *plugin.Registry, proxy *Proxy, waitGroup *sync.WaitGroup) {
		t.Helper()

		defer waitGroup.Done()
		<-time.After(500 * time.Millisecond)

		client := NewClient(
			context.Background(),
			&config.Client{
				Network:            "tcp",
				Address:            "127.0.0.1:15432",
				ReceiveChunkSize:   config.DefaultChunkSize,
				ReceiveDeadline:    config.DefaultReceiveDeadline,
				SendDeadline:       config.DefaultSendDeadline,
				DialTimeout:        config.DefaultDialTimeout,
				TCPKeepAlive:       false,
				TCPKeepAlivePeriod: config.DefaultTCPKeepAlivePeriod,
			},
			logger,
			nil)

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
		t.Log("data", data)
		t.Log("size", size)
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

		// Terminate the connection.
		sent, err = client.Send(CreatePgTerminatePacket())
		assert.Nil(t, err)
		assert.Len(t, CreatePgTerminatePacket(), sent)

		// Close the connection.
		client.Close()

		<-time.After(100 * time.Millisecond)

		if server != nil {
			server.Shutdown()
		}

		if pluginRegistry != nil {
			pluginRegistry.Shutdown()
		}

		// Wait for the server to stop.
		<-time.After(100 * time.Millisecond)

		// check server status and connections
		assert.False(t, server.running.Load())
		assert.Zero(t, server.connections)

		// Read the log file and check if the log file contains the expected log messages.
		require.FileExists(t, "server_test.log")
		logFile, origErr := os.Open("server_test.log")
		assert.Nil(t, origErr)

		reader := bufio.NewReader(logFile)
		assert.NotNil(t, reader)

		buffer, origErr := io.ReadAll(reader)
		assert.Nil(t, origErr)
		assert.NotEmpty(t, buffer) // The log file should not be empty.
		require.NoError(t, logFile.Close())

		logLines := string(buffer)
		assert.Contains(t, logLines, "GatewayD is running")
		assert.Contains(t, logLines, "GatewayD is opening a connection")
		assert.Contains(t, logLines, "Client has been assigned")
		assert.Contains(t, logLines, "Received data from client")
		assert.Contains(t, logLines, "Sent data to database")
		assert.Contains(t, logLines, "Received data from database")
		assert.Contains(t, logLines, "Sent data to client")
		assert.Contains(t, logLines, "GatewayD is closing a connection")
		assert.Contains(t, logLines, "TLS is disabled")
		assert.Contains(t, logLines, "GatewayD is shutting down")
		assert.Contains(t, logLines, "All available connections have been closed")
		assert.Contains(t, logLines, "All busy connections have been closed")
		assert.Contains(t, logLines, "Server stopped")

		require.NoError(t, os.Remove("server_test.log"))

		// Test Prometheus metrics.
		// FIXME: Metric tests are flaky.
		// CollectAndComparePrometheusMetrics(t)
	}(t, server, pluginRegistry, proxy, &waitGroup)

	waitGroup.Wait()
}

func onIncomingTraffic(
	_ context.Context,
	params *v1.Struct,
	_ ...grpc.CallOption,
) (*v1.Struct, error) {
	paramsMap := params.AsMap()
	if paramsMap["request"] == nil {
		return nil, errors.New("request is nil") //nolint:goerr113
	}

	if _, ok := paramsMap["request"].([]byte); !ok {
		return nil, errors.New("request is not a []byte") //nolint:goerr113
	}

	return params, nil
}

func onOutgoingTraffic(
	_ context.Context,
	params *v1.Struct,
	_ ...grpc.CallOption,
) (*v1.Struct, error) {
	paramsMap := params.AsMap()
	if paramsMap["response"] == nil {
		return nil, errors.New("response is nil") //nolint:goerr113
	}

	if _, ok := paramsMap["response"].([]byte); !ok {
		return nil, errors.New("response is not a []byte") //nolint:goerr113
	}

	return params, nil
}
