package httpx_test

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Ammarmyp/payments/internal/platform/apperr"
	"github.com/Ammarmyp/payments/internal/platform/httpx"
	"github.com/Ammarmyp/payments/internal/platform/logging"
)

func TestErrorMapsApplicationCodes(t *testing.T) {
	tests := []struct {
		name   string
		err    error
		status int
		code   string
		msg    string
	}{
		{
			name:   "invalid_argument",
			err:    apperr.New(apperr.InvalidArgument, "bad input"),
			status: http.StatusBadRequest,
			code:   "invalid_argument",
			msg:    "bad input",
		},
		{
			name:   "not_found",
			err:    apperr.New(apperr.NotFound, "account not found"),
			status: http.StatusNotFound,
			code:   "not_found",
			msg:    "account not found",
		},
		{
			name:   "failed_precondition",
			err:    apperr.New(apperr.FailedPrecondition, "not ready"),
			status: http.StatusPreconditionFailed,
			code:   "failed_precondition",
			msg:    "not ready",
		},
		{
			name:   "conflict",
			err:    apperr.New(apperr.Conflict, "already exists"),
			status: http.StatusConflict,
			code:   "conflict",
			msg:    "already exists",
		},
		{
			name:   "internal",
			err:    apperr.New(apperr.Internal, "storage unavailable"),
			status: http.StatusInternalServerError,
			code:   "internal",
			msg:    "internal server error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			setTestLogger(t, discardLogger(t))
			rec := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodGet, "/", nil)
			httpx.Error(rec, req, tt.err)
			assertErrorResponse(t, rec, tt.status, tt.code, tt.msg)
		})
	}
}

func TestErrorMapsUnknownToInternal(t *testing.T) {
	setTestLogger(t, discardLogger(t))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	httpx.Error(rec, req, errors.New("sql: connection refused"))

	assertErrorResponse(t, rec, http.StatusInternalServerError, "internal", "internal server error")
	if strings.Contains(rec.Body.String(), "connection refused") {
		t.Fatalf("body leaked internals: %s", rec.Body.String())
	}
}

func TestErrorOmitsWrappedCauseFromBody(t *testing.T) {
	setTestLogger(t, discardLogger(t))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	httpx.Error(rec, req, apperr.Wrap(apperr.NotFound, "account not found", errors.New("row missing")))

	assertErrorResponse(t, rec, http.StatusNotFound, "not_found", "account not found")
	if strings.Contains(rec.Body.String(), "row missing") {
		t.Fatalf("body leaked cause: %s", rec.Body.String())
	}
}

func TestErrorMapsValidationError(t *testing.T) {
	var v apperr.ValidationError
	v.Add("amount", "must be positive")

	setTestLogger(t, discardLogger(t))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	httpx.Error(rec, req, v.Err())

	assertErrorResponse(t, rec, http.StatusUnprocessableEntity, "invalid_input", "invalid input: amount")

	var body struct {
		Error struct {
			Fields map[string]string `json:"fields"`
		} `json:"error"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if body.Error.Fields["amount"] != "must be positive" {
		t.Fatalf("fields = %#v", body.Error.Fields)
	}
}

func TestErrorMapsSentinelNotFound(t *testing.T) {
	setTestLogger(t, discardLogger(t))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	httpx.Error(rec, req, fmt.Errorf("account A1: %w", apperr.ErrNotFound))

	assertErrorResponse(t, rec, http.StatusNotFound, "not_found", "not found")
}

func TestErrorIncludesRequestIDInBody(t *testing.T) {
	setTestLogger(t, discardLogger(t))
	h := httpx.RequestID(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		httpx.Error(w, r, apperr.ErrNotFound)
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set(httpx.RequestIDHeader, "body-req-1")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	var body struct {
		Error struct {
			RequestID string `json:"request_id"`
		} `json:"error"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if body.Error.RequestID != "body-req-1" {
		t.Fatalf("request_id = %q, want body-req-1", body.Error.RequestID)
	}
}

func TestNotFoundReturnsJSON(t *testing.T) {
	setTestLogger(t, discardLogger(t))
	h := httpx.RequestID(http.HandlerFunc(httpx.NotFound))
	req := httptest.NewRequest(http.MethodGet, "/missing", nil)
	req.Header.Set(httpx.RequestIDHeader, "nf-1")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	assertErrorResponse(t, rec, http.StatusNotFound, "not_found", "not found")
}

func discardLogger(t *testing.T) *slog.Logger {
	t.Helper()
	log, err := logging.New(logging.Options{
		Writer:  io.Discard,
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

func setTestLogger(t *testing.T, log *slog.Logger) {
	t.Helper()
	prev := slog.Default()
	slog.SetDefault(log)
	t.Cleanup(func() { slog.SetDefault(prev) })
}

func TestWriteJSONEncodesSuccessBody(t *testing.T) {
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	payload := map[string]string{"status": "ok"}

	httpx.WriteJSON(rec, req, http.StatusOK, payload)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	if ct := rec.Header().Get("Content-Type"); ct != "application/json" {
		t.Fatalf("Content-Type = %q, want application/json", ct)
	}

	var got map[string]string
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatalf("decode body: %v\nbody=%s", err, rec.Body.String())
	}
	if got["status"] != "ok" {
		t.Fatalf("body = %#v, want status=ok", got)
	}
}

func assertErrorResponse(t *testing.T, rec *httptest.ResponseRecorder, status int, code, message string) {
	t.Helper()
	if rec.Code != status {
		t.Fatalf("status = %d, want %d\nbody=%s", rec.Code, status, rec.Body.String())
	}
	if ct := rec.Header().Get("Content-Type"); ct != "application/json" {
		t.Fatalf("Content-Type = %q, want application/json", ct)
	}

	var body struct {
		Error struct {
			Code    string `json:"code"`
			Message string `json:"message"`
		} `json:"error"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode body: %v\nbody=%s", err, rec.Body.String())
	}
	if body.Error.Code != code {
		t.Fatalf("code = %q, want %q", body.Error.Code, code)
	}
	if body.Error.Message != message {
		t.Fatalf("message = %q, want %q", body.Error.Message, message)
	}
}
