package plugin

import (
	"context"

	v1 "github.com/gatewayd-io/gatewayd-plugin-sdk/plugin/v1"
	"github.com/hashicorp/go-hclog"
	goplugin "github.com/hashicorp/go-plugin"
	"google.golang.org/grpc"
)

type Plugin struct {
	goplugin.GRPCPlugin
	v1.GatewayDPluginServiceServer
	Logger hclog.Logger
}

type {{ pascal_case_plugin_name }} struct {
	goplugin.NetRPCUnsupportedPlugin
	Impl Plugin
}

// GRPCServer registers the plugin with the gRPC server.
func (p *{{ pascal_case_plugin_name }}) GRPCServer(b *goplugin.GRPCBroker, s *grpc.Server) error {
	v1.RegisterGatewayDPluginServiceServer(s, &p.Impl)
	return nil
}

// GRPCClient returns the plugin client.
func (p *{{ pascal_case_plugin_name }}) GRPCClient(ctx context.Context, b *goplugin.GRPCBroker, c *grpc.ClientConn) (interface{}, error) {
	return v1.NewGatewayDPluginServiceClient(c), nil
}

// New{{ pascal_case_plugin_name }} returns a new instance of the TestPlugin.
func New{{ pascal_case_plugin_name }}(impl Plugin) *{{ pascal_case_plugin_name }} {
	return &{{ pascal_case_plugin_name }}{
		NetRPCUnsupportedPlugin: goplugin.NetRPCUnsupportedPlugin{},
		Impl:                    impl,
	}
}

// GetPluginConfig returns the plugin config. This is called by GatewayD
// when the plugin is loaded. The plugin config is used to configure the
// plugin.
func (p *Plugin) GetPluginConfig(
	ctx context.Context, _ *v1.Struct) (*v1.Struct, error) {
	GetPluginConfig.Inc()

	return v1.NewStruct(PluginConfig)
}

// OnConfigLoaded is called when the global config is loaded by GatewayD.
// This can be used to modify the global config. Note that the plugin config
// cannot be modified via plugins.
func (p *Plugin) OnConfigLoaded(ctx context.Context, req *v1.Struct) (*v1.Struct, error) {
	OnConfigLoaded.Inc()
	// Example req:
	// {
	// 	"clients.default.address": "localhost:5432",
	// 	"clients.default.network": "tcp",
	// 	"clients.default.receiveBufferSize": 16777216,
	// 	"clients.default.receiveChunkSize": 4096,
	// 	"clients.default.receiveDeadline": "0ms",
	// 	"clients.default.sendDeadline": "0ms",
	// 	"clients.default.tcpKeepAlive": false,
	// 	"clients.default.tcpKeepAlivePeriod": "30s",
	// 	"loggers.default.compress": true,
	// 	"loggers.default.consoleTimeFormat": "RFC3339",
	// 	"loggers.default.fileName": "gatewayd.log",
	// 	"loggers.default.level": "debug",
	// 	"loggers.default.localTime": false,
	// 	"loggers.default.maxAge": 30,
	// 	"loggers.default.maxBackups": 5,
	// 	"loggers.default.maxSize": 500,
	// 	"loggers.default.noColor": false,
	// 	"loggers.default.output": ["console", "file"],
	// 	"loggers.default.rsyslogAddress": "localhost:514",
	// 	"loggers.default.rsyslogNetwork": "tcp",
	// 	"loggers.default.syslogPriority": "info",
	// 	"loggers.default.timeFormat": "unix",
	// 	"metrics.default.address": "localhost:9090  ",
	// 	"metrics.default.enabled": true,
	// 	"metrics.default.path": "/metrics",
	// 	"pools.default.size": 10,
	// 	"proxies.default.elastic": false,
	// 	"proxies.default.healthCheckPeriod": "60s",
	// 	"proxies.default.reuseElasticClients": false,
	// 	"proxy.default.elastic": false,
	// 	"proxy.default.healthCheckPeriod": "1m0s",
	// 	"proxy.default.reuseElasticClients": false,
	// 	"servers.default.address": "0.0.0.0:15432",
	// 	"servers.default.enableTicker": false,
	// 	"servers.default.hardLimit": 0,
	// 	"servers.default.loadBalancer": "roundrobin",
	// 	"servers.default.lockOSThread": false,
	// 	"servers.default.multiCore": true,
	// 	"servers.default.network": "tcp",
	// 	"servers.default.readBufferCap": 16777216,
	// 	"servers.default.reuseAddress": true,
	// 	"servers.default.reusePort": true,
	// 	"servers.default.socketRecvBuffer": 16777216,
	// 	"servers.default.socketSendBuffer": 16777216,
	// 	"servers.default.softLimit": 0,
	// 	"servers.default.tcpKeepAlive": "3s",
	// 	"servers.default.tcpNoDelay": true,
	// 	"servers.default.tickInterval": "5s",
	// 	"servers.default.writeBufferCap": 16777216
	// }
	p.Logger.Debug("OnConfigLoaded", "req", req) // See what's in request from GatewayD.

	if req.Fields == nil {
		req.Fields = make(map[string]*v1.Value)
	}

	req.Fields["loggers.default.level"] = v1.NewStringValue("debug")
	req.Fields["loggers.default.noColor"] = v1.NewBoolValue(false)

	return req, nil
}

