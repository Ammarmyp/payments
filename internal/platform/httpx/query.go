package httpx

import (
	"net/http"
	"strconv"

	"github.com/Ammarmyp/payments/internal/platform/apperr"
)

// QueryInt returns the named query parameter as an int.
// Missing uses defaultVal. Present but non-integer or outside [min, max]
// returns a ValidationError on name.
func QueryInt(r *http.Request, name string, defaultVal, min, max int) (int, error) {
	raw := r.URL.Query().Get(name)
	if raw == "" {
		return defaultVal, nil
	}
	n, err := strconv.Atoi(raw)
	if err != nil {
		var v apperr.ValidationError
		v.Add(name, "must be an integer")
		return 0, v.Err()
	}
	if n < min || n > max {
		var v apperr.ValidationError
		v.Add(name, "out of range")
		return 0, v.Err()
	}
	return n, nil
}
