package cmd

import (
	"os"
	"time"

	"github.com/gatewayd-io/gatewayd/logging"
	"github.com/gatewayd-io/gatewayd/network"
	"github.com/knadh/koanf"
	"github.com/panjf2000/gnet/v2"
	"github.com/rs/zerolog"
)

// Global koanf instance. Using "." as the key path delimiter.
var konfig = koanf.New(".")

func getPath(path string) string {
	ref := konfig.String(path)
	if konfig.Exists(path) && konfig.StringMap(ref) != nil {
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

func loggerConfig() logging.LoggerConfig {
	cfg := logging.LoggerConfig{}
	switch konfig.String("loggers.logger.output") {
	case "stdout":
		cfg.Output = os.Stdout
	case "console":
	default:
		cfg.Output = nil
	}

	switch konfig.String("loggers.logger.timeFormat") {
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

	switch konfig.String("loggers.logger.level") {
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

	cfg.NoColor = konfig.Bool("loggers.logger.noColor")

	return cfg
}

func poolConfig() (int, *network.Client) {
	poolSize := konfig.Int("pool.size")
	if poolSize == 0 {
		poolSize = network.DefaultPoolSize
	}

	ref := getPath("pool.client")
	net := konfig.String(ref + ".network")
	address := konfig.String(ref + ".address")
	receiveBufferSize := konfig.Int(ref + ".receiveBufferSize")

	return poolSize, &network.Client{
		Network:           net,
		Address:           address,
		ReceiveBufferSize: receiveBufferSize,
	}
}

func proxyConfig() (bool, bool, *network.Client) {
	elastic := konfig.Bool("proxy.elastic")
	reuseElasticClients := konfig.Bool("proxy.reuseElasticClients")

	ref := getPath("pool.client")
	net := konfig.String(ref + ".network")
	address := konfig.String(ref + ".address")
	receiveBufferSize := konfig.Int(ref + ".receiveBufferSize")

	return elastic, reuseElasticClients, &network.Client{
		Network:           net,
		Address:           address,
		ReceiveBufferSize: receiveBufferSize,
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
	// OnIncomingTraffic string
	// OnOutgoingTraffic string
}

var loadBalancer = map[string]gnet.LoadBalancing{
	"roundrobin":       gnet.RoundRobin,
	"leastconnections": gnet.LeastConnections,
	"sourceaddrhash":   gnet.SourceAddrHash,
}

func getLoadBalancer(name string) gnet.LoadBalancing {
	if lb, ok := loadBalancer[name]; ok {
		return lb
	}

	return gnet.RoundRobin
}

func getTCPNoDelay() gnet.TCPSocketOpt {
	if konfig.Bool("server.tcpNoDelay") {
		return gnet.TCPNoDelay
	}

	return gnet.TCPDelay
}

func serverConfig() *ServerConfig {
	return &ServerConfig{
		Network:          konfig.String("server.network"),
		Address:          konfig.String("server.address"),
		SoftLimit:        uint64(konfig.Int64("server.softLimit")),
		HardLimit:        uint64(konfig.Int64("server.hardLimit")),
		EnableTicker:     konfig.Bool("server.enableTicker"),
		TickInterval:     konfig.Duration("server.tickInterval") * time.Second,
		MultiCore:        konfig.Bool("server.multiCore"),
		LockOSThread:     konfig.Bool("server.lockOSThread"),
		LoadBalancer:     getLoadBalancer(konfig.String("server.loadBalancer")),
		ReadBufferCap:    konfig.Int("server.readBufferCap"),
		WriteBufferCap:   konfig.Int("server.writeBufferCap"),
		SocketRecvBuffer: konfig.Int("server.socketRecvBuffer"),
		SocketSendBuffer: konfig.Int("server.socketSendBuffer"),
		ReuseAddress:     konfig.Bool("server.reuseAddress"),
		ReusePort:        konfig.Bool("server.reusePort"),
		TCPKeepAlive:     konfig.Duration("server.tcpKeepAlive") * time.Second,
		TCPNoDelay:       getTCPNoDelay(),
		// OnIncomingTraffic: konfig.String("server.onIncomingTraffic"),
		// OnOutgoingTraffic: konfig.String("server.onOutgoingTraffic"),
	}
}
