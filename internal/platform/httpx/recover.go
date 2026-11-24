package httpx

import (
	"log/slog"
	"net/http"
	"runtime/debug"
)

// Recover turns a panic in a handler into a 500 JSON error, and logs it with
// its stack trace. Place RequestID outside so recovery logs include request_id.
// panic(http.ErrAbortHandler) is re-raised so net/http can abort the handler.
func Recover(logger *slog.Logger) Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				v := recover()
				if v == nil {
					return
				}
				if v == http.ErrAbortHandler {
					panic(v) // a deliberate abort: let net/http handle it
				}
				if logger != nil {
					logger.ErrorContext(r.Context(), "panic recovered",
						"panic", v,
						"stack", string(debug.Stack()),
					)
				}
				writeError(w, r, http.StatusInternalServerError, errorDetail{
					Code:    "internal",
					Message: "internal server error",
				})
			}()
			next.ServeHTTP(w, r)
		})
	}
}
