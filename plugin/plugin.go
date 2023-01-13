package plugin

import (
	"net"

	gerr "github.com/gatewayd-io/gatewayd/errors"
	pluginV1 "github.com/gatewayd-io/gatewayd/plugin/v1"
	goplugin "github.com/hashicorp/go-plugin"
)

type IPlugin interface {
	Start() (net.Addr, error)
	Stop()
	Dispense() (pluginV1.GatewayDPluginServiceClient, *gerr.GatewayDError)
}

type Identifier struct {
	Name      string
	Version   string
	RemoteURL string
	Checksum  string
}

type Plugin struct {
	goplugin.NetRPCUnsupportedPlugin
	pluginV1.GatewayDPluginServiceServer

	client *goplugin.Client

	ID          Identifier
	Description string
	Authors     []string
	License     string
	ProjectURL  string
	LocalPath   string
	Args        []string
	Env         []string
	Enabled     bool
	// internal and external config options
	Config map[string]string
	// hooks it attaches to
	Hooks    []string
	Priority Priority
	// required plugins to be loaded before this one
	// Built-in plugins are always loaded first
	Requires   []Identifier
	Tags       []string
	Categories []string
}

var _ IPlugin = &Plugin{}

// Start starts the plugin.
func (p *Plugin) Start() (net.Addr, error) {
	var addr net.Addr
	var err error
	if addr, err = p.client.Start(); err != nil {
		return nil, gerr.ErrFailedToStartPlugin.Wrap(err)
	}
	return addr, nil
}

// Stop kills the plugin.
func (p *Plugin) Stop() {
	p.client.Kill()
}

// Dispense returns the plugin client.
func (p *Plugin) Dispense() (pluginV1.GatewayDPluginServiceClient, *gerr.GatewayDError) {
	rpcClient, err := p.client.Client()
	if err != nil {
		return nil, gerr.ErrFailedToGetRPCClient.Wrap(err)
	}

	raw, err := rpcClient.Dispense(p.ID.Name)
	if err != nil {
		return nil, gerr.ErrFailedToDispensePlugin.Wrap(err)
	}

	if gatewaydPlugin, ok := raw.(pluginV1.GatewayDPluginServiceClient); ok {
		return gatewaydPlugin, nil
	}

	return nil, gerr.ErrPluginNotReady
}
