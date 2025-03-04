package api

import (
	"encoding/json"
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
	api := getAPIConfig(
		"localhost:18082",
		"localhost:19092",
	)
	healthchecker := &HealthChecker{Servers: api.Servers}
	grpcServer := NewGRPCServer(
		t.Context(), GRPCServer{API: api, HealthChecker: healthchecker})
	go grpcServer.Start()
	require.NotNil(t, grpcServer)

	httpServer := NewHTTPServer(api.Options)
	go httpServer.Start()
	require.NotNil(t, httpServer)

	time.Sleep(time.Second)

	// Check version via the gRPC server.
	req, err := http.NewRequestWithContext(
		t.Context(),
		http.MethodGet,
		"http://localhost:18082/v1/GatewayDPluginService/Version",
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
		t.Context(),
		http.MethodGet,
		"http://localhost:18082/healthz",
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
		t.Context(),
		http.MethodGet,
		"http://localhost:18082/version",
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

	grpcServer.Shutdown(nil) //nolint:staticcheck
	httpServer.Shutdown(t.Context())
}
