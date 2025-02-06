package api

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"testing"
	"time"

	"github.com/gatewayd-io/gatewayd/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Test_HTTP_Server tests the HTTP to gRPC gateway.
func Test_HTTP_Server(t *testing.T) {
	api := getAPIConfig()
	healthchecker := &HealthChecker{Servers: api.Servers}
	grpcServer := NewGRPCServer(
		context.Background(), GRPCServer{API: api, HealthChecker: healthchecker})
	require.NotNil(t, grpcServer, "gRPC server should not be nil")

	httpServer := NewHTTPServer(api.Options)
	require.NotNil(t, httpServer, "HTTP server should not be nil")

	// Start gRPC server with error handling
	errChan := make(chan error, 1)
	go func(grpcServer *GRPCServer) {
		errChan <- func() error {
			defer func() {
				if r := recover(); r != nil {
					errChan <- fmt.Errorf("gRPC server panicked: %v", r)
				}
			}()
			grpcServer.Start()
			return nil
		}()
	}(grpcServer)

	go func(httpServer *HTTPServer) {
		httpServer.Start()
	}(httpServer)

	// Wait for potential startup errors
	select {
	case err := <-errChan:
		require.NoError(t, err, "gRPC server failed to start")
	case <-time.After(1 * time.Second): // Wait for the servers to start
	}

	// Check version via the gRPC server.
	req, err := http.NewRequestWithContext(
		context.Background(),
		http.MethodGet,
		"http://localhost:18080/v1/GatewayDPluginService/Version",
		nil,
	)
	require.NoError(t, err)
	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Equal(t, "application/json", resp.Header.Get("Content-Type"))
	var respBody map[string]any
	err = json.NewDecoder(resp.Body).Decode(&respBody)
	require.NoError(t, err)
	assert.Equal(t, config.Version, respBody["version"])
	assert.Equal(t, config.VersionInfo(), respBody["versionInfo"])

	// Check health via the gRPC gateway.
	req, err = http.NewRequestWithContext(
		context.Background(),
		http.MethodGet,
		"http://localhost:18080/healthz",
		nil,
	)
	require.NoError(t, err)
	resp, err = http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()
	assert.Equal(t, http.StatusServiceUnavailable, resp.StatusCode)
	assert.Equal(t, "application/json", resp.Header.Get("Content-Type"))
	err = json.NewDecoder(resp.Body).Decode(&respBody)
	require.NoError(t, err)
	assert.Equal(t, "NOT_SERVING", respBody["status"])

	// Check version via the gRPC gateway.
	req, err = http.NewRequestWithContext(
		context.Background(),
		http.MethodGet,
		"http://localhost:18080/version",
		nil,
	)
	require.NoError(t, err)
	resp, err = http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Equal(t, "text/plain; charset=utf-8", resp.Header.Get("Content-Type"))
	respBodyBytes, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	assert.Equal(t, len(config.Version), len(respBodyBytes))
	assert.Equal(t, config.Version, string(respBodyBytes))

	grpcServer.Shutdown(context.Background())
	httpServer.Shutdown(context.Background())
}
