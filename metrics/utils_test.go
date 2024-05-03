package metrics

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_HeaderBypassResponseWriter(t *testing.T) {
	testServer := httptest.NewServer(
		http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
			respWriter := HeaderBypassResponseWriter{writer}
			respWriter.WriteHeader(http.StatusBadGateway) // This is a no-op.
			sent, err := respWriter.Write([]byte("Hello, World!"))
			require.NoError(t, err)
			assert.Equal(t, 13, sent)
		}),
	)
	defer testServer.Close()

	req, err := http.NewRequestWithContext(
		context.Background(), http.MethodGet, testServer.URL, nil)
	require.NoError(t, err)
	assert.NotNil(t, req)
	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	assert.NotNil(t, resp)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	// The WriteHeader method intentionally does nothing, to prevent a bug
	// in the merging metrics that causes the headers to be written twice,
	// which results in an error: "http: superfluous response.WriteHeader call".
	assert.NotEqual(t, http.StatusBadGateway, resp.StatusCode)
	greeting, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	resp.Body.Close()
	assert.Equal(t, "Hello, World!", string(greeting))
}
