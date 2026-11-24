package httpx

import (
	"net/http"
	"slices"
)

// Middleware wraps an http.Handler.
type Middleware func(http.Handler) http.Handler

// Chain applies mw around h. The first middleware is outermost.
func Chain(h http.Handler, mw ...Middleware) http.Handler {
	for _, m := range slices.Backward(mw) {
		h = m(h)
	}
	return h
}
