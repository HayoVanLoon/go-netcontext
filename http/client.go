package http

import (
	"context"
	"net/http"

	"github.com/HayoVanLoon/go-netcontext"
)

func Client() *http.Client {
	return WrapClient(&http.Client{})
}

func WrapClient(c *http.Client) *http.Client {
	base := c.Transport
	if base == nil {
		base = http.DefaultTransport
	}
	c.Transport = ContextRoundTripper{base: base}
	return c
}

type ContextRoundTripper struct {
	base http.RoundTripper
}

func (c ContextRoundTripper) RoundTrip(r *http.Request) (*http.Response, error) {
	for k, vs := range c.createHeaders(r.Context()) {
		for _, v := range vs {
			r.Header.Add(k, v)
		}
	}
	if e, ok := netcontext.Deadline(); ok {
		if t, ok := r.Context().Deadline(); ok {
			r.Header.Add(headerKey(e), e.ValueToString(t))
		}
	}
	return c.base.RoundTrip(r)
}

func (c ContextRoundTripper) createHeaders(ctx context.Context) http.Header {
	h := http.Header{}
	for _, e := range netcontext.Entries() {
		v := ctx.Value(e.CtxKey())
		if v != nil {
			h.Add(headerKey(e), e.ValueToString(v))
		}
	}
	return h
}
