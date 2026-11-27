package httpx_test

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Ammarmyp/payments/internal/platform/apperr"
	"github.com/Ammarmyp/payments/internal/platform/httpx"
)

func TestQueryIntReturnsDefaultWhenMissing(t *testing.T) {
	r := httptest.NewRequest(http.MethodGet, "/items", nil)
	got, err := httpx.QueryInt(r, "limit", 20, 1, 100)
	if err != nil {
		t.Fatalf("QueryInt: %v", err)
	}
	if got != 20 {
		t.Fatalf("got %d, want 20", got)
	}
}

func TestQueryIntParsesPresentValue(t *testing.T) {
	r := httptest.NewRequest(http.MethodGet, "/items?limit=7", nil)
	got, err := httpx.QueryInt(r, "limit", 20, 1, 100)
	if err != nil {
		t.Fatalf("QueryInt: %v", err)
	}
	if got != 7 {
		t.Fatalf("got %d, want 7", got)
	}
}

func TestQueryIntRejectsNonInteger(t *testing.T) {
	r := httptest.NewRequest(http.MethodGet, "/items?limit=abc", nil)
	_, err := httpx.QueryInt(r, "limit", 20, 1, 100)
	assertValidationField(t, err, "limit")
}

func TestQueryIntRejectsOutOfRange(t *testing.T) {
	r := httptest.NewRequest(http.MethodGet, "/items?limit=101", nil)
	_, err := httpx.QueryInt(r, "limit", 20, 1, 100)
	assertValidationField(t, err, "limit")

	r = httptest.NewRequest(http.MethodGet, "/items?limit=0", nil)
	_, err = httpx.QueryInt(r, "limit", 20, 1, 100)
	assertValidationField(t, err, "limit")
}

func assertValidationField(t *testing.T, err error, field string) {
	t.Helper()
	var v *apperr.ValidationError
	if !errors.As(err, &v) {
		t.Fatalf("err type = %T (%v), want ValidationError", err, err)
	}
	if _, ok := v.Fields[field]; !ok {
		t.Fatalf("fields missing %q: %#v", field, v.Fields)
	}
}
