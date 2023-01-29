package plugin

import (
	"os/exec"

	goplugin "github.com/hashicorp/go-plugin"
)

const (
	minPort uint = 50000
	maxPort uint = 50002
)

type Clients struct {
	Clients []*goplugin.Client
}

func NewPluginClient() *Clients {
	return &Clients{
		Clients: make([]*goplugin.Client, 0, 1),
	}
}

func (p *Clients) Load() error {
	client := goplugin.NewClient(&goplugin.ClientConfig{
		HandshakeConfig: goplugin.HandshakeConfig{
			ProtocolVersion: 1,
			// MagicCookieKey:   "BASIC_PLUGIN",
			// MagicCookieValue: "hello",
		},
		Managed:          true,
		Plugins:          map[string]goplugin.Plugin{},
		Cmd:              exec.Command("python", "pluginA/pluginA_server.py"),
		AllowedProtocols: []goplugin.Protocol{goplugin.ProtocolGRPC, goplugin.ProtocolNetRPC},
		// VersionedPlugins: nil,
		// Logger:           nil,
		// TLSConfig:        nil,
		// Reattach:         nil,
		// SecureConfig:     nil,
		MinPort: minPort,
		MaxPort: maxPort,
		// StartTimeout: 10,
		// AutoMTLS:        false,
		// GRPCDialOptions: nil,
	})
	// defer client.Kill()
	p.Clients = append(p.Clients, client)

	return nil
}
