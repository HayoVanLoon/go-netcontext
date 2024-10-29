package http

import (
	"context"
	"net/http"
	"time"

	"github.com/HayoVanLoon/go-netcontext"
)

// Extract extracts the values from the headers (or trailers) and returns a new
// context with the values found. This method will never set a deadline on the
// context.
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

// ExtractWithDeadline works as Extract, but will set a deadline if one is
// found in the headers. In that case (only), the cancellation function will be
// nil.
func ExtractWithDeadline(ctx context.Context, h http.Header) (context.Context, context.CancelFunc) {
	ctx = Extract(ctx, h)
	return CopyDeadline(ctx, h)
}

// CopyDeadline searches for the deadline in the headers and returns an updated
// context with a cancellation function. If the headers do not include the
// deadline value, the context is returned unchanged and the cancellation
// function will be nil.
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
