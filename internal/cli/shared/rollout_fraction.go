package shared

import (
	"fmt"
	"math"
)

// ValidateRolloutFraction checks a --rollout value. NaN needs its own check,
// because NaN < 0 and NaN > 1 are both false.
func ValidateRolloutFraction(fraction float64) error {
	if math.IsNaN(fraction) || fraction < 0 || fraction > 1 {
		return fmt.Errorf("--rollout must be between 0.0 and 1.0")
	}
	return nil
}
