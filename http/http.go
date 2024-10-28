package http

import (
	"context"
	"net/http"
	"time"

	"github.com/HayoVanLoon/go-netcontext"
)

// Extract extracts the values from the headers (or trailers) and returns a new
// context with the values found.
func Extract(ctx context.Context, h http.Header) context.Context {
	for _, e := range netcontext.Entries() {
		v := h.Get(headerKey(e))
		if v == "" {
			continue
		}
		var a any
		if err := e.Unmarshal(v, &a); err != nil {
			netcontext.Log("could not parse value for key %q: %v", e.StringKey(), v)
			continue
		}
		ctx = context.WithValue(ctx, e.CtxKey(), a)
	}
	return ctx
}

func CopyDeadline(ctx context.Context, h http.Header) (context.Context, context.CancelFunc) {
	e, ok := netcontext.Deadline()
	if !ok {
		return ctx, nil
	}
	s := h.Get(headerKey(e))
	if s == "" {
		return ctx, nil
	}
	var t time.Time
	if err := e.Unmarshal(s, &t); err != nil {
		netcontext.Log("error parsing deadline header: %s", err.Error())
		return ctx, nil
	}
	return context.WithDeadline(ctx, t)
}

func headerKey(e netcontext.Entry) string {
	return netcontext.HTTPHeaderPrefix() + e.StringKey()
}
