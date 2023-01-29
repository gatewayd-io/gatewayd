package plugin

import (
	"net"

	sdkPlugin "github.com/gatewayd-io/gatewayd-plugin-sdk/plugin"
	v1 "github.com/gatewayd-io/gatewayd-plugin-sdk/plugin/v1"
	gerr "github.com/gatewayd-io/gatewayd/errors"
)

type Plugin sdkPlugin.Plugin

type IPlugin interface {
	Start() (net.Addr, error)
	Stop()
	Dispense() (v1.GatewayDPluginServiceClient, *gerr.GatewayDError)
}

var _ IPlugin = &Plugin{}

// Start starts the plugin.
func (p *Plugin) Start() (net.Addr, error) {
	var addr net.Addr
	var err error
	if addr, err = p.Client.Start(); err != nil {
		return nil, gerr.ErrFailedToStartPlugin.Wrap(err)
	}
	return addr, nil
}

// Stop kills the plugin.
func (p *Plugin) Stop() {
	p.Client.Kill()
}

// Dispense returns the plugin client.
func (p *Plugin) Dispense() (v1.GatewayDPluginServiceClient, *gerr.GatewayDError) {
	rpcClient, err := p.Client.Client()
	if err != nil {
		return nil, gerr.ErrFailedToGetRPCClient.Wrap(err)
	}

	raw, err := rpcClient.Dispense(p.ID.Name)
	if err != nil {
		return nil, gerr.ErrFailedToDispensePlugin.Wrap(err)
	}

	if gatewaydPlugin, ok := raw.(v1.GatewayDPluginServiceClient); ok {
		return gatewaydPlugin, nil
	}

	return nil, gerr.ErrPluginNotReady
}
