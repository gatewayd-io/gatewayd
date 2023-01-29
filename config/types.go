package config

import (
	"time"
)

// // getPath returns the path to the referenced config value.
// func getPath(cfg *koanf.Koanf, path string) string {
// 	ref := cfg.String(path)
// 	if cfg.Exists(path) && cfg.StringMap(ref) != nil {
// 		return ref
// 	}

// 	return path
// }

type Plugin struct {
	Name      string   `koanf:"name"`
	Enabled   bool     `koanf:"enabled"`
	LocalPath string   `koanf:"localPath"`
	Args      []string `koanf:"args"`
	Env       []string `koanf:"env"`
	Checksum  string   `koanf:"checksum"`
}

type PluginConfig struct {
	VerificationPolicy  string        `koanf:"verificationPolicy"`
	CompatibilityPolicy string        `koanf:"compatibilityPolicy"`
	AcceptancePolicy    string        `koanf:"acceptancePolicy"`
	MetricsMergerPeriod time.Duration `koanf:"metricsMergerPeriod"`
	Plugins             []Plugin      `koanf:"plugins"`
}

type Client struct {
	Network            string        `koanf:"network"`
	Address            string        `koanf:"address"`
	TCPKeepAlive       bool          `koanf:"tcpKeepAlive"`
	TCPKeepAlivePeriod time.Duration `koanf:"tcpKeepAlivePeriod"`
	ReceiveBufferSize  int           `koanf:"receiveBufferSize"`
	ReceiveChunkSize   int           `koanf:"receiveChunkSize"`
	ReceiveDeadline    time.Duration `koanf:"receiveDeadline"`
	SendDeadline       time.Duration `koanf:"sendDeadline"`
}

type Logger struct {
	Output            []string `koanf:"output"`
	TimeFormat        string   `koanf:"timeFormat"`
	Level             string   `koanf:"level"`
	ConsoleTimeFormat string   `koanf:"consoleTimeFormat"`
	NoColor           bool     `koanf:"noColor"`
	StartupMsg        bool     `koanf:"startupMsg"`
	FileName          string   `koanf:"fileName"`
	MaxSize           int      `koanf:"maxSize"`
	MaxBackups        int      `koanf:"maxBackups"`
	MaxAge            int      `koanf:"maxAge"`
	Compress          bool     `koanf:"compress"`
}

type Pool struct {
	Size int `koanf:"size"`
}

type Proxy struct {
	Elastic             bool          `koanf:"elastic"`
	ReuseElasticClients bool          `koanf:"reuseElasticClients"`
	HealthCheckPeriod   time.Duration `koanf:"healthCheckPeriod"`
}

type Server struct {
	EnableTicker     bool          `koanf:"enableTicker"`
	MultiCore        bool          `koanf:"multiCore"`
	LockOSThread     bool          `koanf:"lockOSThread"`
	ReuseAddress     bool          `koanf:"reuseAddress"`
	ReusePort        bool          `koanf:"reusePort"`
	TCPNoDelay       bool          `koanf:"tcpNoDelay"`
	ReadBufferCap    int           `koanf:"readBufferCap"`
	WriteBufferCap   int           `koanf:"writeBufferCap"`
	SocketRecvBuffer int           `koanf:"socketRecvBuffer"`
	SocketSendBuffer int           `koanf:"socketSendBuffer"`
	SoftLimit        uint64        `koanf:"softLimit"`
	HardLimit        uint64        `koanf:"hardLimit"`
	TCPKeepAlive     time.Duration `koanf:"tcpKeepAlive"`
	TickInterval     time.Duration `koanf:"tickInterval"`
	Network          string        `koanf:"network"`
	Address          string        `koanf:"address"`
	LoadBalancer     string        `koanf:"loadBalancer"`
}

type Metrics struct {
	Enabled bool   `koanf:"enabled"`
	Address string `koanf:"address"`
	Path    string `koanf:"path"`
}

type GlobalConfig struct {
	Loggers map[string]Logger  `koanf:"loggers"`
	Clients map[string]Client  `koanf:"clients"`
	Pools   map[string]Pool    `koanf:"pools"`
	Proxy   map[string]Proxy   `koanf:"proxy"`
	Server  Server             `koanf:"server"`
	Metrics map[string]Metrics `koanf:"metrics"`
}
