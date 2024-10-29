package grpc

import (
	"context"
	"github.com/HayoVanLoon/go-netcontext"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

// UnaryServerIntercept extracts configured values from the incoming metadata
// and stores them in the context. Sets a deadline (and handles its
// cancellation) when one is found. Does not process outgoing metadata.
func UnaryServerIntercept(ctx context.Context, r any, _ *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
	ctx = ExtractMetadata(ctx)
	ctx, cancel := CopyDeadline(ctx)
	if cancel != nil {
		defer cancel()
	}
	return handler(ctx, r)
}

// ExtractMetadata extracts configured values from the metadata and stores them
// in the returned context.
func ExtractMetadata(ctx context.Context) context.Context {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return ctx
	}
	for _, e := range netcontext.Entries() {
		vs := md.Get(metadataKey(e))
		if len(vs) == 0 {
			vs = md.Get(metadataKey(e))
		}
		if len(vs) > 0 {
			var a any
			if err := e.Unmarshal(vs[0], &a); err != nil {
				netcontext.Log("error parsing %q: %s", e.StringKey(), err.Error())
				continue
			}
			ctx = context.WithValue(ctx, e.CtxKey(), a)
		}
	}
	return ctx
}
