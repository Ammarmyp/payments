package httpx_test

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Ammarmyp/payments/internal/platform/httpx"
)

func TestWriteJSONEncodeFailureReturnsInternal(t *testing.T) {
	setTestLogger(t, discardLogger(t))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)

	httpx.WriteJSON(rec, req, http.StatusOK, make(chan int))
	assertErrorResponse(t, rec, http.StatusInternalServerError, "internal", "internal server error")
}

func TestDecodeJSONAcceptsSingleValue(t *testing.T) {
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(`{"name":"ada"}`))

	var dst struct {
		Name string `json:"name"`
	}
	if err := httpx.DecodeJSON(rec, req, &dst); err != nil {
		t.Fatalf("DecodeJSON: %v", err)
	}
	if dst.Name != "ada" {
		t.Fatalf("name = %q, want ada", dst.Name)
	}
}

func TestDecodeJSONRejectsBadBodies(t *testing.T) {
	tests := []struct {
		name   string
		body   string
		status int
		code   string
		msg    string
	}{
		{
			name:   "empty",
			body:   "",
			status: http.StatusBadRequest,
			code:   "bad_request",
			msg:    "body must not be empty",
		},
		{
			name:   "malformed",
			body:   `{"name":`,
			status: http.StatusBadRequest,
			code:   "bad_request",
			msg:    "malformed JSON",
		},
		{
			name:   "unknown_field",
			body:   `{"name":"ada","extra":true}`,
			status: http.StatusBadRequest,
			code:   "bad_request",
			msg:    `unknown field "extra"`,
		},
		{
			name:   "wrong_type",
			body:   `{"name":1}`,
			status: http.StatusBadRequest,
			code:   "bad_request",
			msg:    `field "name" must be of type string`,
		},
		{
			name:   "multiple_values",
			body:   `{"name":"ada"}{"name":"grace"}`,
			status: http.StatusBadRequest,
			code:   "bad_request",
			msg:    "body must contain a single JSON value",
		},
	}

	type dst struct {
		Name string `json:"name"`
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			setTestLogger(t, discardLogger(t))
			rec := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(tt.body))

			err := httpx.DecodeJSON(rec, req, &dst{})
			if err == nil {
				t.Fatal("DecodeJSON: want error")
			}

			errRec := httptest.NewRecorder()
			httpx.Error(errRec, req, err)
			assertErrorResponse(t, errRec, tt.status, tt.code, tt.msg)
		})
	}
}

func TestDecodeJSONRejectsOversizedBody(t *testing.T) {
	setTestLogger(t, discardLogger(t))

	// Larger than maxBodyBytes (1 MiB) so MaxBytesReader trips.
	body := `{"name":"` + strings.Repeat("a", 1<<20) + `"}`
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(body))

	var dst struct {
		Name string `json:"name"`
	}
	err := httpx.DecodeJSON(rec, req, &dst)
	if err == nil {
		t.Fatal("DecodeJSON: want error")
	}

	errRec := httptest.NewRecorder()
	httpx.Error(errRec, req, err)
	assertErrorResponse(t, errRec, http.StatusRequestEntityTooLarge, "body_too_large", "body must be at most 1048576 bytes")
}

func TestDecodeJSONAllowsTrailingWhitespace(t *testing.T) {
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/", bytes.NewBufferString("{\"name\":\"ada\"}\n  "))

	var dst struct {
		Name string `json:"name"`
	}
	if err := httpx.DecodeJSON(rec, req, &dst); err != nil {
		t.Fatalf("DecodeJSON: %v", err)
	}
	if dst.Name != "ada" {
		t.Fatalf("name = %q, want ada", dst.Name)
	}
}

func TestDecodeJSONSyntaxErrorIncludesOffset(t *testing.T) {
	setTestLogger(t, discardLogger(t))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(`{"name":tru}`))

	var dst struct {
		Name string `json:"name"`
	}
	err := httpx.DecodeJSON(rec, req, &dst)
	if err == nil {
		t.Fatal("DecodeJSON: want error")
	}

	errRec := httptest.NewRecorder()
	httpx.Error(errRec, req, err)

	var body struct {
		Error struct {
			Code    string `json:"code"`
			Message string `json:"message"`
		} `json:"error"`
	}
	if err := json.Unmarshal(errRec.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if body.Error.Code != "bad_request" {
		t.Fatalf("code = %q, want bad_request", body.Error.Code)
	}
	if !strings.HasPrefix(body.Error.Message, "malformed JSON at byte ") {
		t.Fatalf("message = %q, want malformed JSON at byte …", body.Error.Message)
	}
}

func TestWriteJSONBodyEndsWithNewline(t *testing.T) {
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	httpx.WriteJSON(rec, req, http.StatusOK, map[string]string{"status": "ok"})

	raw, err := io.ReadAll(rec.Body)
	if err != nil {
		t.Fatalf("read body: %v", err)
	}
	if len(raw) == 0 || raw[len(raw)-1] != '\n' {
		t.Fatalf("body = %q, want trailing newline", raw)
	}
}
