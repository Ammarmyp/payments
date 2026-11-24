// Package apperr defines shared service errors: what went wrong, not how to
// report it. HTTP and later gRPC map these to transport codes.
package apperr

import (
	"errors"
	"maps"
	"slices"
	"strings"
)

// Code is a stable failure class for clients.
type Code string

const (
	InvalidArgument    Code = "invalid_argument"
	NotFound           Code = "not_found"
	FailedPrecondition Code = "failed_precondition"
	Conflict           Code = "conflict"
	Internal           Code = "internal"
)

// Sentinels for errors.Is. Use New/Wrap when the client needs a specific message.
//
//	fmt.Errorf("account %s: %w", id, apperr.ErrNotFound)
var (
	ErrNotFound = New(NotFound, "not found")
	ErrConflict = New(Conflict, "conflict")
)

// Error is an application failure without HTTP or gRPC status.
type Error struct {
	Code    Code
	Message string
	cause   error
}

// New returns an application error with a client-safe message.
func New(code Code, message string) *Error {
	return &Error{Code: code, Message: message}
}

// Wrap returns an application error that unwraps to cause.
func Wrap(code Code, message string, cause error) *Error {
	return &Error{Code: code, Message: message, cause: cause}
}

func (e *Error) Error() string {
	if e == nil {
		return ""
	}
	if e.cause != nil {
		return string(e.Code) + ": " + e.Message + ": " + e.cause.Error()
	}
	return string(e.Code) + ": " + e.Message
}

func (e *Error) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.cause
}

// Is matches another *Error with the same Code.
func (e *Error) Is(target error) bool {
	t, ok := target.(*Error)
	if !ok || e == nil || t == nil {
		return false
	}
	return e.Code == t.Code
}

// ValidationError collects invalid request fields.
type ValidationError struct {
	Fields map[string]string
}

// Add records a problem with one field.
func (v *ValidationError) Add(field, problem string) {
	if v.Fields == nil {
		v.Fields = make(map[string]string)
	}
	v.Fields[field] = problem
}

// Err returns v when any field is invalid, otherwise nil.
// Always return Err(), never a nil *ValidationError as error.
func (v *ValidationError) Err() error {
	if len(v.Fields) == 0 {
		return nil
	}
	return v
}

func (v *ValidationError) Error() string {
	fields := slices.Sorted(maps.Keys(v.Fields))
	return "invalid input: " + strings.Join(fields, ", ")
}

// AsError maps err to an *Error when possible.
func AsError(err error) (*Error, bool) {
	var appErr *Error
	if errors.As(err, &appErr) && appErr != nil {
		return appErr, true
	}
	var valid *ValidationError
	if errors.As(err, &valid) && valid != nil && len(valid.Fields) > 0 {
		return New(InvalidArgument, valid.Error()), true
	}
	return nil, false
}
