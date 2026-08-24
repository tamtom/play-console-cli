package sync

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/tamtom/play-console-cli/internal/cli/shared"
	"github.com/tamtom/play-console-cli/internal/playclient"
	"github.com/tamtom/play-console-cli/internal/rootfs"
)

func TestPlanCommandWritesContentAddressedOfficialAPIPlan(t *testing.T) {
	metadataDir := t.TempDir()
	localeDir := filepath.Join(metadataDir, "en-US")
	if err := os.MkdirAll(localeDir, 0o755); err != nil {
		t.Fatalf("mkdir locale: %v", err)
	}
	for name, content := range map[string]string{
		titleFile:     "New title\n",
		shortDescFile: "Short description\n",
		fullDescFile:  "Full description\n",
	} {
		if err := os.WriteFile(filepath.Join(localeDir, name), []byte(content), 0o644); err != nil {
			t.Fatalf("write %s: %v", name, err)
		}
	}

	var mutations int
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			mutations++
		}
		w.Header().Set("Content-Type", "application/json")
		if isEditGet(r) {
			_, _ = w.Write([]byte(`{"id":"edit-1","expiryTimeSeconds":"4102444800"}`))
			return
		}
		_, _ = w.Write([]byte(`{"listings":[{"language":"en-US","title":"Old title","shortDescription":"Short description","fullDescription":"Full description"}]}`))
	}))
	t.Cleanup(server.Close)

	ctx := playclient.ContextWithServiceFactory(context.Background(), func(ctx context.Context) (*playclient.Service, error) {
		return playclient.NewServiceWithClient(ctx, server.Client(), server.URL+"/")
	})
	planPath := filepath.Join(t.TempDir(), "plan.json")
	cmd := PlanCommand()
	if err := cmd.FlagSet.Parse([]string{
		"--package", "com.example.app",
		"--edit", "edit-1",
		"--dir", metadataDir,
		"--plan-file", planPath,
	}); err != nil {
		t.Fatalf("parse flags: %v", err)
	}
	if err := cmd.Exec(ctx, nil); err != nil {
		t.Fatalf("plan command: %v", err)
	}
	if mutations != 0 {
		t.Fatalf("plan made %d mutation requests", mutations)
	}

	data, err := os.ReadFile(planPath)
	if err != nil {
		t.Fatalf("read plan: %v", err)
	}
	var artifact struct {
		Version           int      `json:"version"`
		Provider          string   `json:"provider"`
		Package           string   `json:"package"`
		EditID            string   `json:"editId"`
		SourceFingerprint string   `json:"sourceFingerprint"`
		PlanHash          string   `json:"planHash"`
		ExpiresAt         string   `json:"expiresAt"`
		DestructiveCount  int      `json:"destructiveCount"`
		CapabilityIDs     []string `json:"capabilityIds"`
		Effects           []struct {
			ID        string `json:"id"`
			Kind      string `json:"kind"`
			Locale    string `json:"locale"`
			OldSHA256 string `json:"oldSha256"`
			NewSHA256 string `json:"newSha256"`
		} `json:"effects"`
	}
	if err := json.Unmarshal(data, &artifact); err != nil {
		t.Fatalf("decode plan: %v", err)
	}
	if artifact.Version != 1 || artifact.Provider != "official-api" {
		t.Fatalf("unexpected plan contract: %#v", artifact)
	}
	if artifact.ExpiresAt == "" || artifact.DestructiveCount != 0 || len(artifact.CapabilityIDs) != 2 {
		t.Fatalf("plan safety metadata is incomplete: %#v", artifact)
	}
	if artifact.Package != "com.example.app" || artifact.EditID != "edit-1" {
		t.Fatalf("unexpected target: %#v", artifact)
	}
	if len(artifact.Effects) != 1 || artifact.Effects[0].Kind != "listing.update" || artifact.Effects[0].Locale != "en-US" {
		t.Fatalf("unexpected effects: %#v", artifact.Effects)
	}
	shaPattern := regexp.MustCompile(`^[0-9a-f]{64}$`)
	for name, value := range map[string]string{
		"sourceFingerprint": artifact.SourceFingerprint,
		"planHash":          artifact.PlanHash,
		"effect.id":         artifact.Effects[0].ID,
		"effect.oldSha256":  artifact.Effects[0].OldSHA256,
		"effect.newSha256":  artifact.Effects[0].NewSHA256,
	} {
		if !shaPattern.MatchString(value) {
			t.Errorf("%s = %q, want SHA-256", name, value)
		}
	}
}

