package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

const (
	Namespace = "gatewayd"
)

var (
	ClientConnections = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: Namespace,
		Name:      "client_connections",
		Help:      "Number of client connections",
	}, []string{"group", "block"})
	ServerConnections = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: Namespace,
		Name:      "server_connections",
		Help:      "Number of server connections",
	}, []string{"group", "block"})
	TLSConnections = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: Namespace,
		Name:      "tls_connections",
		Help:      "Number of TLS connections",
	}, []string{"group", "block"})
	ServerTicksFired = promauto.NewCounter(prometheus.CounterOpts{
		Namespace: Namespace,
		Name:      "server_ticks_fired_total",
		Help:      "Total number of server ticks fired",
	})
	BytesReceivedFromClient = promauto.NewSummaryVec(prometheus.SummaryOpts{
		Namespace: Namespace,
		Name:      "bytes_received_from_client",
		Help:      "Number of bytes received from client",
	}, []string{"group", "block"})
	BytesSentToServer = promauto.NewSummaryVec(prometheus.SummaryOpts{
		Namespace: Namespace,
		Name:      "bytes_sent_to_server",
		Help:      "Number of bytes sent to server",
	}, []string{"group", "block"})
	BytesReceivedFromServer = promauto.NewSummaryVec(prometheus.SummaryOpts{
		Namespace: Namespace,
		Name:      "bytes_received_from_server",
		Help:      "Number of bytes received from server",
	}, []string{"group", "block"})
	BytesSentToClient = promauto.NewSummaryVec(prometheus.SummaryOpts{
		Namespace: Namespace,
		Name:      "bytes_sent_to_client",
		Help:      "Number of bytes sent to client",
	}, []string{"group", "block"})
	TotalTrafficBytes = promauto.NewSummaryVec(prometheus.SummaryOpts{
		Namespace: Namespace,
		Name:      "traffic_bytes",
		Help:      "Number of total bytes passed through GatewayD via client or server",
	}, []string{"group", "block"})
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
	ProxyHealthChecks = promauto.NewCounterVec(prometheus.CounterOpts{
		Namespace: Namespace,
		Name:      "proxy_health_checks_total",
		Help:      "Number of proxy health checks",
	}, []string{"group", "block"})
	ProxiedConnections = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: Namespace,
		Name:      "proxied_connections",
		Help:      "Number of proxy connects",
	}, []string{"group", "block"})
	ProxyPassThroughsToClient = promauto.NewCounterVec(prometheus.CounterOpts{
		Namespace: Namespace,
		Name:      "proxy_passthroughs_to_client_total",
		Help:      "Number of successful proxy passthroughs from server to client",
	}, []string{"group", "block"})
	ProxyPassThroughsToServer = promauto.NewCounterVec(prometheus.CounterOpts{
		Namespace: Namespace,
		Name:      "proxy_passthroughs_to_server_total",
		Help:      "Number of successful proxy passthroughs from client to server",
	}, []string{"group", "block"})
	ProxyPassThroughTerminations = promauto.NewCounterVec(prometheus.CounterOpts{
		Namespace: Namespace,
		Name:      "proxy_passthrough_terminations_total",
		Help:      "Number of proxy passthrough terminations by plugins",
	}, []string{"group", "block"})
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
