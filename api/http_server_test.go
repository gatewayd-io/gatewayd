package api

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"testing"

	"github.com/gatewayd-io/gatewayd/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Test_HTTP_Server tests the HTTP to gRPC gateway.
func Test_HTTP_Server(t *testing.T) {
	api := getAPIConfig()
	healthchecker := &HealthChecker{Servers: api.Servers}
	grpcServer := NewGRPCServer(GRPCServer{API: api, HealthChecker: healthchecker})
	assert.NotNil(t, grpcServer)
	httpServer := NewHTTPServer(api.Options)
	assert.NotNil(t, httpServer)

	go func(grpcServer *GRPCServer) {
		grpcServer.Start()
	}(grpcServer)

	go func(httpServer *HTTPServer) {
		httpServer.Start()
	}(httpServer)

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
	var respBody map[string]interface{}
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
