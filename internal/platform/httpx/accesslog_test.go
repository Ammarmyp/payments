package httpx_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/Ammarmyp/payments/internal/platform/httpx"
)

func TestAccessLogRecordsRequestLineAtInfo(t *testing.T) {
	var buf bytes.Buffer
	log := mustLogger(t, &buf)

	h := httpx.AccessLog(log)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(2 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"ok":true}`))
	}))

	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	line := decodeAccessLog(t, buf.Bytes())
	if line["level"] != "INFO" {
		t.Fatalf("level = %v, want INFO", line["level"])
	}
	if line["msg"] != "request" {
		t.Fatalf("msg = %v, want request", line["msg"])
	}
	if line["method"] != http.MethodGet {
		t.Fatalf("method = %v, want GET", line["method"])
	}
	if line["path"] != "/healthz" {
		t.Fatalf("path = %v, want /healthz", line["path"])
	}
	if status, ok := line["status"].(float64); !ok || int(status) != http.StatusOK {
		t.Fatalf("status = %v, want 200", line["status"])
	}
	if bytesN, ok := line["bytes"].(float64); !ok || int(bytesN) != len(`{"ok":true}`) {
		t.Fatalf("bytes = %v, want %d", line["bytes"], len(`{"ok":true}`))
	}
	if _, ok := line["duration"]; !ok {
		t.Fatalf("missing duration: %#v", line)
	}
}

func TestAccessLogServerErrorsAtErrorLevel(t *testing.T) {
	var buf bytes.Buffer
	log := mustLogger(t, &buf)

	h := httpx.AccessLog(log)(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "nope", http.StatusInternalServerError)
	}))

	req := httptest.NewRequest(http.MethodPost, "/explode", nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	line := decodeAccessLog(t, buf.Bytes())
	if line["level"] != "ERROR" {
		t.Fatalf("level = %v, want ERROR\n%s", line["level"], buf.String())
	}
	if status, ok := line["status"].(float64); !ok || int(status) != http.StatusInternalServerError {
		t.Fatalf("status = %v, want 500", line["status"])
	}
	if line["method"] != http.MethodPost {
		t.Fatalf("method = %v, want POST", line["method"])
	}
	if line["path"] != "/explode" {
		t.Fatalf("path = %v, want /explode", line["path"])
	}
}

func TestAccessLogIncludesRequestIDFromOuterMiddleware(t *testing.T) {
	var buf bytes.Buffer
	log := mustLogger(t, &buf)

	h := httpx.Chain(
		http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusNoContent)
		}),
		httpx.RequestID,
		httpx.AccessLog(log),
	)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set(httpx.RequestIDHeader, "access-req-1")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	line := decodeAccessLog(t, buf.Bytes())
	if line["request_id"] != "access-req-1" {
		t.Fatalf("request_id = %v, want access-req-1\n%s", line["request_id"], buf.String())
	}
}

func decodeAccessLog(t *testing.T, raw []byte) map[string]any {
	t.Helper()
	var line map[string]any
	if err := json.Unmarshal(raw, &line); err != nil {
		t.Fatalf("decode log: %v\n%s", err, raw)
	}
	return line
}
