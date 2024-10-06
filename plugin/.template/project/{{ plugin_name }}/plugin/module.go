package plugin

import (
	sdkConfig "github.com/gatewayd-io/gatewayd-plugin-sdk/config"
	v1 "github.com/gatewayd-io/gatewayd-plugin-sdk/plugin/v1"
	goplugin "github.com/hashicorp/go-plugin"
)

var (
	PluginID = v1.PluginID{
		Name:      "{{ plugin_name }}",
		Version:   "{{ version }}",
		RemoteUrl: "{{ remote_url }}",
	}
	PluginMap = map[string]goplugin.Plugin{
		"{{  plugin_name }}": &{{ pascal_case_plugin_name }}{},
	}
	// TODO: Handle this in a better way
	// https://github.com/gatewayd-io/gatewayd-plugin-sdk/issues/3
	PluginConfig = map[string]any{
		"id": map[string]any{
			"{{ plugin_name }}": PluginID.GetName(),
			"version":    PluginID.GetVersion(),
			"remoteUrl":  PluginID.GetRemoteUrl(),
		},
		"description": "Template plugin",
		"authors": []any{
			{% if authors %}"{{ authors|join:'", "'|safe }}",{% endif %}
		},
		"license":    "Apache 2.0",
		"projectUrl": "{{ remote_url }}",
		// Compile-time configuration
		"config": map[string]any{
			"metricsEnabled": sdkConfig.GetEnv("METRICS_ENABLED", "true"),
			"metricsUnixDomainSocket": sdkConfig.GetEnv(
				"METRICS_UNIX_DOMAIN_SOCKET", "/tmp/{{  plugin_name }}.sock"),
			"metricsEndpoint": sdkConfig.GetEnv("METRICS_ENDPOINT", "/metrics"),
		},
		"hooks": []any{
			// Converting HookName to int32 is required because the plugin
			// framework doesn't support enums.	See:
			// https://github.com/gatewayd-io/gatewayd-plugin-sdk/issues/3
			int32(v1.HookName_HOOK_NAME_ON_CONFIG_LOADED),
			int32(v1.HookName_HOOK_NAME_ON_NEW_LOGGER),
			int32(v1.HookName_HOOK_NAME_ON_NEW_POOL),
			int32(v1.HookName_HOOK_NAME_ON_NEW_CLIENT),
			int32(v1.HookName_HOOK_NAME_ON_NEW_PROXY),
			int32(v1.HookName_HOOK_NAME_ON_NEW_SERVER),
			int32(v1.HookName_HOOK_NAME_ON_SIGNAL),
			int32(v1.HookName_HOOK_NAME_ON_RUN),
			int32(v1.HookName_HOOK_NAME_ON_BOOTING),
			int32(v1.HookName_HOOK_NAME_ON_BOOTED),
			int32(v1.HookName_HOOK_NAME_ON_OPENING),
			int32(v1.HookName_HOOK_NAME_ON_OPENED),
			int32(v1.HookName_HOOK_NAME_ON_CLOSING),
			int32(v1.HookName_HOOK_NAME_ON_CLOSED),
			int32(v1.HookName_HOOK_NAME_ON_TRAFFIC),
			int32(v1.HookName_HOOK_NAME_ON_TRAFFIC_FROM_CLIENT),
			int32(v1.HookName_HOOK_NAME_ON_TRAFFIC_TO_SERVER),
			int32(v1.HookName_HOOK_NAME_ON_TRAFFIC_FROM_SERVER),
			int32(v1.HookName_HOOK_NAME_ON_TRAFFIC_TO_CLIENT),
			int32(v1.HookName_HOOK_NAME_ON_SHUTDOWN),
			int32(v1.HookName_HOOK_NAME_ON_TICK),
			// The following hook is invalid, and leads to an error in GatewayD,
			// but it'll be ignored. This is used for testing purposes. Feel free
			// to remove it in your plugin.
			int32(v1.HookName(1000)),
		},
		"tags":       []any{"template", "plugin"},
		"categories": []any{"template"},
	}
)
