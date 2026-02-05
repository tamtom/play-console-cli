package shared

import (
	"errors"
	"fmt"
	"net/http"
	"strings"

	"google.golang.org/api/googleapi"
)

// ActionableError adds context and an optional hint to an error.
type ActionableError struct {
	Op    string
	Cause error
	Hint  string
}

func (e *ActionableError) Error() string {
	return formatActionable(e.Op, e.Cause, e.Hint)
}

func (e *ActionableError) Unwrap() error {
	return e.Cause
}

// AuthError represents authentication failures.
type AuthError struct{ ActionableError }

// PermissionError represents permission/authorization failures.
type PermissionError struct{ ActionableError }

// NotFoundError represents missing resources.
type NotFoundError struct{ ActionableError }

// ValidationError represents malformed requests or invalid inputs.
type ValidationError struct{ ActionableError }

// NewActionableError builds an ActionableError.
func NewActionableError(op string, cause error, hint string) error {
	if cause == nil && strings.TrimSpace(op) == "" {
		return nil
	}
	return &ActionableError{Op: op, Cause: cause, Hint: hint}
}

// WrapActionable wraps err with context and an optional hint.
func WrapActionable(err error, op, hint string) error {
	if err == nil {
		return nil
	}
	return &ActionableError{Op: op, Cause: err, Hint: hint}
}

// NewAuthError builds an AuthError.
func NewAuthError(op string, cause error, hint string) error {
	if cause == nil && strings.TrimSpace(op) == "" {
		return nil
	}
	return &AuthError{ActionableError: ActionableError{Op: op, Cause: cause, Hint: hint}}
}

// NewPermissionError builds a PermissionError.
func NewPermissionError(op string, cause error, hint string) error {
	if cause == nil && strings.TrimSpace(op) == "" {
		return nil
	}
	return &PermissionError{ActionableError: ActionableError{Op: op, Cause: cause, Hint: hint}}
}

// NewNotFoundError builds a NotFoundError.
func NewNotFoundError(op string, cause error, hint string) error {
	if cause == nil && strings.TrimSpace(op) == "" {
		return nil
	}
	return &NotFoundError{ActionableError: ActionableError{Op: op, Cause: cause, Hint: hint}}
}

// NewValidationError builds a ValidationError.
func NewValidationError(op string, cause error, hint string) error {
	if cause == nil && strings.TrimSpace(op) == "" {
		return nil
	}
	return &ValidationError{ActionableError: ActionableError{Op: op, Cause: cause, Hint: hint}}
}

// WrapGoogleAPIError adds contextual hints for common Google API failures.
func WrapGoogleAPIError(op string, err error) error {
	if err == nil {
		return nil
	}
	hint, kind := hintForGoogleAPIError(err)
	switch kind {
	case "auth":
		return NewAuthError(op, err, hint)
	case "permission":
		return NewPermissionError(op, err, hint)
	case "not_found":
		return NewNotFoundError(op, err, hint)
	case "validation":
		return NewValidationError(op, err, hint)
	default:
		return WrapActionable(err, op, hint)
	}
}

func hintForGoogleAPIError(err error) (string, string) {
	var gerr *googleapi.Error
	if !errors.As(err, &gerr) {
		return "", ""
	}
	switch gerr.Code {
	case http.StatusUnauthorized:
		return "Check that the service account or OAuth token is valid and has access to the Play Console.", "auth"
	case http.StatusForbidden:
		return "Check that the account has access to the app and the required Play Console permission (for uploads, Release Manager is typically required).", "permission"
	case http.StatusNotFound:
		return "Check that the package name, edit ID, and resource IDs are correct.", "not_found"
	case http.StatusBadRequest:
		return "Check request parameters and file type. Use --help to verify flags.", "validation"
	default:
		return "", ""
	}
}

func formatActionable(op string, cause error, hint string) string {
	op = strings.TrimSpace(op)
	hint = strings.TrimSpace(hint)

	var msg string
	switch {
	case op == "" && cause == nil:
		msg = "unknown error"
	case op == "":
		msg = cause.Error()
	case cause == nil:
		msg = op
	default:
		msg = fmt.Sprintf("%s: %v", op, cause)
	}

	if hint != "" {
		msg = fmt.Sprintf("%s\n\nHint: %s", msg, hint)
	}
	return msg
}
