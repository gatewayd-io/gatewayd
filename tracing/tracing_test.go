package tracing

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// Test_OTLPTracer tests the OTLPTracer function.
func Test_OTLPTracer(t *testing.T) {
	shutdown := OTLPTracer(false, "localhost:4317", "gatewayd")
	require.NotNil(t, shutdown)
}
