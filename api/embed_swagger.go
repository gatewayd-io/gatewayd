//go:build embed_swagger

package api

import "embed"

//go:embed v1/api.swagger.json
//go:embed v1/swagger-ui
var swaggerUI embed.FS

func IsSwaggerEmbedded() bool {
	return true
}
