package validation

import "fmt"

// Screenshot dimension limits for Google Play.
const (
	MinScreenshotDimension = 320
	MaxScreenshotDimension = 3840
)

// ValidateScreenshotDimensions checks that screenshot dimensions are within
// the allowed range and aspect ratio for the given device type.
// Device types: "phone", "tablet7", "tablet10".
// Phone: min 320px, max 3840px, aspect ratio between 1:2 and 2:1.
// Tablet 7": min 320px, max 3840px.
// Tablet 10": min 320px, max 3840px.
// Returns nil if dimensions are valid.
func ValidateScreenshotDimensions(deviceType string, width, height int) *CheckResult {
	id := fmt.Sprintf("screenshot-%s-dimensions", deviceType)

	// Check for non-positive dimensions
	if width <= 0 || height <= 0 {
		return &CheckResult{
			ID:          id,
			Severity:    SeverityError,
			Field:       "screenshot",
			Message:     fmt.Sprintf("Invalid dimensions %dx%d: width and height must be positive", width, height),
			Remediation: fmt.Sprintf("Use dimensions between %dpx and %dpx", MinScreenshotDimension, MaxScreenshotDimension),
		}
	}

	// Check width bounds
	if width < MinScreenshotDimension || width > MaxScreenshotDimension {
		return &CheckResult{
			ID:          id,
			Severity:    SeverityError,
			Field:       "screenshot",
			Message:     fmt.Sprintf("Width %dpx is out of range for %s (min: %dpx, max: %dpx)", width, deviceType, MinScreenshotDimension, MaxScreenshotDimension),
			Remediation: fmt.Sprintf("Resize the screenshot width to be between %dpx and %dpx", MinScreenshotDimension, MaxScreenshotDimension),
		}
	}

	// Check height bounds
	if height < MinScreenshotDimension || height > MaxScreenshotDimension {
		return &CheckResult{
			ID:          id,
			Severity:    SeverityError,
			Field:       "screenshot",
			Message:     fmt.Sprintf("Height %dpx is out of range for %s (min: %dpx, max: %dpx)", height, deviceType, MinScreenshotDimension, MaxScreenshotDimension),
			Remediation: fmt.Sprintf("Resize the screenshot height to be between %dpx and %dpx", MinScreenshotDimension, MaxScreenshotDimension),
		}
	}

	// Check aspect ratio for phone screenshots (between 1:2 and 2:1)
	if deviceType == "phone" {
		ratio := float64(width) / float64(height)
		if ratio < 0.5 || ratio > 2.0 {
			return &CheckResult{
				ID:          id,
				Severity:    SeverityError,
				Field:       "screenshot",
				Message:     fmt.Sprintf("Aspect ratio %.2f:1 for %dx%d is out of range for %s (must be between 1:2 and 2:1)", ratio, width, height, deviceType),
				Remediation: "Resize the screenshot so the aspect ratio is between 1:2 and 2:1",
			}
		}
	}

	return nil
}
