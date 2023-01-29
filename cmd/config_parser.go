package cmd

import (
	"os"
	"time"

	"github.com/gatewayd-io/gatewayd/logging"
	"github.com/gatewayd-io/gatewayd/network"
	"github.com/gatewayd-io/gatewayd/plugin"
	"github.com/knadh/koanf"
	"github.com/panjf2000/gnet/v2"
	"github.com/rs/zerolog"
)

// Global koanf instance. Using "." as the key path delimiter.
var globalConfig = koanf.New(".")

// Plugin koanf instance. Using "." as the key path delimiter.
var pluginConfig = koanf.New(".")

// getPath returns the path to the referenced config value.
func getPath(path string) string {
	ref := globalConfig.String(path)
	if globalConfig.Exists(path) && globalConfig.StringMap(ref) != nil {
		return ref
	}

	return path
}

// func resolvePath(path string) map[string]string {
// 	ref := getPath(path)
// 	if ref != path {
// 		return konfig.StringMap(ref)
// 	}
// 	return nil
// }

// verificationPolicy returns the hook verification policy from plugin config file.
func verificationPolicy() plugin.Policy {
	vPolicy := globalConfig.String("plugins.verificationPolicy")
	verificationPolicy := plugin.PassDown // default
	switch vPolicy {
	case "ignore":
		verificationPolicy = plugin.Ignore
	case "abort":
		verificationPolicy = plugin.Abort
	case "remove":
		verificationPolicy = plugin.Remove
	}

	return verificationPolicy
}

// pluginCompatPolicy returns the plugin compatibility policy from plugin config file.
func pluginCompatPolicy() plugin.PluginCompatPolicy {
	vPolicy := pluginConfig.String("plugins.compatibilityPolicy")
	compatPolicy := plugin.Strict // default
	switch vPolicy {
	case "strict":
		compatPolicy = plugin.Strict
	case "loose":
		compatPolicy = plugin.Loose
	}

	return compatPolicy
}

// loggerConfig returns the logger config from config file.
func loggerConfig() logging.LoggerConfig {
	cfg := logging.LoggerConfig{StartupMsg: true}
	switch globalConfig.String("loggers.logger.output") {
	case "stdout":
		cfg.Output = os.Stdout
	case "console":
	default:
		cfg.Output = nil
	}

	switch globalConfig.String("loggers.logger.timeFormat") {
	case "unixms":
		cfg.TimeFormat = zerolog.TimeFormatUnixMs
	case "unixmicro":
		cfg.TimeFormat = zerolog.TimeFormatUnixMicro
	case "unixnano":
		cfg.TimeFormat = zerolog.TimeFormatUnixNano
	case "unix":
		cfg.TimeFormat = zerolog.TimeFormatUnix
	default:
		cfg.TimeFormat = zerolog.TimeFormatUnix
	}

	switch globalConfig.String("loggers.logger.level") {
	case "debug":
		cfg.Level = zerolog.DebugLevel
	case "info":
		cfg.Level = zerolog.InfoLevel
	case "warn":
		cfg.Level = zerolog.WarnLevel
	case "error":
		cfg.Level = zerolog.ErrorLevel
	case "fatal":
		cfg.Level = zerolog.FatalLevel
	case "panic":
		cfg.Level = zerolog.PanicLevel
	case "disabled":
		cfg.Level = zerolog.Disabled
	case "trace":
		cfg.Level = zerolog.TraceLevel
	default:
		cfg.Level = zerolog.InfoLevel
	}

	cfg.NoColor = globalConfig.Bool("loggers.logger.noColor")

	return cfg
}

// serverConfig returns the pool config from config file.
func poolConfig() (int, *network.Client) {
	poolSize := globalConfig.Int("pool.size")
	if poolSize == 0 {
		poolSize = network.DefaultPoolSize
	}

	// Minimum pool size is 2.
	if poolSize < network.MinimumPoolSize {
		poolSize = network.MinimumPoolSize
	}

	ref := getPath("pool.client")
	net := globalConfig.String(ref + ".network")
	address := globalConfig.String(ref + ".address")
	receiveBufferSize := globalConfig.Int(ref + ".receiveBufferSize")
	receiveChunkSize := globalConfig.Int(ref + ".receiveChunkSize")
	receiveDeadline := globalConfig.Duration(ref + ".receiveDeadline")
	sendDeadline := globalConfig.Duration(ref + ".sendDeadline")
	tcpKeepAlive := globalConfig.Bool(ref + ".tcpKeepAlive")
	tcpKeepAlivePeriod := globalConfig.Duration(ref + ".tcpKeepAlivePeriod")

	return poolSize, &network.Client{
		Network:            net,
		Address:            address,
		TCPKeepAlive:       tcpKeepAlive,
		TCPKeepAlivePeriod: tcpKeepAlivePeriod,
		ReceiveBufferSize:  receiveBufferSize,
		ReceiveChunkSize:   receiveChunkSize,
		ReceiveDeadline:    receiveDeadline,
		SendDeadline:       sendDeadline,
	}
}

