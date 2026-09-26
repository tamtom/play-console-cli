package shared

import (
	"strings"
	"testing"

	"google.golang.org/api/androidpublisher/v3"
)

func TestLoadJSONArgRejectsFieldsTheSDKWouldDrop(t *testing.T) {
	var req androidpublisher.UpdateAppStoreHostedAppRequest
	err := LoadJSONArg(`{"packageName":"com.example.test","activeApks":{"activeApkSets":[{"versionCode":"42"}]}}`, &req)
	if err == nil || !strings.Contains(err.Error(), "unknown field") {
		t.Fatalf("unknown SDK field silently dropped: %v", err)
	}
}
