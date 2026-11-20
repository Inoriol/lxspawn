package errors

import (
	"errors"
	"testing"
)

func TestNewMachineNotFoundError(t *testing.T) {
	err := NewMachineNotFoundError("test-machine")
	if err.Code != ErrCodeMachineNotFound {
		t.Errorf("expected code %s, got %s", ErrCodeMachineNotFound, err.Code)
	}
	if err.Error() != "MACHINE_NOT_FOUND: machine 'test-machine' not found" {
		t.Errorf("unexpected error message: %s", err.Error())
	}
}

func TestNewMachineExistsError(t *testing.T) {
	err := NewMachineExistsError("test-machine")
	if err.Code != ErrCodeMachineExists {
		t.Errorf("expected code %s, got %s", ErrCodeMachineExists, err.Code)
	}
}

func TestNewValidationError(t *testing.T) {
	err := NewValidationError("name", "name cannot be empty")
	if err.Code != ErrCodeValidationFailed {
		t.Errorf("expected code %s, got %s", ErrCodeValidationFailed, err.Code)
	}
	if err.Field != "name" {
		t.Errorf("expected field 'name', got %q", err.Field)
	}
}

func TestNewStorageError(t *testing.T) {
	inner := errors.New("disk full")
	err := NewStorageError("failed to create", inner)
	if err.Code != ErrCodeStorageError {
		t.Errorf("expected code %s, got %s", ErrCodeStorageError, err.Code)
	}
	if !errors.Is(err, inner) {
		t.Error("expected error to wrap inner error")
	}
}

func TestIsLxspawnError(t *testing.T) {
	lxErr := NewMachineNotFoundError("test")
	if !IsLxspawnError(lxErr) {
		t.Error("expected IsLxspawnError to return true for LxspawnError")
	}

	plainErr := errors.New("plain error")
	if IsLxspawnError(plainErr) {
		t.Error("expected IsLxspawnError to return false for plain error")
	}
}

func TestGetErrorCode(t *testing.T) {
	lxErr := NewMachineNotFoundError("test")
	if code := GetErrorCode(lxErr); code != ErrCodeMachineNotFound {
		t.Errorf("expected %s, got %s", ErrCodeMachineNotFound, code)
	}

	plainErr := errors.New("plain error")
	if code := GetErrorCode(plainErr); code != ErrCodeInternalError {
		t.Errorf("expected %s for plain error, got %s", ErrCodeInternalError, code)
	}
}

func TestExitCodeForError(t *testing.T) {
	tests := []struct {
		err      error
		expected int
	}{
		{NewMachineNotFoundError("test"), 2},
		{NewMachineExistsError("test"), 3},
		{NewMachineRunningError("test"), 4},
		{NewMachineStoppedError("test"), 5},
		{NewPermissionDeniedError("denied"), 13},
		{NewValidationError("field", "msg"), 65},
		{NewConfigInvalidError("msg"), 65},
		{NewStorageError("msg", nil), 70},
		{NewProvisionError("msg", nil), 71},
		{NewNetworkError("msg", nil), 72},
		{NewSnapshotError("msg", nil), 73},
		{NewSystemdError("msg", nil), 74},
		{errors.New("plain error"), 1},
	}

	for _, tt := range tests {
		if code := ExitCodeForError(tt.err); code != tt.expected {
			t.Errorf("ExitCodeForError(%v) = %d, want %d", tt.err, code, tt.expected)
		}
	}
}

func TestErrorUnwrap(t *testing.T) {
	inner := errors.New("inner error")
	err := NewStorageError("outer", inner)
	if !errors.Is(err, inner) {
		t.Error("expected errors.Is to find inner error")
	}
}
