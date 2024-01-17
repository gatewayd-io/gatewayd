package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

const (
	Namespace = "gatewayd"
)

var (
	ClientConnections = promauto.NewGauge(prometheus.GaugeOpts{
		Namespace: Namespace,
		Name:      "client_connections",
		Help:      "Number of client connections",
	})
	ServerConnections = promauto.NewGauge(prometheus.GaugeOpts{
		Namespace: Namespace,
		Name:      "server_connections",
		Help:      "Number of server connections",
	})
	TLSConnections = promauto.NewGauge(prometheus.GaugeOpts{
		Namespace: Namespace,
		Name:      "tls_connections",
		Help:      "Number of TLS connections",
	})
	ServerTicksFired = promauto.NewCounter(prometheus.CounterOpts{
		Namespace: Namespace,
		Name:      "server_ticks_fired_total",
		Help:      "Total number of server ticks fired",
	})
	BytesReceivedFromClient = promauto.NewSummary(prometheus.SummaryOpts{
		Namespace: Namespace,
		Name:      "bytes_received_from_client",
		Help:      "Number of bytes received from client",
	})
	BytesSentToServer = promauto.NewSummary(prometheus.SummaryOpts{
		Namespace: Namespace,
		Name:      "bytes_sent_to_server",
		Help:      "Number of bytes sent to server",
	})
	BytesReceivedFromServer = promauto.NewSummary(prometheus.SummaryOpts{
		Namespace: Namespace,
		Name:      "bytes_received_from_server",
		Help:      "Number of bytes received from server",
	})
	BytesSentToClient = promauto.NewSummary(prometheus.SummaryOpts{
		Namespace: Namespace,
		Name:      "bytes_sent_to_client",
		Help:      "Number of bytes sent to client",
	})
	TotalTrafficBytes = promauto.NewSummary(prometheus.SummaryOpts{
		Namespace: Namespace,
		Name:      "traffic_bytes",
		Help:      "Number of total bytes passed through GatewayD via client or server",
	})
	PluginsLoaded = promauto.NewCounter(prometheus.CounterOpts{
		Namespace: Namespace,
		Name:      "plugins_loaded_total",
		Help:      "Number of plugins loaded",
	})
	PluginHooksRegistered = promauto.NewCounter(prometheus.CounterOpts{
		Namespace: Namespace,
		Name:      "plugin_hooks_registered_total",
		Help:      "Number of plugin hooks registered",
	})
	PluginHooksExecuted = promauto.NewCounter(prometheus.CounterOpts{
		Namespace: Namespace,
		Name:      "plugin_hooks_executed_total",
		Help:      "Number of plugin hooks executed",
	})
	ProxyHealthChecks = promauto.NewCounter(prometheus.CounterOpts{
		Namespace: Namespace,
		Name:      "proxy_health_checks_total",
		Help:      "Number of proxy health checks",
	})
	ProxiedConnections = promauto.NewGauge(prometheus.GaugeOpts{
		Namespace: Namespace,
		Name:      "proxied_connections",
		Help:      "Number of proxy connects",
	})
	ProxyPassThroughsToClient = promauto.NewCounter(prometheus.CounterOpts{
		Namespace: Namespace,
		Name:      "proxy_passthroughs_to_client_total",
		Help:      "Number of successful proxy passthroughs from server to client",
	})
	ProxyPassThroughsToServer = promauto.NewCounter(prometheus.CounterOpts{
		Namespace: Namespace,
		Name:      "proxy_passthroughs_to_server_total",
		Help:      "Number of successful proxy passthroughs from client to server",
	})
	ProxyPassThroughTerminations = promauto.NewCounter(prometheus.CounterOpts{
		Namespace: Namespace,
		Name:      "proxy_passthrough_terminations_total",
		Help:      "Number of proxy passthrough terminations by plugins",
	})
	APIRequests = promauto.NewCounterVec(prometheus.CounterOpts{
		Namespace: Namespace,
		Name:      "api_requests_total",
		Help:      "Number of API requests",
	}, []string{"method", "endpoint"})
	APIRequestsErrors = promauto.NewCounterVec(prometheus.CounterOpts{
		Namespace: Namespace,
		Name:      "api_requests_errors_total",
		Help:      "Number of API request errors",
	}, []string{"method", "endpoint", "error"})
)
