package network

import (
	"encoding/binary"
	"strings"
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/stretchr/testify/assert"
)

type WriteBuffer struct {
	Bytes []byte

	msgStart int
}

// writeStartupMsg writes a PostgreSQL startup message to the buffer.
func writeStartupMsg(buf *WriteBuffer, user, database, appName string) {
	// Write startup message header
	buf.msgStart = len(buf.Bytes)
	buf.Bytes = append(buf.Bytes, 0, 0, 0, 0)

	// Write protocol version
	buf.Bytes = append(buf.Bytes, 0, 0, 0, 0)
	binary.BigEndian.PutUint32(buf.Bytes[len(buf.Bytes)-4:], uint32(196608))

	buf.WriteString("user")
	buf.WriteString(user)
	buf.WriteString("database")
	buf.WriteString(database)
	buf.WriteString("application_name")
	buf.WriteString(appName)
	buf.WriteString("")

	// Write message length
	binary.BigEndian.PutUint32(
		buf.Bytes[buf.msgStart:], uint32(len(buf.Bytes)-buf.msgStart))
}

// WriteString writes a null-terminated string to the buffer.
func (buf *WriteBuffer) WriteString(s string) {
	buf.Bytes = append(buf.Bytes, s...)
	buf.Bytes = append(buf.Bytes, 0)
}

// CreatePostgreSQLPacket creates a PostgreSQL packet.
func CreatePostgreSQLPacket(typ byte, msg []byte) []byte {
	packet := make([]byte, 1+4+len(msg))

	packet[0] = typ
	binary.BigEndian.PutUint32(packet[1:], uint32(len(msg)+4))
	for i, b := range msg {
		packet[i+5] = b
	}

	return packet
}

// CreatePgStartupPacket creates a PostgreSQL startup packet.
func CreatePgStartupPacket() []byte {
	buf := &WriteBuffer{}
	writeStartupMsg(buf, "postgres", "postgres", "gatewayd")
	return buf.Bytes
}

func CollectAndComparePrometheusMetrics(t *testing.T) {
	t.Helper()

	const metadata = `
		# HELP gatewayd_bytes_received_from_client Number of bytes received from client
		# TYPE gatewayd_bytes_received_from_client summary
		# HELP gatewayd_bytes_received_from_server Number of bytes received from server
		# TYPE gatewayd_bytes_received_from_server summary
		# HELP gatewayd_bytes_sent_to_client Number of bytes sent to client
		# TYPE gatewayd_bytes_sent_to_client summary
		# HELP gatewayd_bytes_sent_to_server Number of bytes sent to server
		# TYPE gatewayd_bytes_sent_to_server summary
		# HELP gatewayd_client_connections Number of client connections
		# TYPE gatewayd_client_connections gauge
		# HELP gatewayd_plugin_hooks_executed_total Number of plugin hooks executed
		# TYPE gatewayd_plugin_hooks_executed_total counter
		# HELP gatewayd_plugin_hooks_registered_total Number of plugin hooks registered
		# TYPE gatewayd_plugin_hooks_registered_total counter
		# HELP gatewayd_plugins_loaded_total Number of plugins loaded
		# TYPE gatewayd_plugins_loaded_total counter
		# HELP gatewayd_proxied_connections Number of proxy connects
		# TYPE gatewayd_proxied_connections gauge
		# HELP gatewayd_proxy_health_checks_total Number of proxy health checks
		# TYPE gatewayd_proxy_health_checks_total counter
		# HELP gatewayd_proxy_passthrough_terminations_total Number of proxy passthrough terminations by plugins
		# TYPE gatewayd_proxy_passthrough_terminations_total counter
		# HELP gatewayd_proxy_passthroughs_total Number of successful proxy passthroughs
		# TYPE gatewayd_proxy_passthroughs_total counter
		# HELP gatewayd_server_connections Number of server connections
		# TYPE gatewayd_server_connections gauge
		# HELP gatewayd_server_ticks_fired_total Total number of server ticks fired
		# TYPE gatewayd_server_ticks_fired_total counter
		# HELP gatewayd_traffic_bytes Number of total bytes passed through GatewayD via client or server
		# TYPE gatewayd_traffic_bytes summary
		`

	var (
		want = metadata + `
			gatewayd_bytes_received_from_client_sum 67
			gatewayd_bytes_received_from_client_count 1
			gatewayd_bytes_received_from_server_sum 96
			gatewayd_bytes_received_from_server_count 4
			gatewayd_bytes_sent_to_client_sum 24
			gatewayd_bytes_sent_to_client_count 1
			gatewayd_bytes_sent_to_server_sum 282
			gatewayd_bytes_sent_to_server_count 5
			gatewayd_client_connections 1
			gatewayd_plugin_hooks_executed_total 11
			gatewayd_plugin_hooks_registered_total 0
			gatewayd_plugins_loaded_total 0
			gatewayd_proxied_connections 1
			gatewayd_proxy_health_checks_total 0
			gatewayd_proxy_passthrough_terminations_total 0
			gatewayd_proxy_passthroughs_total 1
			gatewayd_server_connections 5
			gatewayd_traffic_bytes_sum 182
			gatewayd_traffic_bytes_count 4
			gatewayd_server_ticks_fired_total 1
		`

		metrics = []string{
			"gatewayd_bytes_received_from_client",
			"gatewayd_bytes_received_from_server",
			"gatewayd_bytes_sent_to_client",
			"gatewayd_bytes_sent_to_server",
			"gatewayd_client_connections",
			"gatewayd_plugin_hooks_executed_total",
			"gatewayd_plugin_hooks_registered_total",
			"gatewayd_plugins_loaded_total",
			"gatewayd_proxied_connections",
			"gatewayd_proxy_health_checks_total",
			"gatewayd_proxy_passthrough_terminations_total",
			"gatewayd_proxy_passthroughs_total",
			"gatewayd_server_connections",
			"gatewayd_traffic_bytes",
			"gatewayd_server_ticks_fired_total",
		}
	)
	assert.NoError(t,
		testutil.GatherAndCompare(prometheus.DefaultGatherer, strings.NewReader(want), metrics...))
}
