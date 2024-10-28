package grpc

import (
	"context"

	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"

	"github.com/HayoVanLoon/go-netcontext"
)

func UnaryClientIntercept(ctx context.Context, method string, req, reply any, cc *grpc.ClientConn, invoker grpc.UnaryInvoker, opts ...grpc.CallOption) error {
	if kvs := getKeyValues(ctx); kvs != nil {
		ctx = metadata.AppendToOutgoingContext(ctx, kvs...)
	}
	return invoker(ctx, method, req, reply, cc, opts...)
}

func getKeyValues(ctx context.Context) []string {
	var kvs []string
	for _, e := range netcontext.Entries() {
		v := ctx.Value(e.CtxKey())
		if v != nil {
			kvs = append(kvs, metadataKey(e), e.ValueToString(v))
		}
	}
	if e, ok := netcontext.Deadline(); ok {
		if t, ok := ctx.Deadline(); ok {
			kvs = append(kvs, metadataKey(e), e.ValueToString(t))
		}
	}
	return kvs
}
