package metrics

import (
	"net"
	"net/http"
	"os"
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

func exposeMetrics(t *testing.T) {
	t.Helper()

	testCounter := promauto.NewCounter(prometheus.CounterOpts{
		Name:      "test_total",
		Help:      "Test counter",
		Namespace: "gatewayd",
	})
	testCounter.Inc()

	if file, err := os.Stat("/tmp/test.sock"); err == nil && !file.IsDir() && file.Mode().Type() == os.ModeSocket {
		if err := os.Remove("/tmp/test.sock"); err != nil {
			t.Error("Failed to remove unix domain socket")
		}
	}

	listener, err := net.Listen("unix", "/tmp/test.sock")
	if err != nil {
		t.Error("Failed to start metrics server")
	}

	if err := http.Serve(listener, promhttp.Handler()); err != nil {
		t.Error("Failed to start metrics server")
	}
}
