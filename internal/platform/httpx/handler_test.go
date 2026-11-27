package httpx_test

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Ammarmyp/payments/internal/platform/apperr"
	"github.com/Ammarmyp/payments/internal/platform/httpx"
)

func TestHandlerFuncMapsReturnedError(t *testing.T) {
	setTestLogger(t, discardLogger(t))
	h := httpx.HandlerFunc(func(http.ResponseWriter, *http.Request) error {
		return fmt.Errorf("account: %w", apperr.ErrNotFound)
	})

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	h.ServeHTTP(rec, req)

	assertErrorResponse(t, rec, http.StatusNotFound, "not_found", "not found")
}

func TestHandlerFuncLeavesSuccessAlone(t *testing.T) {
	h := httpx.HandlerFunc(func(w http.ResponseWriter, r *http.Request) error {
		httpx.WriteJSON(w, r, http.StatusOK, map[string]string{"ok": "1"})
		return nil
	})

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
}
