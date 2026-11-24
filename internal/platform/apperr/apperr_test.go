package apperr_test

import (
	"errors"
	"fmt"
	"testing"

	"github.com/Ammarmyp/payments/internal/platform/apperr"
)

func TestSentinelIsMatchesSameCode(t *testing.T) {
	err := fmt.Errorf("account %s: %w", "A1", apperr.ErrNotFound)
	if !errors.Is(err, apperr.ErrNotFound) {
		t.Fatal("expected errors.Is to match ErrNotFound")
	}
	if errors.Is(err, apperr.ErrConflict) {
		t.Fatal("did not expect ErrConflict match")
	}

	custom := apperr.New(apperr.NotFound, "account not found")
	if !errors.Is(custom, apperr.ErrNotFound) {
		t.Fatal("expected same-code New to match ErrNotFound")
	}
}

func TestValidationErrorErrNilWhenEmpty(t *testing.T) {
	var v apperr.ValidationError
	if err := v.Err(); err != nil {
		t.Fatalf("Err() = %v, want nil", err)
	}
}

func TestValidationErrorErrReturnsSelfWhenInvalid(t *testing.T) {
	var v apperr.ValidationError
	v.Add("email", "required")
	v.Add("amount", "must be positive")
	err := v.Err()
	if err == nil {
		t.Fatal("Err() = nil, want validation error")
	}
	got := err.Error()
	want := "invalid input: amount, email"
	if got != want {
		t.Fatalf("Error() = %q, want %q", got, want)
	}
}

func TestAsErrorMapsValidation(t *testing.T) {
	var v apperr.ValidationError
	v.Add("name", "required")
	appErr, ok := apperr.AsError(v.Err())
	if !ok {
		t.Fatal("AsError: want ok")
	}
	if appErr.Code != apperr.InvalidArgument {
		t.Fatalf("Code = %q, want %q", appErr.Code, apperr.InvalidArgument)
	}
	if appErr.Message != "invalid input: name" {
		t.Fatalf("Message = %q, want invalid input: name", appErr.Message)
	}
}
