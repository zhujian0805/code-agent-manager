package desktop

import (
	"errors"
	"testing"
)

func TestAppError(t *testing.T) {
	err := NewError("CODE", "message", map[string]string{"k": "v"})
	if err.Code != "CODE" {
		t.Fatalf("expected CODE, got %s", err.Code)
	}
	if err.Error() != "CODE: message" {
		t.Fatalf("unexpected error string: %s", err.Error())
	}

	wrapped := wrapError("WRAPPED", errors.New("boom"))
	var appErr AppError
	if !errors.As(wrapped, &appErr) {
		t.Fatal("expected AppError")
	}
	if appErr.Code != "WRAPPED" {
		t.Fatalf("expected WRAPPED, got %s", appErr.Code)
	}
}
