package httpx_test

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"regexp"
	"testing"

	"github.com/Ammarmyp/payments/internal/platform/httpx"
	"github.com/Ammarmyp/payments/internal/platform/logging"
	"github.com/google/uuid"
)

func TestRequestIDHonorsValidInboundHeader(t *testing.T) {
	var buf bytes.Buffer
	log, err := logging.New(logging.Options{
		Writer:  &buf,
		Service: "wallet",
		Version: "dev",
		Level:   slog.LevelInfo,
		Format:  logging.FormatJSON,
	})
	if err != nil {
		t.Fatalf("logging.New: %v", err)
	}

	var seen string
	h := httpx.RequestID(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		seen = w.Header().Get(httpx.RequestIDHeader)
		log.InfoContext(r.Context(), "handled")
		w.WriteHeader(http.StatusNoContent)
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set(httpx.RequestIDHeader, "client-req_1")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if got := rec.Header().Get(httpx.RequestIDHeader); got != "client-req_1" {
		t.Fatalf("response %s = %q, want client-req_1", httpx.RequestIDHeader, got)
	}
	if seen != "client-req_1" {
		t.Fatalf("handler saw response header %q, want client-req_1", seen)
	}
	assertLogRequestID(t, buf.Bytes(), "client-req_1")
}

func TestRequestIDMintsWhenMissingOrInvalid(t *testing.T) {
	tests := []struct {
		name  string
		value string
	}{
		{name: "missing", value: ""},
		{name: "too_long", value: "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789_extra"},
		{name: "bad_chars", value: "req id!"},
	}

	uuidRE := regexp.MustCompile(`(?i)^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			log, err := logging.New(logging.Options{
				Writer:  &buf,
				Service: "wallet",
				Version: "dev",
				Level:   slog.LevelInfo,
				Format:  logging.FormatJSON,
			})
			if err != nil {
				t.Fatalf("logging.New: %v", err)
			}

			h := httpx.RequestID(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				log.InfoContext(r.Context(), "handled")
				w.WriteHeader(http.StatusNoContent)
			}))

			req := httptest.NewRequest(http.MethodGet, "/", nil)
			if tt.value != "" {
				req.Header.Set(httpx.RequestIDHeader, tt.value)
			}
			rec := httptest.NewRecorder()
			h.ServeHTTP(rec, req)

			got := rec.Header().Get(httpx.RequestIDHeader)
			if !uuidRE.MatchString(got) {
				t.Fatalf("response %s = %q, want UUID", httpx.RequestIDHeader, got)
			}
			if _, err := uuid.Parse(got); err != nil {
				t.Fatalf("parse minted id: %v", err)
			}
			assertLogRequestID(t, buf.Bytes(), got)
		})
	}
}

func assertLogRequestID(t *testing.T, raw []byte, want string) {
	t.Helper()
	var line map[string]any
	if err := json.Unmarshal(raw, &line); err != nil {
		t.Fatalf("decode log: %v\n%s", err, raw)
	}
	if line["request_id"] != want {
		t.Fatalf("log request_id = %v, want %q", line["request_id"], want)
	}
}