func TestApplyCommandPersistsReceiptAndDoesNotReplayCompletedEffect(t *testing.T) {
	metadataDir := writeTestListing(t, "New title")
	currentTitle := "Old title"
	updates := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if writeTestEdit(w, r) {
			return
		}
		switch r.Method {
		case http.MethodGet:
			if filepath.Base(r.URL.Path) == "en-US" {
				_, _ = w.Write([]byte(`{"language":"en-US","title":"` + currentTitle + `","shortDescription":"Short description","fullDescription":"Full description"}`))
				return
			}
			_, _ = w.Write([]byte(`{"listings":[{"language":"en-US","title":"` + currentTitle + `","shortDescription":"Short description","fullDescription":"Full description"}]}`))
		case http.MethodPut:
			body, err := io.ReadAll(r.Body)
			if err != nil {
				t.Errorf("read update body: %v", err)
			}
			var listing struct {
				Title string `json:"title"`
			}
			if err := json.Unmarshal(body, &listing); err != nil {
				t.Errorf("decode update body: %v", err)
			}
			updates++
			currentTitle = listing.Title
			_, _ = w.Write([]byte(`{"language":"en-US","title":"` + currentTitle + `","shortDescription":"Short description","fullDescription":"Full description"}`))
		default:
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
	}))
	t.Cleanup(server.Close)
	ctx := playclient.ContextWithServiceFactory(context.Background(), func(ctx context.Context) (*playclient.Service, error) {
		return playclient.NewServiceWithClient(ctx, server.Client(), server.URL+"/")
	})

	stateDir := t.TempDir()
	planPath := filepath.Join(stateDir, "plan.json")
	planCommand := PlanCommand()
	if err := planCommand.FlagSet.Parse([]string{"--package", "com.example.app", "--edit", "edit-1", "--dir", metadataDir, "--plan-file", planPath}); err != nil {
		t.Fatalf("parse plan flags: %v", err)
	}
	if err := planCommand.Exec(ctx, nil); err != nil {
		t.Fatalf("create plan: %v", err)
	}

	receiptPath := filepath.Join(stateDir, "receipt.json")
	for attempt := 1; attempt <= 2; attempt++ {
		applyCommand := ApplyCommand()
		if err := applyCommand.FlagSet.Parse([]string{"--plan-file", planPath, "--receipt-file", receiptPath}); err != nil {
			t.Fatalf("parse apply flags on attempt %d: %v", attempt, err)
		}
		if err := applyCommand.Exec(ctx, nil); err != nil {
			t.Fatalf("apply attempt %d: %v", attempt, err)
		}
	}
	if updates != 1 {
		t.Fatalf("official listing update requests = %d, want exactly 1", updates)
	}

	data, err := os.ReadFile(receiptPath)
	if err != nil {
		t.Fatalf("read receipt: %v", err)
	}
	var receipt struct {
		Status  string `json:"status"`
		Effects []struct {
			ID     string `json:"id"`
			Status string `json:"status"`
		} `json:"effects"`
	}
	if err := json.Unmarshal(data, &receipt); err != nil {
		t.Fatalf("decode receipt: %v", err)
	}
	if receipt.Status != "complete" || len(receipt.Effects) != 1 || receipt.Effects[0].Status != "applied" {
		t.Fatalf("unexpected receipt: %#v", receipt)
	}
}

