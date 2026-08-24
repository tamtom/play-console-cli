package appsigning

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/tamtom/play-console-cli/internal/appsigningclient"
	"github.com/tamtom/play-console-cli/internal/cli/shared"
	"github.com/tamtom/play-console-cli/internal/rootfs"
)

const existingAppRequest = `{"enrollExistingApp":{"cloudKmsKey":{"cryptoKeyVersionResource":"projects/p/locations/l/keyRings/r/cryptoKeys/k/cryptoKeyVersions/1"}}}`

const rotationRequest = `{"keyRotationReason":"ROUTINE_KEY_UPGRADE","rotatedCloudKmsKey":{"cloudKmsKeyAndCert":{"cloudKmsKey":{"cryptoKeyVersionResource":"projects/p/locations/l/keyRings/r/cryptoKeys/k/cryptoKeyVersions/2"},"pemCertificate":"BASE64_PEM"},"signingCertificateLineage":"BASE64_LINEAGE"}}`

func TestEnrollPlanIsSealedAndNeverAuthenticates(t *testing.T) {
	planPath := filepath.Join(t.TempDir(), "enroll-plan.json")
	authCalls := 0
	ctx := appsigningclient.ContextWithServiceFactory(context.Background(), func(context.Context) (*appsigningclient.Service, error) {
		authCalls++
		return nil, errors.New("authentication must not run while planning")
	})
	command := EnrollPlanCommand()
	if err := command.Parse([]string{
		"--package", "app.example", "--json", existingAppRequest,
		"--enterprise-self-hosted-kms", "--plan-file", planPath,
	}); err != nil {
		t.Fatal(err)
	}
	if err := command.Run(shared.ContextWithIO(ctx, &bytes.Buffer{}, &bytes.Buffer{})); err != nil {
		t.Fatalf("plan: %v", err)
	}
	if authCalls != 0 {
		t.Fatalf("authentication calls = %d", authCalls)
	}
	plan := readSigningPlanForTest(t, planPath)
	if plan.Provider != "official-api" || plan.Operation != operationEnroll || plan.Package != "app.example" || plan.PlanHash == "" || plan.RequestSHA256 == "" {
		t.Fatalf("incomplete plan: %#v", plan)
	}
	if expected, err := signingPlanHash(plan); err != nil || expected != plan.PlanHash {
		t.Fatalf("plan hash = %q, expected %q, error %v", plan.PlanHash, expected, err)
	}
}

func TestApplyRequiresExactPlanHashBeforeAuthentication(t *testing.T) {
	planPath := createSigningPlanForTest(t)
	authCalls := 0
	ctx := appsigningclient.ContextWithServiceFactory(context.Background(), func(context.Context) (*appsigningclient.Service, error) {
		authCalls++
		return nil, errors.New("must not authenticate")
	})
	command := ApplyCommand()
	if err := command.Parse([]string{
		"--plan-file", planPath, "--receipt-file", filepath.Join(t.TempDir(), "receipt.json"),
		"--confirm-plan", strings.Repeat("0", 64), "--enterprise-self-hosted-kms",
	}); err != nil {
		t.Fatal(err)
	}
	err := command.Run(ctx)
	if err == nil || !strings.Contains(err.Error(), "must exactly match") {
		t.Fatalf("error = %v", err)
	}
	if authCalls != 0 {
		t.Fatalf("authentication calls = %d", authCalls)
	}
}

