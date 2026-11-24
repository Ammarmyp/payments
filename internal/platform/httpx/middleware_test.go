package httpx_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Ammarmyp/payments/internal/platform/httpx"
)

func TestChainAppliesFirstMiddlewareOutermost(t *testing.T) {
	var order []string

	trace := func(name string) httpx.Middleware {
		return func(next http.Handler) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				order = append(order, name+"-in")
				next.ServeHTTP(w, r)
				order = append(order, name+"-out")
			})
		}
	}

	h := httpx.Chain(
		http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
			order = append(order, "handler")
		}),
		trace("a"),
		trace("b"),
	)

	h.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/", nil))

	want := []string{"a-in", "b-in", "handler", "b-out", "a-out"}
	if len(order) != len(want) {
		t.Fatalf("order = %v, want %v", order, want)
	}
	for i := range want {
		if order[i] != want[i] {
			t.Fatalf("order = %v, want %v", order, want)
		}
	}
}
