package validation

import (
	"testing"
)

func TestValidateScreenshotDimensions_Phone_Valid(t *testing.T) {
	result := ValidateScreenshotDimensions("phone", 1080, 1920)
	if result != nil {
		t.Errorf("expected nil for valid phone screenshot, got: %+v", result)
	}
}

func TestValidateScreenshotDimensions_Phone_MinWidth(t *testing.T) {
	result := ValidateScreenshotDimensions("phone", 320, 640)
	if result != nil {
		t.Errorf("expected nil for phone screenshot at min width, got: %+v", result)
	}
}

func TestValidateScreenshotDimensions_Phone_MaxWidth(t *testing.T) {
	result := ValidateScreenshotDimensions("phone", 3840, 1920)
	if result != nil {
		t.Errorf("expected nil for phone screenshot at max width, got: %+v", result)
	}
}

func TestValidateScreenshotDimensions_Phone_TooSmall(t *testing.T) {
	result := ValidateScreenshotDimensions("phone", 319, 640)
	if result == nil {
		t.Fatal("expected non-nil result for phone screenshot too small")
	}
	if result.Severity != SeverityError {
		t.Errorf("expected error severity, got %s", result.Severity)
	}
}

func TestValidateScreenshotDimensions_Phone_TooLarge(t *testing.T) {
	result := ValidateScreenshotDimensions("phone", 3841, 1920)
	if result == nil {
		t.Fatal("expected non-nil result for phone screenshot too large")
	}
	if result.Severity != SeverityError {
		t.Errorf("expected error severity, got %s", result.Severity)
	}
}

func TestValidateScreenshotDimensions_Phone_HeightTooSmall(t *testing.T) {
	result := ValidateScreenshotDimensions("phone", 640, 319)
	if result == nil {
		t.Fatal("expected non-nil result for phone screenshot height too small")
	}
	if result.Severity != SeverityError {
		t.Errorf("expected error severity, got %s", result.Severity)
	}
}

func TestValidateScreenshotDimensions_Phone_HeightTooLarge(t *testing.T) {
	result := ValidateScreenshotDimensions("phone", 1920, 3841)
	if result == nil {
		t.Fatal("expected non-nil result for phone screenshot height too large")
	}
	if result.Severity != SeverityError {
		t.Errorf("expected error severity, got %s", result.Severity)
	}
}

func TestValidateScreenshotDimensions_Phone_AspectRatioTooNarrow(t *testing.T) {
	// Aspect ratio 1:3 (too narrow, beyond 1:2)
	result := ValidateScreenshotDimensions("phone", 320, 961)
	if result == nil {
		t.Fatal("expected non-nil result for aspect ratio beyond 1:2")
	}
	if result.Severity != SeverityError {
		t.Errorf("expected error severity, got %s", result.Severity)
	}
}

func TestValidateScreenshotDimensions_Phone_AspectRatioTooWide(t *testing.T) {
	// Aspect ratio 3:1 (too wide, beyond 2:1)
	result := ValidateScreenshotDimensions("phone", 960, 320)
	if result == nil {
		t.Fatal("expected non-nil result for aspect ratio beyond 2:1")
	}
	if result.Severity != SeverityError {
		t.Errorf("expected error severity, got %s", result.Severity)
	}
}

func TestValidateScreenshotDimensions_Phone_AspectRatioExactly2to1(t *testing.T) {
	result := ValidateScreenshotDimensions("phone", 640, 320)
	if result != nil {
		t.Errorf("expected nil for aspect ratio exactly 2:1, got: %+v", result)
	}
}

func TestValidateScreenshotDimensions_Phone_AspectRatioExactly1to2(t *testing.T) {
	result := ValidateScreenshotDimensions("phone", 320, 640)
	if result != nil {
		t.Errorf("expected nil for aspect ratio exactly 1:2, got: %+v", result)
	}
}

func TestValidateScreenshotDimensions_Tablet7_Valid(t *testing.T) {
	result := ValidateScreenshotDimensions("tablet7", 1200, 1920)
	if result != nil {
		t.Errorf("expected nil for valid tablet 7 screenshot, got: %+v", result)
	}
}

func TestValidateScreenshotDimensions_Tablet7_TooSmall(t *testing.T) {
	result := ValidateScreenshotDimensions("tablet7", 319, 640)
	if result == nil {
		t.Fatal("expected non-nil result for tablet 7 screenshot too small")
	}
}

func TestValidateScreenshotDimensions_Tablet7_TooLarge(t *testing.T) {
	result := ValidateScreenshotDimensions("tablet7", 3841, 1920)
	if result == nil {
		t.Fatal("expected non-nil result for tablet 7 screenshot too large")
	}
}

func TestValidateScreenshotDimensions_Tablet10_Valid(t *testing.T) {
	result := ValidateScreenshotDimensions("tablet10", 1600, 2560)
	if result != nil {
		t.Errorf("expected nil for valid tablet 10 screenshot, got: %+v", result)
	}
}

func TestValidateScreenshotDimensions_Tablet10_TooSmall(t *testing.T) {
	result := ValidateScreenshotDimensions("tablet10", 319, 640)
	if result == nil {
		t.Fatal("expected non-nil result for tablet 10 screenshot too small")
	}
}

func TestValidateScreenshotDimensions_Tablet10_TooLarge(t *testing.T) {
	result := ValidateScreenshotDimensions("tablet10", 3841, 1920)
	if result == nil {
		t.Fatal("expected non-nil result for tablet 10 screenshot too large")
	}
}

func TestValidateScreenshotDimensions_ZeroDimensions(t *testing.T) {
	result := ValidateScreenshotDimensions("phone", 0, 0)
	if result == nil {
		t.Fatal("expected non-nil result for zero dimensions")
	}
}

func TestValidateScreenshotDimensions_NegativeDimensions(t *testing.T) {
	result := ValidateScreenshotDimensions("phone", -1, 640)
	if result == nil {
		t.Fatal("expected non-nil result for negative dimensions")
	}
}
