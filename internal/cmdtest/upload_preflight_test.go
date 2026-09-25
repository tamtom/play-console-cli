package cmdtest_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestUploadCommandsRejectEmptyFilesBeforeAuthentication(t *testing.T) {
	emptyFile := filepath.Join(t.TempDir(), "empty.apk")
	if err := os.WriteFile(emptyFile, nil, 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("GPLAY_CONFIG_PATH", filepath.Join(t.TempDir(), "missing-config.json"))
	t.Setenv("GPLAY_SERVICE_ACCOUNT_JSON", "")
	t.Setenv("GPLAY_OAUTH_TOKEN_PATH", "")
	t.Setenv("GOOGLE_APPLICATION_CREDENTIALS", filepath.Join(t.TempDir(), "missing-credentials.json"))

	tests := []struct {
		name string
		args []string
	}{
		{name: "bundle", args: []string{"bundles", "upload", "--package", "com.example.app", "--edit", "edit-1", "--file", emptyFile}},
		{name: "APK", args: []string{"apks", "upload", "--package", "com.example.app", "--edit", "edit-1", "--file", emptyFile}},
		{name: "internal sharing APK", args: []string{"internal-sharing", "upload-apk", "--package", "com.example.app", "--file", emptyFile}},
		{name: "internal sharing bundle", args: []string{"internal-sharing", "upload-bundle", "--package", "com.example.app", "--file", emptyFile}},
		{name: "deobfuscation", args: []string{"deobfuscation", "upload", "--package", "com.example.app", "--edit", "edit-1", "--apk-version", "1", "--type", "proguard", "--file", emptyFile}},
		{name: "expansion", args: []string{"expansion", "upload", "--package", "com.example.app", "--edit", "edit-1", "--apk-version", "1", "--type", "main", "--file", emptyFile}},
		{name: "listing image", args: []string{"images", "upload", "--package", "com.example.app", "--edit", "edit-1", "--locale", "en-US", "--type", "phoneScreenshots", "--file", emptyFile}},
		{name: "Checks binary", args: []string{"checks", "upload", "--account", "accounts/test", "--app", "apps/test", "--binary", emptyFile}},
		{name: "custom app", args: []string{"custom-apps", "create", "--developer", "123", "--title", "Example", "--apk", emptyFile}},
		{name: "third-party store APK", args: []string{"app-stores", "upload-apk", "--app-store-package", "com.example.store", "--package", "com.example.app", "--file", emptyFile, "--registered-third-party-store", "--confirm"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, _, err := runCommand(t, tt.args...)
			if err == nil || !strings.Contains(err.Error(), "empty") {
				t.Fatalf("error = %v, want empty-file rejection", err)
			}
			if strings.Contains(err.Error(), "authentication") || strings.Contains(err.Error(), "credentials") {
				t.Fatalf("command attempted authentication before file validation: %v", err)
			}
		})
	}
}