func TestApplyPersistsSealedReceiptAndDoesNotReplay(t *testing.T) {
	planPath := createSigningPlanForTest(t)
	plan := readSigningPlanForTest(t, planPath)
	receiptPath := filepath.Join(t.TempDir(), "receipt.json")
	posts := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		posts++
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"signingCertificate":{"certificateHashSha256":"AA:BB"}}`))
	}))
	t.Cleanup(server.Close)
	ctx := appsigningclient.ContextWithServiceFactory(context.Background(), func(context.Context) (*appsigningclient.Service, error) {
		return appsigningclient.NewServiceWithClient(server.Client(), server.URL+"/", nil), nil
	})
	for attempt := 0; attempt < 2; attempt++ {
		command := ApplyCommand()
		if err := command.Parse([]string{
			"--plan-file", planPath, "--receipt-file", receiptPath,
			"--confirm-plan", plan.PlanHash, "--enterprise-self-hosted-kms",
		}); err != nil {
			t.Fatal(err)
		}
		if err := command.Run(shared.ContextWithIO(ctx, &bytes.Buffer{}, &bytes.Buffer{})); err != nil {
			t.Fatalf("apply attempt %d: %v", attempt+1, err)
		}
	}
	if posts != 1 {
		t.Fatalf("POST requests = %d, want 1", posts)
	}
	receipt := readSigningReceiptForTest(t, receiptPath)
	if receipt.Status != receiptComplete || receipt.PlanHash != plan.PlanHash || receipt.ReceiptHash == "" {
		t.Fatalf("receipt = %#v", receipt)
	}
	if expected, err := signingReceiptHash(receipt); err != nil || expected != receipt.ReceiptHash {
		t.Fatalf("receipt hash = %q, expected %q, error %v", receipt.ReceiptHash, expected, err)
	}
}

func TestAmbiguousApplyNeverReplays(t *testing.T) {
	planPath := createSigningPlanForTest(t)
	plan := readSigningPlanForTest(t, planPath)
	receiptPath := filepath.Join(t.TempDir(), "receipt.json")
	requests := 0
	client := &http.Client{Transport: roundTripperFunc(func(*http.Request) (*http.Response, error) {
		requests++
		return nil, errors.New("connection reset after request write")
	})}
	ctx := appsigningclient.ContextWithServiceFactory(context.Background(), func(context.Context) (*appsigningclient.Service, error) {
		return appsigningclient.NewServiceWithClient(client, "https://example.invalid/", nil), nil
	})
	apply := func() error {
		command := ApplyCommand()
		if err := command.Parse([]string{
			"--plan-file", planPath, "--receipt-file", receiptPath,
			"--confirm-plan", plan.PlanHash, "--enterprise-self-hosted-kms",
		}); err != nil {
			t.Fatal(err)
		}
		return command.Run(shared.ContextWithIO(ctx, &bytes.Buffer{}, &bytes.Buffer{}))
	}
	if err := apply(); err == nil || !strings.Contains(err.Error(), "ambiguous") {
		t.Fatalf("first apply error = %v", err)
	}
	if err := apply(); err == nil || !strings.Contains(err.Error(), "must not be replayed") {
		t.Fatalf("second apply error = %v", err)
	}
	if requests != 1 {
		t.Fatalf("requests = %d, want 1", requests)
	}
}

func TestRotatePlanRejectsUnknownReasonBeforeWriting(t *testing.T) {
	planPath := filepath.Join(t.TempDir(), "rotation-plan.json")
	command := RotatePlanCommand()
	if err := command.Parse([]string{
		"--package", "app.example", "--json", `{"keyRotationReason":"BECAUSE"}`,
		"--enterprise-self-hosted-kms", "--plan-file", planPath,
	}); err != nil {
		t.Fatal(err)
	}
	err := command.Run(context.Background())
	if err == nil || !strings.Contains(err.Error(), "keyRotationReason") {
		t.Fatalf("error = %v", err)
	}
	if _, err := os.Stat(planPath); !os.IsNotExist(err) {
		t.Fatalf("invalid plan was written: %v", err)
	}
}

func TestEveryAppSigningEndpoint_SuccessAndRejectedResponse(t *testing.T) {
	tests := []struct {
		name     string
		plan     func(*testing.T) string
		wantPath string
	}{
		{"enroll", createSigningPlanForTest, "/androidpublisher/v3/applications/app.example/appSigning:enrollApp"},
		{"rotate", createRotationPlanForTest, "/androidpublisher/v3/applications/app.example/appSigning:rotateAppSigningKey"},
	}
	for _, tc := range tests {
		for _, status := range []int{http.StatusOK, http.StatusForbidden} {
			caseName := "success"
			if status != http.StatusOK {
				caseName = "rejected"
			}
			t.Run(tc.name+" "+caseName, func(t *testing.T) {
				planPath := tc.plan(t)
				plan := readSigningPlanForTest(t, planPath)
				receiptPath := filepath.Join(t.TempDir(), "receipt.json")
				requests := 0
				server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					requests++
					if r.URL.Path != tc.wantPath {
						t.Errorf("path = %q, want %q", r.URL.Path, tc.wantPath)
					}
					w.Header().Set("Content-Type", "application/json")
					w.WriteHeader(status)
					if status == http.StatusOK {
						_, _ = w.Write([]byte(`{}`))
						return
					}
					_, _ = w.Write([]byte(`{"error":{"message":"denied"}}`))
				}))
				t.Cleanup(server.Close)
				ctx := appsigningclient.ContextWithServiceFactory(context.Background(), func(context.Context) (*appsigningclient.Service, error) {
					return appsigningclient.NewServiceWithClient(server.Client(), server.URL+"/", nil), nil
				})
				apply := func() error {
					command := ApplyCommand()
					if err := command.Parse([]string{"--plan-file", planPath, "--receipt-file", receiptPath, "--confirm-plan", plan.PlanHash, "--enterprise-self-hosted-kms"}); err != nil {
						t.Fatal(err)
					}
					return command.Run(shared.ContextWithIO(ctx, &bytes.Buffer{}, &bytes.Buffer{}))
				}
				err := apply()
				if status == http.StatusOK {
					if err != nil {
						t.Fatal(err)
					}
					if receipt := readSigningReceiptForTest(t, receiptPath); receipt.Status != receiptComplete {
						t.Fatalf("receipt status = %q", receipt.Status)
					}
					return
				}
				if err == nil || !strings.Contains(err.Error(), "rejected") {
					t.Fatalf("rejected apply error = %v", err)
				}
				if receipt := readSigningReceiptForTest(t, receiptPath); receipt.Status != receiptRejected {
					t.Fatalf("receipt status = %q", receipt.Status)
				}
				if err := apply(); err == nil || !strings.Contains(err.Error(), "create a new plan") {
					t.Fatalf("replay error = %v", err)
				}
				if requests != 1 {
					t.Fatalf("requests = %d, want 1", requests)
				}
			})
		}
	}
}

func TestAppSigningServerFailureIsAmbiguousAndNeverReplayed(t *testing.T) {
	planPath := createSigningPlanForTest(t)
	plan := readSigningPlanForTest(t, planPath)
	receiptPath := filepath.Join(t.TempDir(), "receipt.json")
	requests := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		requests++
		http.Error(w, `{"error":{"message":"backend timeout"}}`, http.StatusInternalServerError)
	}))
	t.Cleanup(server.Close)
	ctx := appsigningclient.ContextWithServiceFactory(context.Background(), func(context.Context) (*appsigningclient.Service, error) {
		return appsigningclient.NewServiceWithClient(server.Client(), server.URL+"/", nil), nil
	})
	for attempt := 0; attempt < 2; attempt++ {
		_, err := applySigningPlan(ctx, planPath, receiptPath, plan.PlanHash)
		if err == nil || !strings.Contains(err.Error(), "ambiguous") {
			t.Fatalf("attempt %d error = %v", attempt+1, err)
		}
	}
	if requests != 1 {
		t.Fatalf("requests = %d, want 1", requests)
	}
	if receipt := readSigningReceiptForTest(t, receiptPath); receipt.Status != receiptAmbiguous {
		t.Fatalf("receipt status = %q", receipt.Status)
	}
}

func TestConcurrentAppSigningApplyReservesReceiptBeforeAuthentication(t *testing.T) {
	planPath := createSigningPlanForTest(t)
	plan := readSigningPlanForTest(t, planPath)
	receiptPath := filepath.Join(t.TempDir(), "receipt.json")
	var requests atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		requests.Add(1)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{}`))
	}))
	t.Cleanup(server.Close)
	var serviceCalls atomic.Int32
	ctx := appsigningclient.ContextWithServiceFactory(context.Background(), func(context.Context) (*appsigningclient.Service, error) {
		serviceCalls.Add(1)
		return appsigningclient.NewServiceWithClient(server.Client(), server.URL+"/", nil), nil
	})
	filesystem := &barrierFilesystem{receiptPath: receiptPath, ready: make(chan struct{}, 2), release: make(chan struct{})}
	ctx = shared.ContextWithFilesystem(ctx, filesystem)
	results := make(chan error, 2)
	for range 2 {
		go func() {
			_, err := applySigningPlan(ctx, planPath, receiptPath, plan.PlanHash)
			results <- err
		}()
	}
	<-filesystem.ready
	<-filesystem.ready
	close(filesystem.release)
	var succeeded, reserved int
	for range 2 {
		err := <-results
		if err == nil {
			succeeded++
		} else if strings.Contains(err.Error(), "already reserved") {
			reserved++
		} else {
			t.Fatalf("unexpected apply error: %v", err)
		}
	}
	if succeeded != 1 || reserved != 1 || requests.Load() != 1 || serviceCalls.Load() != 1 {
		t.Fatalf("success=%d reserved=%d requests=%d serviceCalls=%d", succeeded, reserved, requests.Load(), serviceCalls.Load())
	}
}

