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
	"github.com/gatewayd-io/gatewayd/act"
	"github.com/gatewayd-io/gatewayd/config"
	"github.com/gatewayd-io/gatewayd/logging"
	"github.com/gatewayd-io/gatewayd/plugin"
	"github.com/gatewayd-io/gatewayd/testhelpers"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
)

// TestRunServer tests an entire server run with a single client connection and hooks.
func TestRunServer(t *testing.T) {
	ctx := context.Background()

	// Reset Prometheus metrics.
	prometheus.DefaultRegisterer = prometheus.NewRegistry()

	logger := logging.NewLogger(ctx, logging.LoggerConfig{
		Output: []config.LogOutput{
			config.File,
		},
		TimeFormat:        zerolog.TimeFormatUnix,
		ConsoleTimeFormat: time.RFC3339,
		Level:             zerolog.DebugLevel,
		NoColor:           true,
		FileName:          "server_test.log",
	})

	actRegistry := act.NewActRegistry(
		act.Registry{
			Signals:              act.BuiltinSignals(),
			Policies:             act.BuiltinPolicies(),
			Actions:              act.BuiltinActions(),
			DefaultPolicyName:    config.DefaultPolicy,
			PolicyTimeout:        config.DefaultPolicyTimeout,
			DefaultActionTimeout: config.DefaultActionTimeout,
			Logger:               logger,
		})
	pluginRegistry := plugin.NewRegistry(
		ctx,
		plugin.Registry{
			ActRegistry:   actRegistry,
			Compatibility: config.Loose,
			Logger:        logger,
		})

	pluginRegistry.AddHook(v1.HookName_HOOK_NAME_ON_TRAFFIC_FROM_CLIENT, 1, onIncomingTraffic)
	pluginRegistry.AddHook(v1.HookName_HOOK_NAME_ON_TRAFFIC_TO_SERVER, 1, onIncomingTraffic)
	pluginRegistry.AddHook(v1.HookName_HOOK_NAME_ON_TRAFFIC_FROM_SERVER, 1, onOutgoingTraffic)
	pluginRegistry.AddHook(v1.HookName_HOOK_NAME_ON_TRAFFIC_TO_CLIENT, 1, onOutgoingTraffic)

	assert.NotNil(t, pluginRegistry.ActRegistry)
	assert.NotNil(t, pluginRegistry.ActRegistry.Signals)
	assert.NotNil(t, pluginRegistry.ActRegistry.Policies)
	assert.NotNil(t, pluginRegistry.ActRegistry.Actions)
	assert.Equal(t, config.DefaultPolicy, pluginRegistry.ActRegistry.DefaultPolicy.Name)
	assert.Equal(t, config.DefaultPolicy, pluginRegistry.ActRegistry.DefaultSignal.Name)

	// Start the test containers.
	postgresHostIP1, postgresMappedPort1 := testhelpers.SetupPostgreSQLTestContainer(ctx, t)
	postgresHostIP2, postgresMappedPort2 := testhelpers.SetupPostgreSQLTestContainer(ctx, t)

	proxy1 := setupProxy(ctx, t, postgresHostIP1, postgresMappedPort1.Port(), logger, pluginRegistry)
	proxy2 := setupProxy(ctx, t, postgresHostIP2, postgresMappedPort2.Port(), logger, pluginRegistry)

	// Create a server.
	server := NewServer(
		ctx,
		Server{
			Network:      "tcp",
			Address:      "127.0.0.1:15432",
			TickInterval: config.DefaultTickInterval,
			Options: Option{
				EnableTicker: true,
			},
			Proxies:                  []IProxy{proxy1, proxy2},
			Logger:                   logger,
			PluginRegistry:           pluginRegistry,
			PluginTimeout:            config.DefaultPluginTimeout,
			HandshakeTimeout:         config.DefaultHandshakeTimeout,
			LoadbalancerStrategyName: config.RoundRobinStrategy,
		},
	)
	assert.NotNil(t, server)
	assert.Zero(t, server.connections)
	assert.Zero(t, server.CountConnections())
	assert.Empty(t, server.host)
	assert.Empty(t, server.port)
	assert.False(t, server.running.Load())

	var waitGroup sync.WaitGroup
	waitGroup.Add(2)

	go func(t *testing.T, server *Server) {
		t.Helper()

		if err := server.Run(); err != nil {
			t.Errorf("server.Run() error = %v", err)
		}
	}(t, server)

	testProxy := func(
		t *testing.T,
		proxy *Proxy,
		waitGroup *sync.WaitGroup,
	) {
		t.Helper()

		defer waitGroup.Done()
		<-time.After(500 * time.Millisecond)

		client := NewClient(
			ctx,
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

		assert.Equal(t, 2, proxy.AvailableConnections.Size())
		assert.Equal(t, 1, proxy.busyConnections.Size())

		// Terminate the connection.
		sent, err = client.Send(CreatePgTerminatePacket())
		assert.Nil(t, err)
		assert.Len(t, CreatePgTerminatePacket(), sent)

		// Close the connection.
		client.Close()

		<-time.After(100 * time.Millisecond)
	}

	// Test both proxies.
	// Based on the default Loadbalancer strategy (RoundRobin), the first client request will be sent to proxy2,
	// followed by proxy1 for the next request.
	go testProxy(t, proxy2, &waitGroup)
	go testProxy(t, proxy1, &waitGroup)

	// Wait for all goroutines.
	waitGroup.Wait()

	// Shutdown the server and plugin registry.
	server.Shutdown()
	pluginRegistry.Shutdown()

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

	// Check if the log contains expected messages.
	logLines := string(buffer)
	expectedLogMessages := []string{
		"GatewayD is running",
		"GatewayD is opening a connection",
		"Client has been assigned",
		"Received data from client",
		"Sent data to database",
		"Received data from database",
		"Sent data to client",
		"GatewayD is closing a connection",
		"TLS is disabled",
		"GatewayD is shutting down",
		"All available connections have been closed",
		"All busy connections have been closed",
		"Server stopped",
	}

	for _, msg := range expectedLogMessages {
		assert.Contains(t, logLines, msg)
	}
	require.NoError(t, os.Remove("server_test.log"))

	// Test Prometheus metrics.
	// FIXME: Metric tests are flaky.
	// CollectAndComparePrometheusMetrics(t)
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
