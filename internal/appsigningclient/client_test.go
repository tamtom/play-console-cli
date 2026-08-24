package appsigningclient

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestEnrollApp_UsesOfficialDocumentedEndpoint(t *testing.T) {
	var gotPath, gotBody string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		body, _ := io.ReadAll(r.Body)
		gotBody = string(body)
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"signingCertificate":{"certificateHashSha256":"AA:BB"}}`)
	}))
	defer server.Close()

	client := New(server.Client(), server.URL+"/")
	response, err := client.EnrollApp(context.Background(), "dev.example.app", &EnrollAppRequest{
		EnrollExistingApp: &EnrollExistingApp{CloudKMSKey: &CloudKMSKey{CryptoKeyVersionResource: "projects/p/locations/l/keyRings/r/cryptoKeys/k/cryptoKeyVersions/1"}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if gotPath != "/androidpublisher/v3/applications/dev.example.app/appSigning:enrollApp" {
		t.Fatalf("path = %q", gotPath)
	}
	if !strings.Contains(gotBody, `"cryptoKeyVersionResource":"projects/p/`) {
		t.Fatalf("unexpected body: %s", gotBody)
	}
	if response.SigningCertificate.CertificateHashSHA256 != "AA:BB" {
		t.Fatalf("unexpected response: %#v", response)
	}
}

func TestRotateKey_UsesOfficialDocumentedEndpoint(t *testing.T) {
	var gotPath string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"rotatedKeyCertificate":{"certificateHashSha1":"11:22"}}`)
	}))
	defer server.Close()

	client := New(server.Client(), server.URL+"/")
	_, err := client.RotateAppSigningKey(context.Background(), "dev.example.app", &RotateAppSigningKeyRequest{
		KeyRotationReason: "ROUTINE_KEY_UPGRADE",
		RotatedCloudKMSKey: &RotatedCloudKMSKey{
			CloudKMSKeyAndCert:        &CloudKMSKeyAndCert{CloudKMSKey: &CloudKMSKey{CryptoKeyVersionResource: "projects/p/locations/l/keyRings/r/cryptoKeys/k/cryptoKeyVersions/2"}, PEMCertificate: "certificate"},
			SigningCertificateLineage: "lineage",
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if gotPath != "/androidpublisher/v3/applications/dev.example.app/appSigning:rotateAppSigningKey" {
		t.Fatalf("path = %q", gotPath)
	}
}
