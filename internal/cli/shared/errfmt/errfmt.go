package errfmt

import (
	"errors"
	"net"
	"os"
	"strings"

	"google.golang.org/api/googleapi"
)

// Category represents the type of error.
type Category string

const (
	CategoryAuth        Category = "auth"
	CategoryPermission  Category = "permission"
	CategoryNotFound    Category = "not_found"
	CategoryTimeout     Category = "timeout"
	CategoryMissingAuth Category = "missing_auth"
	CategoryGeneric     Category = "generic"
)

// ClassifiedError holds a classified error with an actionable hint.
type ClassifiedError struct {
	Original error
	Category Category
	Hint     string
}

func (c *ClassifiedError) Error() string {
	return c.Original.Error()
}

func (c *ClassifiedError) Unwrap() error {
	return c.Original
}

// Classify auto-detects the error category and adds an actionable hint.
func Classify(err error) *ClassifiedError {
	if err == nil {
		return nil
	}

	// Check for Google API errors (401, 403, 404).
	var gerr *googleapi.Error
	if errors.As(err, &gerr) {
		switch gerr.Code {
		case 401:
			return &ClassifiedError{
				Original: err,
				Category: CategoryAuth,
				Hint:     "Your credentials are invalid or expired. Run `gplay auth login` to re-authenticate.",
			}
		case 403:
			return &ClassifiedError{
				Original: err,
				Category: CategoryPermission,
				Hint:     "The service account lacks required permissions. Check Play Console access settings.",
			}
		case 404:
			return &ClassifiedError{
				Original: err,
				Category: CategoryNotFound,
				Hint:     "Resource not found. Verify the package name and resource IDs are correct.",
			}
		}
	}

	// Check for timeout errors.
	var netErr net.Error
	if errors.As(err, &netErr) && netErr.Timeout() {
		return &ClassifiedError{
			Original: err,
			Category: CategoryTimeout,
			Hint:     "Request timed out. Try increasing GPLAY_TIMEOUT or check your network connection.",
		}
	}

	// Check for missing auth (file not found for service account).
	if os.IsNotExist(err) {
		return &ClassifiedError{
			Original: err,
			Category: CategoryMissingAuth,
			Hint:     "Service account file not found. Run `gplay auth doctor` to diagnose.",
		}
	}

	// Check for context deadline exceeded.
	if strings.Contains(err.Error(), "context deadline exceeded") {
		return &ClassifiedError{
			Original: err,
			Category: CategoryTimeout,
			Hint:     "Request timed out. Try increasing GPLAY_TIMEOUT or check your network connection.",
		}
	}

	return &ClassifiedError{Original: err, Category: CategoryGeneric, Hint: ""}
}

// FormatStderr returns a formatted error string for stderr output.
func FormatStderr(err error) string {
	classified := Classify(err)
	if classified == nil {
		return ""
	}

	var sb strings.Builder
	sb.WriteString("Error: ")
	sb.WriteString(classified.Original.Error())

	if classified.Hint != "" {
		sb.WriteString("\n\nHint: ")
		sb.WriteString(classified.Hint)
	}

	return sb.String()
}
