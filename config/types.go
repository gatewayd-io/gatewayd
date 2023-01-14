package config

import (
	"fmt"
	"strings"
	"time"

	"github.com/knadh/koanf"
	"github.com/knadh/koanf/providers/confmap"
	"github.com/knadh/koanf/providers/env"
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
	VerificationPolicy  string   `koanf:"verificationPolicy"`
	CompatibilityPolicy string   `koanf:"compatibilityPolicy"`
	AcceptancePolicy    string   `koanf:"acceptancePolicy"`
	Plugins             []Plugin `koanf:"plugins"`
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
	Output     string `koanf:"output"`
	TimeFormat string `koanf:"timeFormat"`
	Level      string `koanf:"level"`
	NoColor    bool   `koanf:"noColor"`
	StartupMsg bool   `koanf:"startupMsg"`
	FileName   string `koanf:"fileName"`
	MaxSize    int    `koanf:"maxSize"`
	MaxBackups int    `koanf:"maxBackups"`
	MaxAge     int    `koanf:"maxAge"`
	Compress   bool   `koanf:"compress"`
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

type GlobalConfig struct {
	Loggers map[string]Logger `koanf:"loggers"`
	Clients map[string]Client `koanf:"clients"`
	Pools   map[string]Pool   `koanf:"pools"`
	Proxy   map[string]Proxy  `koanf:"proxy"`
	Server  Server            `koanf:"server"`
}

// LoadDefaultConfig loads the default configuration before loading the config file.
func LoadGlobalConfigDefaults(cfg *koanf.Koanf) {
	defaultValues := confmap.Provider(map[string]interface{}{
		"loggers": map[string]interface{}{
			"default": map[string]interface{}{
				"output":     DefaultLogOutput,
				"level":      DefaultLogLevel,
				"fileName":   DefaultLogFileName,
				"maxSize":    DefaultMaxSize,
				"maxBackups": DefaultMaxBackups,
				"maxAge":     DefaultMaxAge,
				"compress":   DefaultCompress,
			},
		},
		"clients": map[string]interface{}{
			"default": map[string]interface{}{
				"receiveBufferSize":  DefaultBufferSize,
				"receiveChunkSize":   DefaultChunkSize,
				"tcpKeepAlivePeriod": DefaultTCPKeepAlivePeriod.String(),
			},
		},
		"pools": map[string]interface{}{
			"default": map[string]interface{}{
				"size": DefaultPoolSize,
			},
		},
		"proxy": map[string]interface{}{
			"default": map[string]interface{}{
				"elastic":             false,
				"reuseElasticClients": false,
				"healthCheckPeriod":   DefaultHealthCheckPeriod.String(),
			},
		},
		"server": map[string]interface{}{
			"network":          DefaultListenNetwork,
			"address":          DefaultListenAddress,
			"softLimit":        0,
			"hardLimit":        0,
			"enableTicker":     false,
			"multiCore":        true,
			"lockOSThread":     false,
			"reuseAddress":     true,
			"reusePort":        true,
			"loadBalancer":     DefaultLoadBalancer,
			"readBufferCap":    DefaultBufferSize,
			"writeBufferCap":   DefaultBufferSize,
			"socketRecvBuffer": DefaultBufferSize,
			"socketSendBuffer": DefaultBufferSize,
		},
	}, "")

	if err := cfg.Load(defaultValues, nil); err != nil {
		panic(fmt.Errorf("failed to load default global configuration: %w", err))
	}
}

// LoadPluginConfigDefaults loads the default plugin configuration
// before loading the plugin config file.
func LoadPluginConfigDefaults(cfg *koanf.Koanf) {
	defaultValues := confmap.Provider(map[string]interface{}{
		"plugins": map[string]interface{}{
			"verificationPolicy":  "passdown",
			"compatibilityPolicy": "strict",
		},
	}, "")

	if err := cfg.Load(defaultValues, nil); err != nil {
		panic(fmt.Errorf("failed to load default plugin configuration: %w", err))
	}
}

// LoadEnvVars loads the environment variables into the configuration with the
// given prefix, "GATEWAYD_".
func LoadEnvVars(cfg *koanf.Koanf) {
	if err := cfg.Load(env.Provider(EnvPrefix, ".", func(env string) string {
		return strings.ReplaceAll(strings.ToLower(strings.TrimPrefix(env, EnvPrefix)), "_", ".")
	}), nil); err != nil {
		panic(fmt.Errorf("failed to load environment variables: %w", err))
	}
}
