package middleware

import (
	"context"
	"net/http"
)

type httpRequestKey struct{}
type responseWriterKey struct{}

// InjectRawHTTP saves *http.Request and http.ResponseWriter in context
// so huma handlers can access them for rate limiting, cookie setting, etc.
func InjectRawHTTP(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := context.WithValue(r.Context(), httpRequestKey{}, r)
		ctx = context.WithValue(ctx, responseWriterKey{}, w)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func GetHTTPRequest(ctx context.Context) *http.Request {
	r, _ := ctx.Value(httpRequestKey{}).(*http.Request)
	return r
}

func GetResponseWriter(ctx context.Context) http.ResponseWriter {
	w, _ := ctx.Value(responseWriterKey{}).(http.ResponseWriter)
	return w
}
