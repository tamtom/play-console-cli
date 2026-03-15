package cmd

import (
	"errors"
	"flag"
	"testing"

	"github.com/tamtom/play-console-cli/internal/cli/shared"
	"google.golang.org/api/googleapi"
)

func TestExitCodeFromError_Nil(t *testing.T) {
	if got := ExitCodeFromError(nil); got != ExitSuccess {
		t.Errorf("ExitCodeFromError(nil) = %d, want %d", got, ExitSuccess)
	}
}

func TestExitCodeFromError_FlagErrHelp(t *testing.T) {
	if got := ExitCodeFromError(flag.ErrHelp); got != ExitUsage {
		t.Errorf("ExitCodeFromError(flag.ErrHelp) = %d, want %d", got, ExitUsage)
	}
}

func TestExitCodeFromError_AuthError(t *testing.T) {
	err := shared.NewAuthError("test", errors.New("auth failed"), "")
	if got := ExitCodeFromError(err); got != ExitAuth {
		t.Errorf("ExitCodeFromError(AuthError) = %d, want %d", got, ExitAuth)
	}
}

func TestExitCodeFromError_NotFoundError(t *testing.T) {
	err := shared.NewNotFoundError("test", errors.New("not found"), "")
	if got := ExitCodeFromError(err); got != ExitNotFound {
		t.Errorf("ExitCodeFromError(NotFoundError) = %d, want %d", got, ExitNotFound)
	}
}

func TestExitCodeFromError_GoogleAPI409(t *testing.T) {
	err := &googleapi.Error{Code: 409, Message: "conflict"}
	if got := ExitCodeFromError(err); got != ExitConflict {
		t.Errorf("ExitCodeFromError(googleapi 409) = %d, want %d", got, ExitConflict)
	}
}

func TestExitCodeFromError_GenericError(t *testing.T) {
	err := errors.New("something went wrong")
	if got := ExitCodeFromError(err); got != ExitError {
		t.Errorf("ExitCodeFromError(generic) = %d, want %d", got, ExitError)
	}
}

func TestHTTPStatusToExitCode_400(t *testing.T) {
	if got := HTTPStatusToExitCode(400); got != 10 {
		t.Errorf("HTTPStatusToExitCode(400) = %d, want 10", got)
	}
}

func TestHTTPStatusToExitCode_401(t *testing.T) {
	if got := HTTPStatusToExitCode(401); got != ExitAuth {
		t.Errorf("HTTPStatusToExitCode(401) = %d, want %d", got, ExitAuth)
	}
}

func TestHTTPStatusToExitCode_404(t *testing.T) {
	if got := HTTPStatusToExitCode(404); got != ExitNotFound {
		t.Errorf("HTTPStatusToExitCode(404) = %d, want %d", got, ExitNotFound)
	}
}

func TestHTTPStatusToExitCode_409(t *testing.T) {
	if got := HTTPStatusToExitCode(409); got != ExitConflict {
		t.Errorf("HTTPStatusToExitCode(409) = %d, want %d", got, ExitConflict)
	}
}

func TestHTTPStatusToExitCode_422(t *testing.T) {
	if got := HTTPStatusToExitCode(422); got != 32 {
		t.Errorf("HTTPStatusToExitCode(422) = %d, want 32", got)
	}
}

func TestHTTPStatusToExitCode_500(t *testing.T) {
	if got := HTTPStatusToExitCode(500); got != 60 {
		t.Errorf("HTTPStatusToExitCode(500) = %d, want 60", got)
	}
}

func TestHTTPStatusToExitCode_503(t *testing.T) {
	if got := HTTPStatusToExitCode(503); got != 63 {
		t.Errorf("HTTPStatusToExitCode(503) = %d, want 63", got)
	}
}
