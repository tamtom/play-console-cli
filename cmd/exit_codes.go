package cmd

import (
	"errors"
	"flag"

	"github.com/tamtom/play-console-cli/internal/cli/shared"
	"google.golang.org/api/googleapi"
)

const (
	ExitSuccess  = 0
	ExitError    = 1
	ExitUsage    = 2
	ExitAuth     = 3
	ExitNotFound = 4
	ExitConflict = 5
)

// ExitCodeFromError maps an error to a structured exit code.
func ExitCodeFromError(err error) int {
	if err == nil {
		return ExitSuccess
	}

	// flag.ErrHelp = usage error
	if errors.Is(err, flag.ErrHelp) {
		return ExitUsage
	}

	// Typed errors from shared/errors.go
	var authErr *shared.AuthError
	if errors.As(err, &authErr) {
		return ExitAuth
	}

	var notFoundErr *shared.NotFoundError
	if errors.As(err, &notFoundErr) {
		return ExitNotFound
	}

	// Google API errors by HTTP status
	var gerr *googleapi.Error
	if errors.As(err, &gerr) {
		return HTTPStatusToExitCode(gerr.Code)
	}

	return ExitError
}

// HTTPStatusToExitCode maps HTTP status codes to exit codes.
// 4xx -> 10-59, 5xx -> 60-99.
func HTTPStatusToExitCode(status int) int {
	switch {
	case status == 401:
		return ExitAuth
	case status == 404:
		return ExitNotFound
	case status == 409:
		return ExitConflict
	case status >= 400 && status < 500:
		code := 10 + (status - 400)
		if code > 59 {
			code = 59
		}
		return code
	case status >= 500 && status < 600:
		code := 60 + (status - 500)
		if code > 99 {
			code = 99
		}
		return code
	default:
		return ExitError
	}
}
