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
	go merger.Start()
	time.Sleep(1 * time.Second)

	want := `# HELP gatewayd_test_total Test counter
# TYPE gatewayd_test_total counter
gatewayd_test_total{plugin="test"} 1`

	assert.Contains(t, string(merger.OutputMetrics), want)
	defer merger.Stop()
}