// OnNewLogger is called when a new logger is created by GatewayD.
// This is a notification and the plugin cannot modify the logger.
func (p *Plugin) OnNewLogger(ctx context.Context, req *v1.Struct) (*v1.Struct, error) {
	OnNewLogger.Inc()
	// Example req:
	// {
	// 	"default": {
	// 		"compress": true,
	// 		"consoleTimeFormat": "RFC3339",
	// 		"fileName": "gatewayd.log",
	// 		"level": "debug",
	// 		"localTime": false,
	// 		"maxAge": 30,
	// 		"maxBackups": 5,
	// 		"maxSize": 500,
	// 		"noColor": false,
	// 		"output": ["console", "file"],
	// 		"rsyslogAddress": "localhost:514",
	// 		"rsyslogNetwork": "tcp",
	// 		"syslogPriority": "info",
	// 		"timeFormat": "unix"
	// 	}
	// }
	p.Logger.Debug("OnNewLogger", "req", req)

	return req, nil
}

// OnNewPool is called when a new pool is created by GatewayD.
// This is a notification and the plugin cannot modify the pool.
func (p *Plugin) OnNewPool(ctx context.Context, req *v1.Struct) (*v1.Struct, error) {
	OnNewPool.Inc()
	// Example req:
	// {"{{ plugin_name }}":"default","size":10}
	p.Logger.Debug("OnNewPool", "req", req)

	return req, nil
}

// OnNewClient is called when a new client is created by GatewayD.
// This is a notification and the plugin cannot modify the client.
func (p *Plugin) OnNewClient(ctx context.Context, req *v1.Struct) (*v1.Struct, error) {
	OnNewClient.Inc()
	// Example req:
	// {
	//     "address": "127.0.0.1:5432",
	//     "id": "8ee872e660e998b35e7f8cfeb2b8b426a315d4f107d79a313ff4ef3c1d2ee4bc",
	//     "network": "tcp",
	//     "receiveBufferSize": 16777216,
	//     "receiveChunkSize": 4096,
	//     "receiveDeadline": "0s",
	//     "sendDeadline": "0s",
	//     "tcpKeepAlive": false,
	//     "tcpKeepAlivePeriod": "30s"
	// }
	p.Logger.Debug("OnNewClient", "req", req)

	return req, nil
}

// OnNewProxy is called when a new proxy is created by GatewayD.
// This is a notification and the plugin cannot modify the proxy.
func (p *Plugin) OnNewProxy(ctx context.Context, req *v1.Struct) (*v1.Struct, error) {
	OnNewProxy.Inc()
	// Example req:
	// {
	//     "default": {
	//         "elastic": false,
	//         "healthCheckPeriod": "1m0s",
	//         "reuseElasticClients": false
	//     }
	// }
	p.Logger.Debug("OnNewProxy", "req", req)

	return req, nil
}

// OnNewServer is called when a new server is created by GatewayD.
// This is a notification and the plugin cannot modify the server.
func (p *Plugin) OnNewServer(ctx context.Context, req *v1.Struct) (*v1.Struct, error) {
	OnNewServer.Inc()
	// Example req:
	// {
	//     "default": {
	//         "address": "0.0.0.0:15432",
	//         "enableTicker": false,
	//         "hardLimit": 0,
	//         "loadBalancer": "roundrobin",
	//         "lockOSThread": false,
	//         "multiCore": true,
	//         "network": "tcp",
	//         "readBufferCap": 16777216,
	//         "reuseAddress": true,
	//         "reusePort": true,
	//         "socketRecvBuffer": 16777216,
	//         "socketSendBuffer": 16777216,
	//         "softLimit": 0,
	//         "tcpKeepAlive": "3s",
	//         "tcpNoDelay": true,
	//         "tickInterval": "5s",
	//         "writeBufferCap": 16777216
	//     }
	// }
	p.Logger.Debug("OnNewServer", "req", req)

	return req, nil
}

// OnSignal is called when a signal (for example, SIGKILL) is received by GatewayD.
// This is a notification and the plugin cannot modify the signal.
func (p *Plugin) OnSignal(ctx context.Context, req *v1.Struct) (*v1.Struct, error) {
	OnSignal.Inc()
	// Example req:
	// {"signal":"interrupt"}
	p.Logger.Debug("OnSignal", "req", req)

	return req, nil
}

