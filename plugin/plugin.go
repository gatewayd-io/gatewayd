package plugin

import (
	"os/exec"

	goplugin "github.com/hashicorp/go-plugin"
)

type PluginClients struct {
	Clients []*goplugin.Client
}

func NewPluginClient() *PluginClients {
	return &PluginClients{
		Clients: make([]*goplugin.Client, 0, 1),
	}
}

func (p *PluginClients) Load() error {
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
		MinPort: 50000,
		MaxPort: 50002,
		// StartTimeout: 10,
		// AutoMTLS:        false,
		// GRPCDialOptions: nil,
	})
	// defer client.Kill()
	p.Clients = append(p.Clients, client)

	return nil
}
