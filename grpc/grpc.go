package grpc

import (
	"context"
	"time"

	"google.golang.org/grpc/metadata"

	"github.com/HayoVanLoon/go-netcontext"
)

// CopyDeadline searches for the deadline in the metadata and returns an
// updated context with a cancellation function. If the headers do not include
// the deadline value, the context is returned unchanged and the cancellation
// function will be nil.
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