func TestPlanCommandIncludesOnlyMissingImageUploads(t *testing.T) {
	metadataDir := writeTestListing(t, "Current title")
	imageDir := filepath.Join(metadataDir, "en-US", imagesDir, phoneScreenshotsDir)
	if err := os.MkdirAll(imageDir, 0o755); err != nil {
		t.Fatalf("mkdir images: %v", err)
	}
	if err := os.WriteFile(filepath.Join(imageDir, "01.png"), []byte("new-image"), 0o644); err != nil {
		t.Fatalf("write image: %v", err)
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if writeTestEdit(w, r) {
			return
		}
		if filepath.Base(r.URL.Path) == "listings" {
			_, _ = w.Write([]byte(`{"listings":[{"language":"en-US","title":"Current title","shortDescription":"Short description","fullDescription":"Full description"}]}`))
			return
		}
		_, _ = w.Write([]byte(`{"images":[{"id":"old","sha256":"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"}]}`))
	}))
	t.Cleanup(server.Close)
	ctx := playclient.ContextWithServiceFactory(context.Background(), func(ctx context.Context) (*playclient.Service, error) {
		return playclient.NewServiceWithClient(ctx, server.Client(), server.URL+"/")
	})

	planPath := filepath.Join(t.TempDir(), "plan.json")
	command := PlanCommand()
	if err := command.FlagSet.Parse([]string{"--package", "com.example.app", "--edit", "edit-1", "--dir", metadataDir, "--plan-file", planPath}); err != nil {
		t.Fatalf("parse plan flags: %v", err)
	}
	if err := command.Exec(ctx, nil); err != nil {
		t.Fatalf("create plan: %v", err)
	}
	data, err := os.ReadFile(planPath)
	if err != nil {
		t.Fatalf("read plan: %v", err)
	}
	var artifact syncPlan
	if err := json.Unmarshal(data, &artifact); err != nil {
		t.Fatalf("decode plan: %v", err)
	}
	if len(artifact.Effects) != 1 {
		t.Fatalf("effects = %#v, want one image upload", artifact.Effects)
	}
	effect := artifact.Effects[0]
	if effect.Kind != "image.upload" || effect.Locale != "en-US" || effect.ImageType != "phoneScreenshots" || effect.RelativePath != "en-US/images/phoneScreenshots/01.png" {
		t.Fatalf("unexpected image effect: %#v", effect)
	}
	if artifact.SourceFingerprint == "" || effect.OldSHA256 == "" || effect.NewSHA256 != sha256Hex([]byte("new-image")) {
		t.Fatalf("image effect is not content addressed: %#v", effect)
	}
}

func TestApplyCommandUploadsImageOnceAndResumesFromReceipt(t *testing.T) {
	metadataDir := writeTestListing(t, "Current title")
	imageDir := filepath.Join(metadataDir, "en-US", imagesDir, phoneScreenshotsDir)
	if err := os.MkdirAll(imageDir, 0o755); err != nil {
		t.Fatalf("mkdir images: %v", err)
	}
	imageBytes := []byte("new-image")
	if err := os.WriteFile(filepath.Join(imageDir, "01.png"), imageBytes, 0o644); err != nil {
		t.Fatalf("write image: %v", err)
	}
	imageSHA := sha256Hex(imageBytes)
	remoteImages := false
	uploads := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if writeTestEdit(w, r) {
			return
		}
		switch r.Method {
		case http.MethodGet:
			if filepath.Base(r.URL.Path) == "listings" {
				_, _ = w.Write([]byte(`{"listings":[{"language":"en-US","title":"Current title","shortDescription":"Short description","fullDescription":"Full description"}]}`))
				return
			}
			if remoteImages {
				_, _ = w.Write([]byte(`{"images":[{"id":"new","sha256":"` + imageSHA + `"}]}`))
				return
			}
			_, _ = w.Write([]byte(`{"images":[]}`))
		case http.MethodPost:
			body, err := io.ReadAll(r.Body)
			if err != nil {
				t.Errorf("read upload body: %v", err)
			}
			if !strings.Contains(string(body), "new-image") {
				t.Errorf("upload body does not contain image bytes")
			}
			uploads++
			remoteImages = true
			_, _ = w.Write([]byte(`{"image":{"id":"new","sha256":"` + imageSHA + `"}}`))
		default:
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
	}))
	t.Cleanup(server.Close)
	ctx := playclient.ContextWithServiceFactory(context.Background(), func(ctx context.Context) (*playclient.Service, error) {
		return playclient.NewServiceWithClient(ctx, server.Client(), server.URL+"/")
	})

	stateDir := t.TempDir()
	planPath := filepath.Join(stateDir, "plan.json")
	planCommand := PlanCommand()
	if err := planCommand.FlagSet.Parse([]string{"--package", "com.example.app", "--edit", "edit-1", "--dir", metadataDir, "--plan-file", planPath}); err != nil {
		t.Fatalf("parse plan flags: %v", err)
	}
	if err := planCommand.Exec(ctx, nil); err != nil {
		t.Fatalf("create plan: %v", err)
	}
	receiptPath := filepath.Join(stateDir, "receipt.json")
	for attempt := 1; attempt <= 2; attempt++ {
		command := ApplyCommand()
		if err := command.FlagSet.Parse([]string{"--plan-file", planPath, "--receipt-file", receiptPath}); err != nil {
			t.Fatalf("parse apply flags: %v", err)
		}
		if err := command.Exec(ctx, nil); err != nil {
			t.Fatalf("apply attempt %d: %v", attempt, err)
		}
	}
	if uploads != 1 {
		t.Fatalf("official image upload requests = %d, want exactly 1", uploads)
	}
}

