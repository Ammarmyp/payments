package httpx

import (
	"log/slog"
	"net/http"
	"time"
)

// AccessLog logs method, path, status, duration, and response size after each
// request. Status >= 500 is logged at error.
func AccessLog(logger *slog.Logger) Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			rec := &statusRecorder{ResponseWriter: w}
			next.ServeHTTP(rec, r)

			if rec.status == 0 {
				rec.status = http.StatusOK
			}
			if logger == nil {
				return
			}

			level := slog.LevelInfo
			if rec.status >= http.StatusInternalServerError {
				level = slog.LevelError
			}
			logger.LogAttrs(r.Context(), level, "request",
				slog.String("method", r.Method),
				slog.String("path", r.URL.Path),
				slog.Int("status", rec.status),
				slog.Duration("duration", time.Since(start)),
				slog.Int("bytes", rec.bytes),
			)
		})
	}
}

type statusRecorder struct {
	http.ResponseWriter
	status int
	bytes  int
}

func (w *statusRecorder) WriteHeader(status int) {
	w.status = status
	w.ResponseWriter.WriteHeader(status)
}

func (w *statusRecorder) Write(p []byte) (int, error) {
	n, err := w.ResponseWriter.Write(p)
	w.bytes += n
	return n, err
}

func (w *statusRecorder) Unwrap() http.ResponseWriter {
	return w.ResponseWriter
}
