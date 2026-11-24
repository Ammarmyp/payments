package httpx_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Ammarmyp/payments/internal/platform/apperr"
	"github.com/Ammarmyp/payments/internal/platform/httpx"
	"github.com/Ammarmyp/payments/internal/platform/logging"
)

func TestRecoverRespondsInternalAndLogsErrorWithRequestID(t *testing.T) {
	var buf bytes.Buffer
	log := mustLogger(t, &buf)
	prev := slog.Default()
	slog.SetDefault(log)
	t.Cleanup(func() { slog.SetDefault(prev) })

	h := httpx.RequestID(httpx.Recover(log)(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		panic("boom")
	})))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set(httpx.RequestIDHeader, "panic-req-1")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	assertErrorResponse(t, rec, http.StatusInternalServerError, "internal", "internal server error")
	if strings.Contains(rec.Body.String(), "boom") {
		t.Fatalf("body leaked panic: %s", rec.Body.String())
	}
	if got := rec.Header().Get(httpx.RequestIDHeader); got != "panic-req-1" {
		t.Fatalf("response %s = %q, want panic-req-1", httpx.RequestIDHeader, got)
	}

	line := findLogLine(t, buf.Bytes(), func(m map[string]any) bool {
		return m["level"] == "ERROR" && m["msg"] == "panic recovered"
	})
	if line["request_id"] != "panic-req-1" {
		t.Fatalf("request_id = %v, want panic-req-1\nlogs=%s", line["request_id"], buf.String())
	}
	if !strings.Contains(fmt.Sprint(line["panic"]), "boom") {
		t.Fatalf("panic attr missing boom: %#v", line["panic"])
	}
	if !strings.Contains(fmt.Sprint(line["stack"]), "runtime/debug.Stack") {
		t.Fatalf("stack attr missing debug.Stack frame: %#v", line["stack"])
	}
}

func TestRecoverRethrowsErrAbortHandler(t *testing.T) {
	var buf bytes.Buffer
	log := mustLogger(t, &buf)

	h := httpx.Recover(log)(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		panic(http.ErrAbortHandler)
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()

	defer func() {
		v := recover()
		if v != http.ErrAbortHandler {
			t.Fatalf("recover = %v, want http.ErrAbortHandler", v)
		}
		if buf.Len() != 0 {
			t.Fatalf("expected no log for deliberate abort, got %s", buf.String())
		}
		if rec.Body.Len() != 0 {
			t.Fatalf("expected no body for deliberate abort, got %s", rec.Body.String())
		}
	}()

	h.ServeHTTP(rec, req)
	t.Fatal("expected panic(http.ErrAbortHandler)")
}

func TestRecoverMintsRequestIDWhenInboundMissing(t *testing.T) {
	var buf bytes.Buffer
	log := mustLogger(t, &buf)
	prev := slog.Default()
	slog.SetDefault(log)
	t.Cleanup(func() { slog.SetDefault(prev) })

	h := httpx.RequestID(httpx.Recover(log)(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		panic("boom")
	})))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	assertErrorResponse(t, rec, http.StatusInternalServerError, "internal", "internal server error")
	id := rec.Header().Get(httpx.RequestIDHeader)
	if id == "" {
		t.Fatal("expected minted request id")
	}
	line := findLogLine(t, buf.Bytes(), func(m map[string]any) bool {
		return m["level"] == "ERROR" && m["msg"] == "panic recovered"
	})
	if line["request_id"] != id {
		t.Fatalf("request_id = %v, want %s", line["request_id"], id)
	}
}

func TestErrorLogsServerFailuresOnly(t *testing.T) {
	tests := []struct {
		name    string
		err     error
		wantLog bool
		code    string
	}{
		{
			name:    "client_not_found",
			err:     apperr.New(apperr.NotFound, "missing"),
			wantLog: false,
			code:    "not_found",
		},
		{
			name:    "app_internal",
			err:     apperr.New(apperr.Internal, "storage unavailable"),
			wantLog: true,
			code:    "internal",
		},
		{
			name:    "unknown",
			err:     errString("sql: connection refused"),
			wantLog: true,
			code:    "internal",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			log := mustLogger(t, &buf)
			prev := slog.Default()
			slog.SetDefault(log)
			t.Cleanup(func() { slog.SetDefault(prev) })

			rec := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodGet, "/", nil)
			req = req.WithContext(logging.With(req.Context(), slog.String("request_id", "log-req-1")))

			httpx.Error(rec, req, tt.err)

			if !tt.wantLog {
				if buf.Len() != 0 {
					t.Fatalf("expected no log for client error, got %s", buf.String())
				}
				return
			}
			line := decodeLogLine(t, buf.Bytes())
			if line["level"] != "ERROR" {
				t.Fatalf("level = %v, want ERROR", line["level"])
			}
			if line["code"] != tt.code {
				t.Fatalf("code = %v, want %s", line["code"], tt.code)
			}
			if line["request_id"] != "log-req-1" {
				t.Fatalf("request_id = %v, want log-req-1", line["request_id"])
			}
		})
	}
}

type errString string

func (e errString) Error() string { return string(e) }

func mustLogger(t *testing.T, buf *bytes.Buffer) *slog.Logger {
	t.Helper()
	log, err := logging.New(logging.Options{
		Writer:  buf,
		Service: "wallet",
		Version: "dev",
		Level:   slog.LevelInfo,
		Format:  logging.FormatJSON,
	})
	if err != nil {
		t.Fatalf("logging.New: %v", err)
	}
	return log
}

func decodeLogLine(t *testing.T, raw []byte) map[string]any {
	t.Helper()
	line := bytes.TrimSpace(raw)
	if i := bytes.IndexByte(line, '\n'); i >= 0 {
		line = line[:i]
	}
	var got map[string]any
	if err := json.Unmarshal(line, &got); err != nil {
		t.Fatalf("decode log: %v\n%s", err, raw)
	}
	return got
}

func findLogLine(t *testing.T, raw []byte, match func(map[string]any) bool) map[string]any {
	t.Helper()
	for _, line := range bytes.Split(bytes.TrimSpace(raw), []byte("\n")) {
		if len(bytes.TrimSpace(line)) == 0 {
			continue
		}
		var got map[string]any
		if err := json.Unmarshal(line, &got); err != nil {
			t.Fatalf("decode log: %v\n%s", err, line)
		}
		if match(got) {
			return got
		}
	}
	t.Fatalf("no matching log line in:\n%s", raw)
	return nil
}