func TestRunCommandPlansAppliesAndPersistsStateArtifacts(t *testing.T) {
	metadataDir := writeTestListing(t, "New title")
	currentTitle := "Old title"
	updates := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if writeTestEdit(w, r) {
			return
		}
		switch r.Method {
		case http.MethodGet:
			if filepath.Base(r.URL.Path) == "en-US" {
				_, _ = w.Write([]byte(`{"language":"en-US","title":"` + currentTitle + `","shortDescription":"Short description","fullDescription":"Full description"}`))
				return
			}
			_, _ = w.Write([]byte(`{"listings":[{"language":"en-US","title":"` + currentTitle + `","shortDescription":"Short description","fullDescription":"Full description"}]}`))
		case http.MethodPut:
			updates++
			currentTitle = "New title"
			_, _ = w.Write([]byte(`{"language":"en-US","title":"New title","shortDescription":"Short description","fullDescription":"Full description"}`))
		}
	}))
	t.Cleanup(server.Close)
	ctx := playclient.ContextWithServiceFactory(context.Background(), func(ctx context.Context) (*playclient.Service, error) {
		return playclient.NewServiceWithClient(ctx, server.Client(), server.URL+"/")
	})

	stateDir := t.TempDir()
	command := RunCommand()
	if err := command.FlagSet.Parse([]string{"--package", "com.example.app", "--edit", "edit-1", "--dir", metadataDir, "--state-dir", stateDir}); err != nil {
		t.Fatalf("parse run flags: %v", err)
	}
	if err := command.Exec(ctx, nil); err != nil {
		t.Fatalf("run command: %v", err)
	}
	if updates != 1 {
		t.Fatalf("listing updates = %d, want 1", updates)
	}
	for _, directory := range []string{"plans", "receipts"} {
		entries, err := os.ReadDir(filepath.Join(stateDir, directory))
		if err != nil {
			t.Fatalf("read %s state: %v", directory, err)
		}
		if len(entries) != 1 || filepath.Ext(entries[0].Name()) != ".json" {
			t.Fatalf("%s artifacts = %#v, want one JSON file", directory, entries)
		}
	}
}

func TestPlanCommandRejectsSymlinkedSourceBeforeAuthentication(t *testing.T) {
	metadataDir := t.TempDir()
	localeDir := filepath.Join(metadataDir, "en-US")
	if err := os.MkdirAll(localeDir, 0o755); err != nil {
		t.Fatalf("mkdir locale: %v", err)
	}
	outside := filepath.Join(t.TempDir(), "title.txt")
	if err := os.WriteFile(outside, []byte("outside"), 0o644); err != nil {
		t.Fatalf("write outside: %v", err)
	}
	if err := os.Symlink(outside, filepath.Join(localeDir, titleFile)); err != nil {
		t.Skipf("symlink unavailable: %v", err)
	}
	authCalls := 0
	ctx := playclient.ContextWithServiceFactory(context.Background(), func(context.Context) (*playclient.Service, error) {
		authCalls++
		return nil, errors.New("authentication should not run")
	})
	command := PlanCommand()
	if err := command.FlagSet.Parse([]string{"--package", "com.example.app", "--edit", "edit-1", "--dir", metadataDir, "--plan-file", filepath.Join(t.TempDir(), "plan.json")}); err != nil {
		t.Fatalf("parse flags: %v", err)
	}
	err := command.Exec(ctx, nil)
	if !errors.Is(err, rootfs.ErrSymlink) {
		t.Fatalf("plan error = %v, want ErrSymlink", err)
	}
	if authCalls != 0 {
		t.Fatalf("authentication calls = %d, want 0", authCalls)
	}
}

