package plugin

import (
	"github.com/gatewayd-io/gatewayd-plugin-sdk/metrics"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// The following metrics are defined in the plugin and are used to
// track the number of times the plugin methods are called. These
// metrics are used as examples to test the plugin metrics functionality.
var (
	GetPluginConfig = promauto.NewCounter(prometheus.CounterOpts{
		Namespace: metrics.Namespace,
		Name:      "get_plugin_config_total",
		Help:      "The total number of calls to the getPluginConfig method",
	})
	OnConfigLoaded = promauto.NewCounter(prometheus.CounterOpts{
		Namespace: metrics.Namespace,
		Help:      "The total number of calls to the onConfigLoaded method",
		Name:      "on_config_loaded_total",
	})
	OnNewLogger = promauto.NewCounter(prometheus.CounterOpts{
		Namespace: metrics.Namespace,
		Name:      "on_new_logger_total",
		Help:      "The total number of calls to the onNewLogger method",
	})
	OnNewPool = promauto.NewCounter(prometheus.CounterOpts{
		Namespace: metrics.Namespace,
		Name:      "on_new_pool_total",
		Help:      "The total number of calls to the onNewPool method",
	})
	OnNewClient = promauto.NewCounter(prometheus.CounterOpts{
		Namespace: metrics.Namespace,
		Name:      "on_new_client_total",
		Help:      "The total number of calls to the onNewClient method",
	})
	OnNewProxy = promauto.NewCounter(prometheus.CounterOpts{
		Namespace: metrics.Namespace,
		Name:      "on_new_proxy_total",
		Help:      "The total number of calls to the onNewProxy method",
	})
	OnNewServer = promauto.NewCounter(prometheus.CounterOpts{
		Namespace: metrics.Namespace,
		Name:      "on_new_server_total",
		Help:      "The total number of calls to the onNewServer method",
	})
	OnSignal = promauto.NewCounter(prometheus.CounterOpts{
		Namespace: metrics.Namespace,
		Name:      "on_signal_total",
		Help:      "The total number of calls to the onSignal method",
	})
	OnRun = promauto.NewCounter(prometheus.CounterOpts{
		Namespace: metrics.Namespace,
		Name:      "on_run_total",
		Help:      "The total number of calls to the onRun method",
	})
	OnBooting = promauto.NewCounter(prometheus.CounterOpts{
		Namespace: metrics.Namespace,
		Name:      "on_booting_total",
		Help:      "The total number of calls to the onBooting method",
	})
	OnBooted = promauto.NewCounter(prometheus.CounterOpts{
		Namespace: metrics.Namespace,
		Name:      "on_booted_total",
		Help:      "The total number of calls to the onBooted method",
	})
	OnOpening = promauto.NewCounter(prometheus.CounterOpts{
		Namespace: metrics.Namespace,
		Name:      "on_opening_total",
		Help:      "The total number of calls to the onOpening method",
	})
	OnOpened = promauto.NewCounter(prometheus.CounterOpts{
		Namespace: metrics.Namespace,
		Name:      "on_opened_total",
		Help:      "The total number of calls to the onOpened method",
	})
	OnClosing = promauto.NewCounter(prometheus.CounterOpts{
		Namespace: metrics.Namespace,
		Name:      "on_closing_total",
		Help:      "The total number of calls to the onClosing method",
	})
	OnClosed = promauto.NewCounter(prometheus.CounterOpts{
		Namespace: metrics.Namespace,
		Name:      "on_closed_total",
		Help:      "The total number of calls to the onClosed method",
	})
	OnTraffic = promauto.NewCounter(prometheus.CounterOpts{
		Namespace: metrics.Namespace,
		Name:      "on_traffic_total",
		Help:      "The total number of calls to the onTraffic method",
	})
	OnShutdown = promauto.NewCounter(prometheus.CounterOpts{
		Namespace: metrics.Namespace,
		Name:      "on_shutdown_total",
		Help:      "The total number of calls to the onShutdown method",
	})
	OnTick = promauto.NewCounter(prometheus.CounterOpts{
		Namespace: metrics.Namespace,
		Name:      "on_tick_total",
		Help:      "The total number of calls to the onTick method",
	})
	OnTrafficFromClient = promauto.NewCounter(prometheus.CounterOpts{
		Namespace: metrics.Namespace,
		Name:      "on_traffic_from_client_total",
		Help:      "The total number of calls to the onTrafficFromClient method",
	})
	OnTrafficToServer = promauto.NewCounter(prometheus.CounterOpts{
		Namespace: metrics.Namespace,
		Name:      "on_traffic_to_server_total",
		Help:      "The total number of calls to the onTrafficToServer method",
	})
	OnTrafficFromServer = promauto.NewCounter(prometheus.CounterOpts{
		Namespace: metrics.Namespace,
		Name:      "on_traffic_from_server_total",
		Help:      "The total number of calls to the onTrafficFromServer method",
	})
	OnTrafficToClient = promauto.NewCounter(prometheus.CounterOpts{
		Namespace: metrics.Namespace,
		Name:      "on_traffic_to_client_total",
		Help:      "The total number of calls to the onTrafficToClient method",
	})
)
