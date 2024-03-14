package api

import "context"

type IAPIServer interface {
	Start()
	Shutdown(ctx context.Context)
}