func TestApplyCommandDryRunDoesNotMutateOrPersistReceipt(t *testing.T) {
	metadataDir := writeTestListing(t, "New title")
	mutations := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if writeTestEdit(w, r) {
			return
		}
		if r.Method != http.MethodGet {
			mutations++
		}
		_, _ = w.Write([]byte(`{"listings":[{"language":"en-US","title":"Old title","shortDescription":"Short description","fullDescription":"Full description"}]}`))
	}))
	t.Cleanup(server.Close)
	ctx := playclient.ContextWithServiceFactory(context.Background(), func(ctx context.Context) (*playclient.Service, error) {
		return playclient.NewServiceWithClient(ctx, server.Client(), server.URL+"/")
	})
	planPath := filepath.Join(t.TempDir(), "plan.json")
	planCommand := PlanCommand()
	if err := planCommand.FlagSet.Parse([]string{"--package", "com.example.app", "--edit", "edit-1", "--dir", metadataDir, "--plan-file", planPath}); err != nil {
		t.Fatalf("parse plan flags: %v", err)
	}
	if err := planCommand.Exec(ctx, nil); err != nil {
		t.Fatalf("create plan: %v", err)
	}

	receiptPath := filepath.Join(t.TempDir(), "receipt.json")
	applyCommand := ApplyCommand()
	if err := applyCommand.FlagSet.Parse([]string{"--plan-file", planPath, "--receipt-file", receiptPath}); err != nil {
		t.Fatalf("parse apply flags: %v", err)
	}
	if err := applyCommand.Exec(shared.ContextWithDryRun(ctx, true), nil); err != nil {
		t.Fatalf("dry-run apply: %v", err)
	}
	if mutations != 0 {
		t.Fatalf("dry-run made %d mutation requests", mutations)
	}
	if _, err := os.Stat(receiptPath); !os.IsNotExist(err) {
		t.Fatalf("dry-run persisted a receipt: %v", err)
	}
}

func TestApplyCommandRejectsExpiredPlanBeforeAuthentication(t *testing.T) {
	metadataDir := writeTestListing(t, "New title")
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if writeTestEdit(w, r) {
			return
		}
		_, _ = w.Write([]byte(`{"listings":[{"language":"en-US","title":"Old title","shortDescription":"Short description","fullDescription":"Full description"}]}`))
	}))
	t.Cleanup(server.Close)
	planCtx := playclient.ContextWithServiceFactory(context.Background(), func(ctx context.Context) (*playclient.Service, error) {
		return playclient.NewServiceWithClient(ctx, server.Client(), server.URL+"/")
	})
	planPath := filepath.Join(t.TempDir(), "plan.json")
	planCommand := PlanCommand()
	if err := planCommand.FlagSet.Parse([]string{"--package", "com.example.app", "--edit", "edit-1", "--dir", metadataDir, "--plan-file", planPath}); err != nil {
		t.Fatalf("parse plan flags: %v", err)
	}
	if err := planCommand.Exec(planCtx, nil); err != nil {
		t.Fatalf("create plan: %v", err)
	}

	authCalls := 0
	applyCtx := shared.ContextWithClock(context.Background(), shared.ClockFunc(func() time.Time {
		return time.Date(2101, time.January, 1, 0, 0, 0, 0, time.UTC)
	}))
	applyCtx = playclient.ContextWithServiceFactory(applyCtx, func(context.Context) (*playclient.Service, error) {
		authCalls++
		return nil, errors.New("authentication should not run")
	})
	command := ApplyCommand()
	if err := command.FlagSet.Parse([]string{"--plan-file", planPath, "--receipt-file", filepath.Join(t.TempDir(), "receipt.json")}); err != nil {
		t.Fatalf("parse apply flags: %v", err)
	}
	err := command.Exec(applyCtx, nil)
	if err == nil || !strings.Contains(err.Error(), "expired") {
		t.Fatalf("apply error = %v, want expired plan", err)
	}
	if authCalls != 0 {
		t.Fatalf("authentication calls = %d, want 0", authCalls)
	}
}

func writeTestListing(t *testing.T, title string) string {
	t.Helper()
	metadataDir := t.TempDir()
	localeDir := filepath.Join(metadataDir, "en-US")
	if err := os.MkdirAll(localeDir, 0o755); err != nil {
		t.Fatalf("mkdir locale: %v", err)
	}
	for name, content := range map[string]string{
		titleFile:     title + "\n",
		shortDescFile: "Short description\n",
		fullDescFile:  "Full description\n",
	} {
		if err := os.WriteFile(filepath.Join(localeDir, name), []byte(content), 0o644); err != nil {
			t.Fatalf("write %s: %v", name, err)
		}
	}
	return metadataDir
}

func isEditGet(r *http.Request) bool {
	return r.Method == http.MethodGet && filepath.Base(strings.TrimSuffix(r.URL.Path, "/")) == "edit-1"
}

func writeTestEdit(w http.ResponseWriter, r *http.Request) bool {
	if !isEditGet(r) {
		return false
	}
	_, _ = w.Write([]byte(`{"id":"edit-1","expiryTimeSeconds":"4102444800"}`))
	return true
}