type barrierFilesystem struct {
	receiptPath string
	reads       atomic.Int32
	ready       chan struct{}
	release     chan struct{}
}

func (f *barrierFilesystem) ReadFile(path string) ([]byte, error) {
	if path == f.receiptPath && f.reads.Add(1) <= 2 {
		f.ready <- struct{}{}
		<-f.release
		return nil, os.ErrNotExist
	}
	return rootfs.ReadFile(path)
}

func (*barrierFilesystem) AtomicWriteFile(path string, data []byte, fileMode, dirMode os.FileMode) error {
	return rootfs.AtomicWriteFile(path, data, fileMode, dirMode)
}

func (*barrierFilesystem) CreateExclusiveFile(path string, data []byte, fileMode, dirMode os.FileMode) error {
	return rootfs.CreateExclusiveFile(path, data, fileMode, dirMode)
}

func createSigningPlanForTest(t *testing.T) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "plan.json")
	command := EnrollPlanCommand()
	if err := command.Parse([]string{
		"--package", "app.example", "--json", existingAppRequest,
		"--enterprise-self-hosted-kms", "--plan-file", path,
	}); err != nil {
		t.Fatal(err)
	}
	if err := command.Run(shared.ContextWithIO(context.Background(), &bytes.Buffer{}, &bytes.Buffer{})); err != nil {
		t.Fatal(err)
	}
	return path
}

func createRotationPlanForTest(t *testing.T) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "rotation-plan.json")
	command := RotatePlanCommand()
	if err := command.Parse([]string{
		"--package", "app.example", "--json", rotationRequest,
		"--enterprise-self-hosted-kms", "--plan-file", path,
	}); err != nil {
		t.Fatal(err)
	}
	if err := command.Run(shared.ContextWithIO(context.Background(), &bytes.Buffer{}, &bytes.Buffer{})); err != nil {
		t.Fatal(err)
	}
	return path
}

func readSigningPlanForTest(t *testing.T, path string) *signingPlan {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var value signingPlan
	if err := json.Unmarshal(data, &value); err != nil {
		t.Fatal(err)
	}
	return &value
}

func readSigningReceiptForTest(t *testing.T, path string) *signingReceipt {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var value signingReceipt
	if err := json.Unmarshal(data, &value); err != nil {
		t.Fatal(err)
	}
	return &value
}

type roundTripperFunc func(*http.Request) (*http.Response, error)

func (f roundTripperFunc) RoundTrip(request *http.Request) (*http.Response, error) {
	return f(request)
}
