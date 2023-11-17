//nolint:lll
package config

import (
	"time"
)

type Plugin struct {
	Name      string   `json:"name" jsonschema:"required"`
	Enabled   bool     `json:"enabled"`
	LocalPath string   `json:"localPath" jsonschema:"required"`
	Args      []string `json:"args"`
	Env       []string `json:"env" jsonschema:"required"`
	Checksum  string   `json:"checksum" jsonschema:"required"`
}

type PluginConfig struct {
	VerificationPolicy  string        `json:"verificationPolicy" jsonschema:"enum=passdown,enum=ignore,enum=abort,enum=remove"`
	CompatibilityPolicy string        `json:"compatibilityPolicy" jsonschema:"enum=strict,enum=loose"`
	AcceptancePolicy    string        `json:"acceptancePolicy" jsonschema:"enum=accept,enum=reject"`
	TerminationPolicy   string        `json:"terminationPolicy" jsonschema:"enum=continue,enum=stop"`
	EnableMetricsMerger bool          `json:"enableMetricsMerger"`
	MetricsMergerPeriod time.Duration `json:"metricsMergerPeriod" jsonschema:"oneof_type=string;integer"`
	HealthCheckPeriod   time.Duration `json:"healthCheckPeriod" jsonschema:"oneof_type=string;integer"`
	ReloadOnCrash       bool          `json:"reloadOnCrash"`
	Timeout             time.Duration `json:"timeout" jsonschema:"oneof_type=string;integer"`
	StartTimeout        time.Duration `json:"startTimeout" jsonschema:"oneof_type=string;integer"`
	Plugins             []Plugin      `json:"plugins"`
}

type Client struct {
	Network            string        `json:"network" jsonschema:"enum=tcp,enum=udp,enum=unix"`
	Address            string        `json:"address"`
	TCPKeepAlive       bool          `json:"tcpKeepAlive"`
	TCPKeepAlivePeriod time.Duration `json:"tcpKeepAlivePeriod" jsonschema:"oneof_type=string;integer"`
	ReceiveChunkSize   int           `json:"receiveChunkSize"`
	ReceiveDeadline    time.Duration `json:"receiveDeadline" jsonschema:"oneof_type=string;integer"`
	ReceiveTimeout     time.Duration `json:"receiveTimeout" jsonschema:"oneof_type=string;integer"`
	SendDeadline       time.Duration `json:"sendDeadline" jsonschema:"oneof_type=string;integer"`
	DialTimeout        time.Duration `json:"dialTimeout" jsonschema:"oneof_type=string;integer"`
}

type Logger struct {
	Output            []string `json:"output"`
	TimeFormat        string   `json:"timeFormat" jsonschema:"enum=unix,enum=unixms,enum=unixmicro,enum=unixnano"`
	Level             string   `json:"level" jsonschema:"enum=trace,enum=debug,enum=info,enum=warn,enum=error,enum=fatal,enum=panic,enum=disabled"`
	ConsoleTimeFormat string   `json:"consoleTimeFormat" jsonschema:"enum=Layout,enum=ANSIC,enum=UnixDate,enum=RubyDate,enum=RFC822,enum=RFC822Z,enum=RFC850,enum=RFC1123,enum=RFC1123Z,enum=RFC3339,enum=RFC3339Nano,enum=Kitchen,enum=Stamp,enum=StampMilli,enum=StampMicro,enum=StampNano"`
	NoColor           bool     `json:"noColor"`

	FileName   string `json:"fileName"`
	MaxSize    int    `json:"maxSize"`
	MaxBackups int    `json:"maxBackups"`
	MaxAge     int    `json:"maxAge"`
	Compress   bool   `json:"compress"`
	LocalTime  bool   `json:"localTime"`

	RSyslogNetwork string `json:"rsyslogNetwork" jsonschema:"enum=tcp,enum=udp,enum=unix"`
	RSyslogAddress string `json:"rsyslogAddress"`
	SyslogPriority string `json:"syslogPriority" jsonschema:"enum=debug,enum=info,enum=notice,enum=warning,enum=err,enum=crit,enum=alert,enum=emerg"`
}

type Metrics struct {
	Enabled           bool          `json:"enabled"`
	Address           string        `json:"address"`
	Path              string        `json:"path"`
	ReadHeaderTimeout time.Duration `json:"readHeaderTimeout" jsonschema:"oneof_type=string;integer"`
	Timeout           time.Duration `json:"timeout" jsonschema:"oneof_type=string;integer"`
	CertFile          string        `json:"certFile"`
	KeyFile           string        `json:"keyFile"`
}

type Pool struct {
	Size int `json:"size"`
}

type Proxy struct {
	Elastic             bool          `json:"elastic"`
	ReuseElasticClients bool          `json:"reuseElasticClients"`
	HealthCheckPeriod   time.Duration `json:"healthCheckPeriod" jsonschema:"oneof_type=string;integer"`
}

type Server struct {
	EnableTicker     bool          `json:"enableTicker"`
	TickInterval     time.Duration `json:"tickInterval" jsonschema:"oneof_type=string;integer"`
	Network          string        `json:"network" jsonschema:"enum=tcp,enum=udp,enum=unix"`
	Address          string        `json:"address"`
	EnableTLS        bool          `json:"enableTLS"` //nolint:tagliatelle
	CertFile         string        `json:"certFile"`
	KeyFile          string        `json:"keyFile"`
	HandshakeTimeout time.Duration `json:"handshakeTimeout" jsonschema:"oneof_type=string;integer"`
}

type API struct {
	Enabled     bool   `json:"enabled"`
	HTTPAddress string `json:"httpAddress"`
	GRPCAddress string `json:"grpcAddress"`
	GRPCNetwork string `json:"grpcNetwork" jsonschema:"enum=tcp,enum=udp,enum=unix"`
}

type GlobalConfig struct {
	API     API                 `json:"api"`
	Loggers map[string]*Logger  `json:"loggers"`
	Clients map[string]*Client  `json:"clients"`
	Pools   map[string]*Pool    `json:"pools"`
	Proxies map[string]*Proxy   `json:"proxies"`
	Servers map[string]*Server  `json:"servers"`
	Metrics map[string]*Metrics `json:"metrics"`
}
