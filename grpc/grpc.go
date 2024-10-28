package grpc

import (
	"context"
	"time"

	"google.golang.org/grpc/metadata"

	"github.com/HayoVanLoon/go-netcontext"
)

func harvestMetadata(ctx context.Context) context.Context {
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

func CopyDeadline(ctx context.Context) (context.Context, context.CancelFunc) {
	e, ok := netcontext.Deadline()
	if !ok {
		return ctx, nil
	}
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return ctx, nil
	}
	vs := md.Get(metadataKey(e))
	if len(vs) == 0 {
		return ctx, nil
	}
	var t time.Time
	if err := e.Unmarshal(vs[0], &t); err != nil {
		netcontext.Log("error parsing deadline header: %s", err.Error())
		return ctx, nil
	}
	return context.WithDeadline(ctx, t)
}

func metadataKey(e netcontext.Entry) string {
	key := e.StringKey()
	return netcontext.GRPCMetadataPrefix() + key
}
