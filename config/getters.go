package config

import (
	"os"
	"path/filepath"
	"time"

	"github.com/rs/zerolog"
)

var (
	CompatibilityPolicies = map[string]CompatibilityPolicy{
		"strict": Strict,
		"loose":  Loose,
	}
	logOutputs = map[string]LogOutput{
		"console": Console,
		"stdout":  Stdout,
		"stderr":  Stderr,
		"file":    File,
		"syslog":  Syslog,
		"rsyslog": RSyslog,
	}
	TimeFormats = map[string]string{
		"":          zerolog.TimeFormatUnix,
		"unix":      zerolog.TimeFormatUnix,
		"unixms":    zerolog.TimeFormatUnixMs,
		"unixmicro": zerolog.TimeFormatUnixMicro,
		"unixnano":  zerolog.TimeFormatUnixNano,
	}
	ConsoleTimeFormats = map[string]string{
		"Layout":      time.Layout,
		"ANSIC":       time.ANSIC,
		"UnixDate":    time.UnixDate,
		"RubyDate":    time.RubyDate,
		"RFC822":      time.RFC822,
		"RFC822Z":     time.RFC822Z,
		"RFC850":      time.RFC850,
		"RFC1123":     time.RFC1123,
		"RFC1123Z":    time.RFC1123Z,
		"RFC3339":     time.RFC3339,
		"RFC3339Nano": time.RFC3339Nano,
		"Kitchen":     time.Kitchen,
		"Stamp":       time.Stamp,
		"StampMilli":  time.StampMilli,
		"StampMicro":  time.StampMicro,
		"StampNano":   time.StampNano,
	}
	LogLevels = map[string]zerolog.Level{
		"trace":    zerolog.TraceLevel,
		"debug":    zerolog.DebugLevel,
		"info":     zerolog.InfoLevel,
		"warn":     zerolog.WarnLevel,
		"error":    zerolog.ErrorLevel,
		"fatal":    zerolog.FatalLevel,
		"panic":    zerolog.PanicLevel,
		"disabled": zerolog.Disabled,
	}
)

// GetOutput returns the logger output from config file.
func (l Logger) GetOutput() []LogOutput {
	var outputs []LogOutput
	for _, output := range l.Output {
		if logOutput, ok := logOutputs[output]; ok {
			outputs = append(outputs, logOutput)
		} else {
			outputs = append(outputs, Console)
		}
	}

	if len(outputs) == 0 {
		outputs = append(outputs, Console)
	}

	return outputs
}

// GetPlugins returns the plugins from config file.
func (p PluginConfig) GetPlugins(name ...string) []Plugin {
	var plugins []Plugin
	for _, plugin := range p.Plugins {
		for _, n := range name {
			if plugin.Name == n {
				plugins = append(plugins, plugin)
			}
		}
	}
	return plugins
}

// GetDefaultConfigFilePath returns the path of the default config file.
func GetDefaultConfigFilePath(filename string) string {
	// Try to find the config file in the current directory.
	path := filepath.Join("./", filename)
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		return path
	}

	// Try to find the config file in the /etc directory.
	path = filepath.Join("/etc/", filename)
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		return path
	}

	// The fallback is the current directory.
	return filepath.Join("./", filename)
}

// Filter returns a filtered global config based on the group name.
func (gc GlobalConfig) Filter(groupName string) *GlobalConfig {
	if _, ok := gc.Servers[groupName]; !ok {
		return nil
	}
	return &GlobalConfig{
		Loggers: map[string]*Logger{groupName: gc.Loggers[groupName]},
		Clients: map[string]map[string]*Client{groupName: gc.Clients[groupName]},
		Pools:   map[string]map[string]*Pool{groupName: gc.Pools[groupName]},
		Proxies: map[string]map[string]*Proxy{groupName: gc.Proxies[groupName]},
		Servers: map[string]*Server{groupName: gc.Servers[groupName]},
		Metrics: map[string]*Metrics{groupName: gc.Metrics[groupName]},
		API:     gc.API,
	}
}
