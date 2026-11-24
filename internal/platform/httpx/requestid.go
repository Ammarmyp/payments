package httpx

import (
	"context"
	"log/slog"
	"net/http"

	"github.com/Ammarmyp/payments/internal/platform/logging"
	"github.com/google/uuid"
)

// RequestIDHeader is the request correlation header.
const RequestIDHeader = "X-Request-ID"

const maxRequestIDLen = 64

type requestIDKey struct{}

// RequestID ensures every request has an X-Request-ID on the response and in
// log context.
func RequestID(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id := r.Header.Get(RequestIDHeader)
		if !validRequestID(id) {
			id = uuid.NewString()
		}
		w.Header().Set(RequestIDHeader, id)
		ctx := context.WithValue(r.Context(), requestIDKey{}, id)
		ctx = logging.With(ctx, slog.String("request_id", id))
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// RequestIDFrom returns the request ID stored on ctx, if any.
func RequestIDFrom(ctx context.Context) string {
	id, _ := ctx.Value(requestIDKey{}).(string)
	return id
}

func validRequestID(id string) bool {
	if id == "" || len(id) > maxRequestIDLen {
		return false
	}
	for _, c := range id {
		switch {
		case c >= 'a' && c <= 'z',
			c >= 'A' && c <= 'Z',
			c >= '0' && c <= '9',
			c == '-', c == '_':
			continue
		default:
			return false
		}
	}
	return true
}
