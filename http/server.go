package http

import (
	"net/http"
)

// WrapHandler wraps an http.Handler, adding configured values to the incoming
// context. Sets a deadline (and handles its cancellation) when one is found.
// Does not process outgoing response headers.
func WrapHandler(h http.Handler) http.Handler {
	return WrapHandlerFunc(h.ServeHTTP)
}

// WrapHandlerFunc wraps an http.HandlerFunc, adding configured values to the
// incoming context. Sets a deadline (and handles its cancellation) when one is
// found. Does not process outgoing response headers.
func WrapHandlerFunc(h http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := ExtractWithDeadline(r.Context(), r.Header)
		if cancel != nil {
			defer cancel()
		}
		r = r.WithContext(ctx)
		h(w, r)
	}
}
