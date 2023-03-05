//go:build !embed_swagger

package api

import (
	"embed"
)

var swaggerUI embed.FS

func IsSwaggerEmbedded() bool {
	return false
}