// OnRun is called when GatewayD is started.
func (p *Plugin) OnRun(ctx context.Context, req *v1.Struct) (*v1.Struct, error) {
	OnRun.Inc()
	// Example req:
	// {"address":"0.0.0.0:15432"}
	p.Logger.Debug("OnRun", "req", req)

	return req, nil
}

// OnBooting is called when GatewayD is booting.
func (p *Plugin) OnBooting(ctx context.Context, req *v1.Struct) (*v1.Struct, error) {
	OnBooting.Inc()
	// Example req:
	// {"status":"1"}
	p.Logger.Debug("OnBooting", "req", req)

	return req, nil
}

// OnBooted is called when GatewayD is booted.
func (p *Plugin) OnBooted(ctx context.Context, req *v1.Struct) (*v1.Struct, error) {
	OnBooted.Inc()
	// Example req:
	// {"status":"0"}
	p.Logger.Debug("OnBooted", "req", req)

	return req, nil
}

// OnOpening is called when a new client connection is being opened.
func (p *Plugin) OnOpening(ctx context.Context, req *v1.Struct) (*v1.Struct, error) {
	OnOpening.Inc()
	// Example req:
	// {
	//     "client": {
	//         "local": "0.0.0.0:15432",
	//         "remote": "127.0.0.1:45612"
	//     }
	// }
	p.Logger.Debug("OnOpening", "req", req)

	return req, nil
}

// OnOpened is called when a new client connection is opened.
func (p *Plugin) OnOpened(ctx context.Context, req *v1.Struct) (*v1.Struct, error) {
	OnOpened.Inc()
	// Example req:
	// {
	//     "client": {
	//         "local": "0.0.0.0:15432",
	//         "remote": "127.0.0.1:45612"
	//     }
	// }
	p.Logger.Debug("OnOpened", "req", req)

	return req, nil
}

// OnClosing is called when a client connection is being closed.
func (p *Plugin) OnClosing(ctx context.Context, req *v1.Struct) (*v1.Struct, error) {
	OnClosing.Inc()
	// Example req:
	// {
	//     "client": {
	//         "local": "0.0.0.0:15432",
	//         "remote": "127.0.0.1:45612"
	//     },
	//     "error": ""
	// }
	p.Logger.Debug("OnClosing", "req", req)

	return req, nil
}

// OnClosed is called when a client connection is closed.
func (p *Plugin) OnClosed(ctx context.Context, req *v1.Struct) (*v1.Struct, error) {
	OnClosed.Inc()
	// Example req:
	// {
	//     "client": {
	//         "local": "0.0.0.0:15432",
	//         "remote": "127.0.0.1:45612"
	//     },
	//     "error": ""
	// }
	p.Logger.Debug("OnClosed", "req", req)

	return req, nil
}

// OnTraffic is called when a request is being received by GatewayD from the client.
// This is a notification and the plugin cannot modify the request at this point.
func (p *Plugin) OnTraffic(ctx context.Context, req *v1.Struct) (*v1.Struct, error) {
	OnTraffic.Inc()
	// Example req:
	// {
	//     "client": {
	//         "local": "0.0.0.0:15432",
	//         "remote": "127.0.0.1:45612"
	//     }
	// }
	p.Logger.Debug("OnTraffic", "req", req)

	return req, nil
}

// OnShutdown is called when GatewayD is shutting down.
func (p *Plugin) OnShutdown(ctx context.Context, req *v1.Struct) (*v1.Struct, error) {
	OnShutdown.Inc()
	// Example req:
	// {"connections":10}
	p.Logger.Debug("OnShutdown", "req", req)

	return req, nil
}

// OnTick is called when GatewayD is ticking (if enabled).
func (p *Plugin) OnTick(ctx context.Context, req *v1.Struct) (*v1.Struct, error) {
	OnTick.Inc()
	// Example req:
	// {"connections":10}
	p.Logger.Debug("OnTick", "req", req)

	return req, nil
}

// OnTrafficFromClient is called when a request is received by GatewayD from the client.
// This can be used to modify the request or terminate the connection by returning an error
// or a response.
func (p *Plugin) OnTrafficFromClient(ctx context.Context, req *v1.Struct) (*v1.Struct, error) {
	OnTrafficFromClient.Inc()
	// Example req:
	// {
	//     "client": {
	//         "local": "0.0.0.0:15432",
	//         "remote": "127.0.0.1:45612"
	//     },
	//     "error": "",
	//     "query": "eyJUeXBlIjoiUXVlcnkiLCJTdHJpbmciOiJzZWxlY3QgMTsifQ==",
	//     "request": "UQAAAA5zZWxlY3QgMTsA",
	//     "server": {
	//         "local": "127.0.0.1:60386",
	//         "remote": "127.0.0.1:5432"
	//     }
	// }
	p.Logger.Debug("OnTrafficFromClient", "req", req)

	request := req.Fields["request"].GetBytesValue()
	p.Logger.Debug("OnTrafficFromClient", "request", string(request))

	return req, nil
}

