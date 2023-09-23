package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/collectors"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

const (
	Namespace         = "gatewayd"
	DefaultGroupLabel = "group"
)

type MetricRegistry struct {
	*prometheus.Registry

	// ClientConnections is the number of client connections
	ClientConnections prometheus.Gauge
	// ServerConnections is the number of server connections
	ServerConnections prometheus.Gauge
	// ServerTicksFired is the total number of server ticks fired
	ServerTicksFired prometheus.Counter
	// BytesReceivedFromClient is the number of bytes received from client
	BytesReceivedFromClient prometheus.Summary
	// BytesSentToServer is the number of bytes sent to server
	BytesSentToServer prometheus.Summary
	// BytesReceivedFromServer is the number of bytes received from server
	BytesReceivedFromServer prometheus.Summary
	// BytesSentToClient is the number of bytes sent to client
	BytesSentToClient prometheus.Summary
	// TotalTrafficBytes is the number of total bytes passed through GatewayD via client or server
	TotalTrafficBytes prometheus.Summary
	// PluginsLoaded is the number of plugins loaded
	PluginsLoaded prometheus.Counter
	// PluginHooksRegistered is the number of plugin hooks registered
	PluginHooksRegistered prometheus.Counter
	// PluginHooksExecuted is the number of plugin hooks executed
	PluginHooksExecuted prometheus.Counter
	// ProxyHealthChecks is the number of proxy health checks
	ProxyHealthChecks prometheus.Counter
	// ProxiedConnections is the number of proxy connects
	ProxiedConnections prometheus.Gauge
	// ProxyPassThroughs is the number of successful proxy passthroughs
	ProxyPassThroughs prometheus.Counter
	// ProxyPassThroughTerminations is the number of proxy passthrough terminations by plugins
	ProxyPassThroughTerminations prometheus.Counter
}

func NewMetricRegistry() *MetricRegistry {
	return &MetricRegistry{
		Registry: prometheus.NewRegistry(),
	}
}

func (r *MetricRegistry) Init(
	labels prometheus.Labels, registerGoMetrics, registerProcessMetrics bool,
) {
	r.ClientConnections = promauto.NewGauge(prometheus.GaugeOpts{
		Namespace:   Namespace,
		Name:        "client_connections",
		Help:        "Number of client connections",
		ConstLabels: labels,
	})
	r.ServerConnections = promauto.NewGauge(prometheus.GaugeOpts{
		Namespace:   Namespace,
		Name:        "server_connections",
		Help:        "Number of server connections",
		ConstLabels: labels,
	})
	r.ServerTicksFired = promauto.NewCounter(prometheus.CounterOpts{
		Namespace:   Namespace,
		Name:        "server_ticks_fired_total",
		Help:        "Total number of server ticks fired",
		ConstLabels: labels,
	})
	r.BytesReceivedFromClient = promauto.NewSummary(prometheus.SummaryOpts{
		Namespace:   Namespace,
		Name:        "bytes_received_from_client",
		Help:        "Number of bytes received from client",
		ConstLabels: labels,
	})
	r.BytesSentToServer = promauto.NewSummary(prometheus.SummaryOpts{
		Namespace:   Namespace,
		Name:        "bytes_sent_to_server",
		Help:        "Number of bytes sent to server",
		ConstLabels: labels,
	})
	r.BytesReceivedFromServer = promauto.NewSummary(prometheus.SummaryOpts{
		Namespace:   Namespace,
		Name:        "bytes_received_from_server",
		Help:        "Number of bytes received from server",
		ConstLabels: labels,
	})
	r.BytesSentToClient = promauto.NewSummary(prometheus.SummaryOpts{
		Namespace:   Namespace,
		Name:        "bytes_sent_to_client",
		Help:        "Number of bytes sent to client",
		ConstLabels: labels,
	})
	r.TotalTrafficBytes = promauto.NewSummary(prometheus.SummaryOpts{
		Namespace:   Namespace,
		Name:        "traffic_bytes",
		Help:        "Number of total bytes passed through GatewayD via client or server",
		ConstLabels: labels,
	})
	r.PluginsLoaded = promauto.NewCounter(prometheus.CounterOpts{
		Namespace:   Namespace,
		Name:        "plugins_loaded_total",
		Help:        "Number of plugins loaded",
		ConstLabels: labels,
	})
	r.PluginHooksRegistered = promauto.NewCounter(prometheus.CounterOpts{
		Namespace:   Namespace,
		Name:        "plugin_hooks_registered_total",
		Help:        "Number of plugin hooks registered",
		ConstLabels: labels,
	})
	r.PluginHooksExecuted = promauto.NewCounter(prometheus.CounterOpts{
		Namespace:   Namespace,
		Name:        "plugin_hooks_executed_total",
		Help:        "Number of plugin hooks executed",
		ConstLabels: labels,
	})
	r.ProxyHealthChecks = promauto.NewCounter(prometheus.CounterOpts{
		Namespace:   Namespace,
		Name:        "proxy_health_checks_total",
		Help:        "Number of proxy health checks",
		ConstLabels: labels,
	})
	r.ProxiedConnections = promauto.NewGauge(prometheus.GaugeOpts{
		Namespace:   Namespace,
		Name:        "proxied_connections",
		Help:        "Number of proxy connects",
		ConstLabels: labels,
	})
	r.ProxyPassThroughs = promauto.NewCounter(prometheus.CounterOpts{
		Namespace:   Namespace,
		Name:        "proxy_passthroughs_total",
		Help:        "Number of successful proxy passthroughs",
		ConstLabels: labels,
	})
	r.ProxyPassThroughTerminations = promauto.NewCounter(prometheus.CounterOpts{
		Namespace:   Namespace,
		Name:        "proxy_passthrough_terminations_total",
		Help:        "Number of proxy passthrough terminations by plugins",
		ConstLabels: labels,
	})

	// Register the metrics
	r.MustRegister(r.ClientConnections)
	r.MustRegister(r.ServerConnections)
	r.MustRegister(r.ServerTicksFired)
	r.MustRegister(r.BytesReceivedFromClient)
	r.MustRegister(r.BytesSentToServer)
	r.MustRegister(r.BytesReceivedFromServer)
	r.MustRegister(r.BytesSentToClient)
	r.MustRegister(r.TotalTrafficBytes)
	r.MustRegister(r.PluginsLoaded)
	r.MustRegister(r.PluginHooksRegistered)
	r.MustRegister(r.PluginHooksExecuted)
	r.MustRegister(r.ProxyHealthChecks)
	r.MustRegister(r.ProxiedConnections)
	r.MustRegister(r.ProxyPassThroughs)
	r.MustRegister(r.ProxyPassThroughTerminations)

	// Register the process and go collectors
	// Go collector is disabled by default, since it'll be shared by all instances
	if registerGoMetrics {
		r.MustRegister(collectors.NewGoCollector())
	}
	// Process collector is disabled by default, since it'll be shared by all instances
	if registerProcessMetrics {
		r.MustRegister(collectors.NewProcessCollector(collectors.ProcessCollectorOpts{}))
	}
}
