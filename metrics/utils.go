package metrics

import "net/http"

// HeaderBypassResponseWriter implements the http.ResponseWriter interface
// and allows us to bypass the response header when writing to the response.
// This is useful for merging metrics from multiple sources.
type HeaderBypassResponseWriter struct {
	http.ResponseWriter
}

// WriteHeader intentionally does nothing, but is required to
// implement the http.ResponseWriter.
func (w *HeaderBypassResponseWriter) WriteHeader(int) {}

// Write writes the data to the response.
func (w *HeaderBypassResponseWriter) Write(data []byte) (int, error) {
	return w.ResponseWriter.Write(data) //nolint:wrapcheck
}
