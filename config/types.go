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
	EnableMetricsMerger bool          `koanf:"enableMetricsMerger"`
	MetricsMergerPeriod time.Duration `koanf:"metricsMergerPeriod"`
	HealthCheckPeriod   time.Duration `koanf:"healthCheckPeriod"`
	ReloadOnCrash       bool          `koanf:"reloadOnCrash"`
	Plugins             []Plugin      `koanf:"plugins"`
}

type Client struct {
	Network            string        `koanf:"network"`
	Address            string        `koanf:"address"`
	TCPKeepAlive       bool          `koanf:"tcpKeepAlive"`
	TCPKeepAlivePeriod time.Duration `koanf:"tcpKeepAlivePeriod"`
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

	FileName   string `koanf:"fileName"`
	MaxSize    int    `koanf:"maxSize"`
	MaxBackups int    `koanf:"maxBackups"`
	MaxAge     int    `koanf:"maxAge"`
	Compress   bool   `koanf:"compress"`
	LocalTime  bool   `koanf:"localTime"`

	RSyslogNetwork string `koanf:"rSyslogNetwork"`
	RSyslogAddress string `koanf:"rSyslogAddress"`
	SyslogPriority string `koanf:"syslogPriority"`
}

type Metrics struct {
	Enabled bool   `koanf:"enabled"`
	Address string `koanf:"address"`
	Path    string `koanf:"path"`
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

type API struct {
	Enabled     bool   `koanf:"enabled"`
	HTTPAddress string `koanf:"httpAddress"`
	GRPCAddress string `koanf:"grpcAddress"`
	GRPCNetwork string `koanf:"grpcNetwork"`
}

type GlobalConfig struct {
	API     API                `koanf:"api"`
	Loggers map[string]Logger  `koanf:"loggers"`
	Clients map[string]Client  `koanf:"clients"`
	Pools   map[string]Pool    `koanf:"pools"`
	Proxies map[string]Proxy   `koanf:"proxies"`
	Servers map[string]Server  `koanf:"servers"`
	Metrics map[string]Metrics `koanf:"metrics"`
}