// OnTrafficToServer is called when a request is sent by GatewayD to the server.
// This can be used to modify the request or terminate the connection by returning an error
// or a response while also sending the request to the server.
func (p *Plugin) OnTrafficToServer(ctx context.Context, req *v1.Struct) (*v1.Struct, error) {
	OnTrafficToServer.Inc()
	// Example req:
	// {
	//     "client": {
	//         "local": "0.0.0.0:15432",
	//         "remote": "127.0.0.1:45612"
	//     },
	//     "error": "",
	//     "request": "UQAAAA5zZWxlY3QgMTsA",
	//     "server": {
	//         "local": "127.0.0.1:60386",
	//         "remote": "127.0.0.1:5432"
	//     }
	// }
	p.Logger.Debug("OnTrafficToServer", "req", req)

	request := req.Fields["request"].GetBytesValue()
	p.Logger.Debug("OnTrafficToServer", "request", string(request))

	return req, nil
}

// OnTrafficFromServer is called when a response is received by GatewayD from the server.
// This can be used to modify the response or terminate the connection by returning an error
// or a response.
func (p *Plugin) OnTrafficFromServer(
	ctx context.Context, resp *v1.Struct) (*v1.Struct, error) {
	OnTrafficFromServer.Inc()
	// Example resp:
	// {
	//     "client": {
	//         "local": "0.0.0.0:15432",
	//         "remote": "127.0.0.1:45612"
	//     },
	//     "commandComplete": "eyJUeXBlIjoiQ29tbWFuZENvbXBsZXRlIiwiQ29tbWFuZFRhZyI6IlNFTEVDVCAxIn0=",
	//     "dataRow": ["eyJUeXBlIjoiRGF0YVJvdyIsIlZhbHVlcyI6W3sidGV4dCI6IjEifV19"],
	//     "error": "",
	//     "readyForQuery": "eyJUeXBlIjoiUmVhZHlGb3JRdWVyeSIsIlR4U3RhdHVzIjoiSSJ9",
	//     "request": "UQAAAA5zZWxlY3QgMTsA",
	//     "response": "VAAAACEAAT9jb2x1bW4/AAAAAAAAAAAAABcABP////8AAEQAAAALAAEAAAABMUMAAAANU0VMRUNUIDEAWgAAAAVJ",
	//     "rowDescription": "eyJUeXBlIjoiUm93RGVzY3JpcHRpb24iLCJGaWVsZHMiOlt7Ik5hbWUiOiI/Y29sdW1uPyIsIlRhYmxlT0lEIjowLCJUYWJsZUF0dHJpYnV0ZU51bWJlciI6MCwiRGF0YVR5cGVPSUQiOjIzLCJEYXRhVHlwZVNpemUiOjQsIlR5cGVNb2RpZmllciI6LTEsIkZvcm1hdCI6MH1dfQ==",
	//     "server": {
	//         "local": "127.0.0.1:60386",
	//         "remote": "127.0.0.1:5432"
	//     }
	// }
	p.Logger.Debug("OnTrafficFromServer", "resp", resp)

	response := resp.Fields["response"].GetBytesValue()
	p.Logger.Debug("OnEgressTraffic", "response", string(response))

	return resp, nil
}

// OnTrafficToClient is called when a response is sent by GatewayD to the client.
// This can be used to modify the response or terminate the connection by returning an error
// or a response.
func (p *Plugin) OnTrafficToClient(
	ctx context.Context, resp *v1.Struct) (*v1.Struct, error) {
	OnTrafficToClient.Inc()
	// Example resp:
	// {
	//     "client": {
	//         "local": "0.0.0.0:15432",
	//         "remote": "127.0.0.1:45612"
	//     },
	//     "error": "",
	//     "request": "UQAAAA5zZWxlY3QgMTsA",
	//     "response": "VAAAACEAAT9jb2x1bW4/AAAAAAAAAAAAABcABP////8AAEQAAAALAAEAAAABMUMAAAANU0VMRUNUIDEAWgAAAAVJ",
	//     "server": {
	//         "local": "127.0.0.1:60386",
	//         "remote": "127.0.0.1:5432"
	//     }
	// }
	p.Logger.Debug("OnTrafficToClient", "resp", resp)

	response := resp.Fields["response"].GetBytesValue()
	p.Logger.Debug("OnEgressTraffic", "response", string(response))

	return resp, nil
}
