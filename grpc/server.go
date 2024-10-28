package grpc

import (
	"context"
	"google.golang.org/grpc"
)

func UnaryServerIntercept(ctx context.Context, r any, _ *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
	ctx = harvestMetadata(ctx)
	ctx, cancel := CopyDeadline(ctx)
	if cancel != nil {
		defer cancel()
	}
	return handler(ctx, r)
}
