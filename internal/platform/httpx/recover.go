package httpx

import (
	"fmt"
	"log/slog"
	"net/http"
)

// Recover turns panics into an internal JSON error. Place RequestID outside
// so recovery logs include request_id.
func Recover(log *slog.Logger, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			v := recover()
			if v == nil {
				return
			}
			if log != nil {
				log.ErrorContext(r.Context(), "panic", "panic", fmt.Sprint(v))
			}
			Error(w, r, errPanic)
		}()
		next.ServeHTTP(w, r)
	})
}

var errPanic = fmt.Errorf("panic")
