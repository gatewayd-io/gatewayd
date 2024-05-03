//go:build embed_plugin_template

package plugin

import "embed"

//go:embed .template/* .template/project/*/.*
var pluginTemplate embed.FS
var pluginTemplateRootDir = ".template"

func IsPluginTemplateEmbedded() bool {
	return true
}
