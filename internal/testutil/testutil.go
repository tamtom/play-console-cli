package testutil

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"os"
	"path/filepath"
	"testing"
)

// MockServiceAccount creates a fake but structurally valid service account JSON file.
func MockServiceAccount(t *testing.T) string {
	t.Helper()
	sa := map[string]string{
		"type":           "service_account",
		"project_id":     "test-project",
		"private_key_id": "key123",
		"private_key":    "-----BEGIN RSA PRIVATE KEY-----\nMIIBogIBAAJBALRiMLAHudeSA/x3hB2f+2NRkJlBEBGoviKrswxMa0sNHBEJTGcb\nkz9/M7FjpSBiVkGBPIxZKylFSk693mUCAwEAAQJAZ6bUJeMhNgDJOuMFsGN2IyGX\nmEaSPLbSJMiJQBGHGYyhE0PBuSl7SgPgyEBJjRTDlHBuC6gyIa3FqxhyzNP7gQIh\nAOHqJBE1YMeitHv/GERNIKc0dCtMJPAAthGvjhSVFILRAiEAzCAuaJOIFpUm6NUM\nT3qlV8kJOEGzMbPFjGaVwPBuysUCIGbqMhE3T5Rj/MBFBiaNjSEt6JFj01l2g0Ev\nCEO+aNYRAiEAu4ETcKMiJp9BbCJHr0MBqEBb71FaDjX11YMq52wHSGkCIDRATuFg\ntgb2ArFe6t+vg0mJe0dCVHlGRv1jGkRoX+4q\n-----END RSA PRIVATE KEY-----\n",
		"client_email":   "test@test-project.iam.gserviceaccount.com",
		"client_id":      "123456789",
		"auth_uri":       "https://accounts.google.com/o/oauth2/auth",
		"token_uri":      "https://oauth2.googleapis.com/token",
	}
	data, err := json.MarshalIndent(sa, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	path := filepath.Join(dir, "service-account.json")
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}

// SandboxServiceAccount writes a service-account JSON file whose private key
// is a freshly generated, valid RSA key and whose token_uri points at the
// given URL. The OAuth JWT flow then signs locally and fetches its token from
// a local sandbox server, so authentication works fully offline.
func SandboxServiceAccount(t *testing.T, tokenURL string) string {
	t.Helper()
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	keyPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(key),
	})
	sa := map[string]string{
		"type":         "service_account",
		"project_id":   "sandbox-project",
		"private_key":  string(keyPEM),
		"client_email": "sandbox@sandbox-project.iam.gserviceaccount.com",
		"client_id":    "0",
		"token_uri":    tokenURL,
	}
	data, err := json.MarshalIndent(sa, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "sandbox-service-account.json")
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}
