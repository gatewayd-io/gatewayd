package metrics

import (
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_HeaderBypassResponseWriter(t *testing.T) {
	testServer := httptest.NewServer(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			respWriter := HeaderBypassResponseWriter{w}
			respWriter.WriteHeader(http.StatusBadGateway) // This is a no-op.
			respWriter.Write([]byte("Hello, World!"))
		}),
	)
	defer testServer.Close()

	resp, err := http.Get(testServer.URL)
	require.NoError(t, err)
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
