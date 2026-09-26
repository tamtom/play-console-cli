package testutil

import (
	"slices"
	"testing"
)

func TestOfficialEnum_SplitsDeprecatedValues(t *testing.T) {
	current, deprecated := OfficialEnum(t, "androidpublisher", "Grant", "appLevelPermissions")
	if !slices.Contains(current, "CAN_VIEW_FINANCIAL_DATA") {
		t.Errorf("current = %v, want CAN_VIEW_FINANCIAL_DATA", current)
	}
	if !slices.Contains(deprecated, "CAN_ACCESS_APP") {
		t.Errorf("deprecated = %v, want CAN_ACCESS_APP", deprecated)
	}
	for _, value := range append(current, deprecated...) {
		if value == "APP_LEVEL_PERMISSION_UNSPECIFIED" {
			t.Errorf("OfficialEnum returned %s", value)
		}
	}
}

func TestAssertHelpListsEnum_MatchesWholeWords(t *testing.T) {
	if containsWord("CAN_VIEW_FINANCIAL_DATA_GLOBAL", "CAN_VIEW_FINANCIAL_DATA") {
		t.Error("containsWord matched a prefix of a longer value")
	}
	if !containsWord("  - CAN_VIEW_FINANCIAL_DATA: View", "CAN_VIEW_FINANCIAL_DATA") {
		t.Error("containsWord did not match a listed value")
	}
}
