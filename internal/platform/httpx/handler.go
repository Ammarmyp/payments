package httpx

import "net/http"

// HandlerFunc is an http.Handler that returns an error for Error to map.
type HandlerFunc func(http.ResponseWriter, *http.Request) error

// ServeHTTP runs f and writes any returned error as a JSON error response.
func (f HandlerFunc) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if err := f(w, r); err != nil {
		Error(w, r, err)
	}
}
