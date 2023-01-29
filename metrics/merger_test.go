package metrics

import (
	"testing"
	"time"

	"github.com/gatewayd-io/gatewayd/config"
	"github.com/gatewayd-io/gatewayd/logging"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
)

func TestMerger(t *testing.T) {
	// This runs inside a plugin and exposes metrics to GatewayD
	// via HTTP on a unix socket. But we don't need to test that here,
	// so we just expose the metrics via the same mechanism to the merger.
	go exposeMetrics(t)

	logger := logging.NewLogger(logging.LoggerConfig{
		Output:     config.Console,
		TimeFormat: zerolog.TimeFormatUnix,
		Level:      zerolog.InfoLevel,
		NoColor:    true,
	})

	merger := NewMerger(1, logger)
	merger.Add("test", "/tmp/test.sock")

	// We need to give the merger some time to read the metrics.
	// TODO: Find a better way to do this.
	// Note: httptest.NewServer only creates a TCP server.
	time.Sleep(100 * time.Millisecond)

	metrics, err := merger.ReadMetrics()
	assert.Nil(t, err)
	err = merger.MergeMetrics(metrics)
	assert.Nil(t, err)

	// We expect the metrics to be merged into a single output.
	// Also, we don't need to test the actual metrics from GatewayD,
	// so the output only contains the metrics from the test plugin.
	// All the plugin's metrics are prefixed with "gatewayd_" and labeled
	// with the plugin name, i.e. "test" during merging.
	want := `# HELP gatewayd_test_total Test counter
# TYPE gatewayd_test_total counter
gatewayd_test_total{plugin="test"} 1`

	assert.Contains(t, string(merger.OutputMetrics), want)
}
