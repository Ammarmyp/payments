// Package httpx provides shared HTTP helpers: JSON responses, error mapping,
// request ID / access-log / recovery middleware, and chaining.
package httpx

import (
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"

	"github.com/Ammarmyp/payments/internal/platform/apperr"
)

type errorBody struct {
	Error errorDetail `json:"error"`
}

type errorDetail struct {
	Code      string            `json:"code"`
	Message   string            `json:"message"`
	RequestID string            `json:"request_id,omitempty"`
	Fields    map[string]string `json:"fields,omitempty"`
}

// requestError is an HTTP-layer failure with a fixed status and code.
type requestError struct {
	status int
	code   string
	msg    string
}

func (e *requestError) Error() string { return e.msg }

// classify maps err to an HTTP status and stable code.
func classify(err error) (int, string) {
	var reqErr *requestError
	var valErr *apperr.ValidationError
	var appErr *apperr.Error
	switch {
	case errors.As(err, &reqErr):
		return reqErr.status, reqErr.code
	case errors.As(err, &valErr):
		return http.StatusUnprocessableEntity, "invalid_input"
	case errors.As(err, &appErr) && appErr != nil:
		return statusFor(appErr.Code), string(appErr.Code)
	case errors.Is(err, apperr.ErrNotFound):
		return http.StatusNotFound, "not_found"
	case errors.Is(err, apperr.ErrConflict):
		return http.StatusConflict, "conflict"
	default:
		return http.StatusInternalServerError, "internal"
	}
}

// Error writes err as a JSON error response. Server failures are logged and
// get a generic client message.
func Error(w http.ResponseWriter, r *http.Request, err error) {
	status, code := classify(err)
	d := errorDetail{Code: code, Message: clientMessage(err)}
	if valErr, ok := errors.AsType[*apperr.ValidationError](err); ok {
		d.Fields = valErr.Fields
		d.Message = valErr.Error()
	}
	if status >= http.StatusInternalServerError {
		slog.ErrorContext(r.Context(), "request failed", "err", err, "code", code)
		d.Message = "internal server error"
	}
	writeError(w, r, status, d)
}

func clientMessage(err error) string {
	var appErr *apperr.Error
	if errors.As(err, &appErr) && appErr != nil && appErr.Message != "" {
		return appErr.Message
	}
	if err == nil {
		return ""
	}
	return err.Error()
}

func writeError(w http.ResponseWriter, r *http.Request, status int, d errorDetail) {
	d.RequestID = RequestIDFrom(r.Context())
	body, _ := json.Marshal(errorBody{Error: d})
	writeBody(w, status, body)
}

// NotFound responds with a JSON 404 for unmatched routes.
func NotFound(w http.ResponseWriter, r *http.Request) {
	Error(w, r, fmt.Errorf("route %w", apperr.ErrNotFound))
}

func statusFor(code apperr.Code) int {
	switch code {
	case apperr.InvalidArgument:
		return http.StatusBadRequest
	case apperr.NotFound:
		return http.StatusNotFound
	case apperr.FailedPrecondition:
		return http.StatusPreconditionFailed
	case apperr.Conflict:
		return http.StatusConflict
	case apperr.Internal:
		return http.StatusInternalServerError
	default:
		return http.StatusInternalServerError
	}
}