// proxyConfig returns the proxy config from config file.
func proxyConfig() (bool, bool, *network.Client) {
	elastic := globalConfig.Bool("proxy.elastic")
	reuseElasticClients := globalConfig.Bool("proxy.reuseElasticClients")

	ref := getPath("pool.client")
	net := globalConfig.String(ref + ".network")
	address := globalConfig.String(ref + ".address")
	receiveBufferSize := globalConfig.Int(ref + ".receiveBufferSize")
	receiveChunkSize := globalConfig.Int(ref + ".receiveChunkSize")
	receiveDeadline := globalConfig.Duration(ref + ".receiveDeadline")
	sendDeadline := globalConfig.Duration(ref + ".sendDeadline")
	tcpKeepAlive := globalConfig.Bool(ref + ".tcpKeepAlive")
	tcpKeepAlivePeriod := globalConfig.Duration(ref + ".tcpKeepAlivePeriod")

	if receiveBufferSize <= 0 {
		receiveBufferSize = network.DefaultBufferSize
	}

	if receiveChunkSize <= 0 {
		receiveChunkSize = network.DefaultChunkSize
	}

	if tcpKeepAlive && tcpKeepAlivePeriod <= 0 {
		tcpKeepAlivePeriod = network.DefaultTCPKeepAlivePeriod
	}

	return elastic, reuseElasticClients, &network.Client{
		Network:            net,
		Address:            address,
		TCPKeepAlive:       tcpKeepAlive,
		TCPKeepAlivePeriod: tcpKeepAlivePeriod,
		ReceiveBufferSize:  receiveBufferSize,
		ReceiveChunkSize:   receiveChunkSize,
		ReceiveDeadline:    receiveDeadline,
		SendDeadline:       sendDeadline,
	}
}

type ServerConfig struct {
	Network          string
	Address          string
	SoftLimit        uint64
	HardLimit        uint64
	EnableTicker     bool
	MultiCore        bool
	LockOSThread     bool
	ReuseAddress     bool
	ReusePort        bool
	LoadBalancer     gnet.LoadBalancing
	TickInterval     time.Duration
	ReadBufferCap    int
	WriteBufferCap   int
	SocketRecvBuffer int
	SocketSendBuffer int
	TCPKeepAlive     time.Duration
	TCPNoDelay       gnet.TCPSocketOpt
}

var loadBalancer = map[string]gnet.LoadBalancing{
	"roundrobin":       gnet.RoundRobin,
	"leastconnections": gnet.LeastConnections,
	"sourceaddrhash":   gnet.SourceAddrHash,
}

// getLoadBalancer returns the load balancer from config file.
func getLoadBalancer(name string) gnet.LoadBalancing {
	if lb, ok := loadBalancer[name]; ok {
		return lb
	}

	return gnet.RoundRobin
}

// getTCPNoDelay returns the TCP no delay option from config file.
func getTCPNoDelay() gnet.TCPSocketOpt {
	if globalConfig.Bool("server.tcpNoDelay") {
		return gnet.TCPNoDelay
	}

	return gnet.TCPDelay
}

// serverConfig returns the server config from config file.
func serverConfig() *ServerConfig {
	readBufferCap := globalConfig.Int("server.readBufferCap")
	if readBufferCap <= 0 {
		readBufferCap = network.DefaultBufferSize
	}

	writeBufferCap := globalConfig.Int("server.writeBufferCap")
	if writeBufferCap <= 0 {
		writeBufferCap = network.DefaultBufferSize
	}

	socketRecvBuffer := globalConfig.Int("server.socketRecvBuffer")
	if socketRecvBuffer <= 0 {
		socketRecvBuffer = network.DefaultBufferSize
	}

	socketSendBuffer := globalConfig.Int("server.socketSendBuffer")
	if socketSendBuffer <= 0 {
		socketSendBuffer = network.DefaultBufferSize
	}

	return &ServerConfig{
		Network:          globalConfig.String("server.network"),
		Address:          globalConfig.String("server.address"),
		SoftLimit:        uint64(globalConfig.Int64("server.softLimit")),
		HardLimit:        uint64(globalConfig.Int64("server.hardLimit")),
		EnableTicker:     globalConfig.Bool("server.enableTicker"),
		TickInterval:     globalConfig.Duration("server.tickInterval"),
		MultiCore:        globalConfig.Bool("server.multiCore"),
		LockOSThread:     globalConfig.Bool("server.lockOSThread"),
		LoadBalancer:     getLoadBalancer(globalConfig.String("server.loadBalancer")),
		ReadBufferCap:    readBufferCap,
		WriteBufferCap:   writeBufferCap,
		SocketRecvBuffer: socketRecvBuffer,
		SocketSendBuffer: socketSendBuffer,
		ReuseAddress:     globalConfig.Bool("server.reuseAddress"),
		ReusePort:        globalConfig.Bool("server.reusePort"),
		TCPKeepAlive:     globalConfig.Duration("server.tcpKeepAlive"),
		TCPNoDelay:       getTCPNoDelay(),
	}
}
