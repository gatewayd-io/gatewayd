//go:build !embed_plugin_template

package plugin

import (
	"embed"
)

var (
	pluginTemplate        embed.FS
	pluginTemplateRootDir = ".template"
)

func IsPluginTemplateEmbedded() bool {
	return false
}
