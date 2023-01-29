package v1

import (
	"context"

	goplugin "github.com/hashicorp/go-plugin"
	"google.golang.org/grpc"
)

// Handshake must be used by all plugins to ensure that the plugin
// and host are compatible.
var Handshake = goplugin.HandshakeConfig{
	ProtocolVersion:  1,
	MagicCookieKey:   "GATEWAYD_PLUGIN",
	MagicCookieValue: "5712b87aa5d7e9f9e9ab643e6603181c5b796015cb1c09d6f5ada882bf2a1872",
}

// GetPluginMap returns the plugin map for the plugin.
func GetPluginMap(pluginName string) map[string]goplugin.Plugin {
	return map[string]goplugin.Plugin{
		pluginName: &Plugin{},
	}
}

// Plugin is the interface that all plugins must implement.
type Plugin struct {
	goplugin.GRPCPlugin
	goplugin.NetRPCUnsupportedPlugin
	Impl struct {
		GatewayDPluginServiceServer
	}
}

// GRPCServer registers the plugin with the gRPC server.
func (p *Plugin) GRPCServer(b *goplugin.GRPCBroker, s *grpc.Server) error {
	RegisterGatewayDPluginServiceServer(s, &p.Impl)
	return nil
}

// GRPCClient returns the plugin client.
func (p *Plugin) GRPCClient(ctx context.Context, b *goplugin.GRPCBroker, c *grpc.ClientConn) (interface{}, error) {
	return NewGatewayDPluginServiceClient(c), nil
}
