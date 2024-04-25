package api

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"

	"github.com/gatewayd-io/gatewayd/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

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

	req, err := http.NewRequestWithContext(
		context.Background(),
		"GET",
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

	grpcServer.Shutdown(context.Background())
	httpServer.Shutdown(context.Background())
}
