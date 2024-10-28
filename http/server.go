package http

import (
	"net/http"
)

func WrapHandler(h http.Handler) http.Handler {
	return WrapHandlerFunc(h.ServeHTTP)
}

func WrapHandlerFunc(h http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := Extract(r.Context(), r.Header)
		ctx, cancel := CopyDeadline(ctx, r.Header)
		if cancel != nil {
			defer cancel()
		}
		r = r.WithContext(ctx)
		h(w, r)
	}
}
