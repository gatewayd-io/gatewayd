package plugin

import (
	"context"

	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/structpb"
)

type (
	// Priority is the priority of a hook.
	// Smaller values are executed first (higher priority).
	Priority uint
	Method   func(
		context.Context, *structpb.Struct, ...grpc.CallOption) (*structpb.Struct, error)
)
