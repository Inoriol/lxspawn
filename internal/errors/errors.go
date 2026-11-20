package errors

import (
	"errors"
	"fmt"
)

type ErrorCode string

const (
	ErrCodeMachineNotFound    ErrorCode = "MACHINE_NOT_FOUND"
	ErrCodeMachineExists      ErrorCode = "MACHINE_EXISTS"
	ErrCodeMachineRunning     ErrorCode = "MACHINE_RUNNING"
	ErrCodeMachineStopped     ErrorCode = "MACHINE_STOPPED"
	ErrCodePermissionDenied   ErrorCode = "PERMISSION_DENIED"
	ErrCodeValidationFailed   ErrorCode = "VALIDATION_FAILED"
	ErrCodeStorageError       ErrorCode = "STORAGE_ERROR"
	ErrCodeProvisionFailed    ErrorCode = "PROVISION_FAILED"
	ErrCodeNetworkError       ErrorCode = "NETWORK_ERROR"
	ErrCodeImageNotFound      ErrorCode = "IMAGE_NOT_FOUND"
	ErrCodeConfigInvalid      ErrorCode = "CONFIG_INVALID"
	ErrCodeSnapshotError      ErrorCode = "SNAPSHOT_ERROR"
	ErrCodeSystemdError       ErrorCode = "SYSTEMD_ERROR"
	ErrCodeOperationCancelled ErrorCode = "OPERATION_CANCELLED"
	ErrCodeInternalError      ErrorCode = "INTERNAL_ERROR"
)

type LxspawnError struct {
	Code    ErrorCode `json:"code"`
	Message string    `json:"message"`
	Field   string    `json:"field,omitempty"`
	Err     error     `json:"-"`
}

func (e *LxspawnError) Error() string {
	if e.Field != "" {
		return fmt.Sprintf("%s: %s (field: %s)", e.Code, e.Message, e.Field)
	}
	return fmt.Sprintf("%s: %s", e.Code, e.Message)
}

func (e *LxspawnError) Unwrap() error {
	return e.Err
}

func NewMachineNotFoundError(name string) *LxspawnError {
	return &LxspawnError{
		Code:    ErrCodeMachineNotFound,
		Message: fmt.Sprintf("machine '%s' not found", name),
	}
}

func NewMachineExistsError(name string) *LxspawnError {
	return &LxspawnError{
		Code:    ErrCodeMachineExists,
		Message: fmt.Sprintf("machine '%s' already exists", name),
	}
}

func NewMachineRunningError(name string) *LxspawnError {
	return &LxspawnError{
		Code:    ErrCodeMachineRunning,
		Message: fmt.Sprintf("machine '%s' is running", name),
	}
}

func NewMachineStoppedError(name string) *LxspawnError {
	return &LxspawnError{
		Code:    ErrCodeMachineStopped,
		Message: fmt.Sprintf("machine '%s' is stopped", name),
	}
}

func NewPermissionDeniedError(msg string) *LxspawnError {
	return &LxspawnError{
		Code:    ErrCodePermissionDenied,
		Message: msg,
	}
}

func NewValidationError(field, msg string) *LxspawnError {
	return &LxspawnError{
		Code:    ErrCodeValidationFailed,
		Message: msg,
		Field:   field,
	}
}

func NewStorageError(msg string, err error) *LxspawnError {
	return &LxspawnError{
		Code:    ErrCodeStorageError,
		Message: msg,
		Err:     err,
	}
}

func NewProvisionError(msg string, err error) *LxspawnError {
	return &LxspawnError{
		Code:    ErrCodeProvisionFailed,
		Message: msg,
		Err:     err,
	}
}

func NewNetworkError(msg string, err error) *LxspawnError {
	return &LxspawnError{
		Code:    ErrCodeNetworkError,
		Message: msg,
		Err:     err,
	}
}

func NewConfigInvalidError(msg string) *LxspawnError {
	return &LxspawnError{
		Code:    ErrCodeConfigInvalid,
		Message: msg,
	}
}

func NewSnapshotError(msg string, err error) *LxspawnError {
	return &LxspawnError{
		Code:    ErrCodeSnapshotError,
		Message: msg,
		Err:     err,
	}
}

func NewSystemdError(msg string, err error) *LxspawnError {
	return &LxspawnError{
		Code:    ErrCodeSystemdError,
		Message: msg,
		Err:     err,
	}
}

func NewInternalError(msg string, err error) *LxspawnError {
	return &LxspawnError{
		Code:    ErrCodeInternalError,
		Message: msg,
		Err:     err,
	}
}

func IsLxspawnError(err error) bool {
	var lxErr *LxspawnError
	return errors.As(err, &lxErr)
}

func GetErrorCode(err error) ErrorCode {
	var lxErr *LxspawnError
	if errors.As(err, &lxErr) {
		return lxErr.Code
	}
	return ErrCodeInternalError
}

func ExitCodeForError(err error) int {
	if !IsLxspawnError(err) {
		return 1
	}

	code := GetErrorCode(err)
	switch code {
	case ErrCodeMachineNotFound:
		return 2
	case ErrCodeMachineExists:
		return 3
	case ErrCodeMachineRunning:
		return 4
	case ErrCodeMachineStopped:
		return 5
	case ErrCodePermissionDenied:
		return 13
	case ErrCodeValidationFailed, ErrCodeConfigInvalid:
		return 65
	case ErrCodeStorageError:
		return 70
	case ErrCodeProvisionFailed:
		return 71
	case ErrCodeNetworkError:
		return 72
	case ErrCodeSnapshotError:
		return 73
	case ErrCodeSystemdError:
		return 74
	case ErrCodeOperationCancelled:
		return 130
	default:
		return 1
	}
}
