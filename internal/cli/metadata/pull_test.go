package metadata

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"google.golang.org/api/androidpublisher/v3"
	"google.golang.org/api/option"
)

func TestPullCommand_Name(t *testing.T) {
	cmd := PullCommand()
	if cmd.Name != "pull" {
		t.Errorf("expected name %q, got %q", "pull", cmd.Name)
	}
}

func TestPullCommand_MissingPackage(t *testing.T) {
	cmd := PullCommand()
	if err := cmd.FlagSet.Parse([]string{"--dir", "/tmp/test"}); err != nil {
		t.Fatal(err)
	}
	err := cmd.Exec(context.Background(), nil)
	if err == nil {
		t.Fatal("expected error for missing --package")
	}
}

func TestPullCommand_MissingDir(t *testing.T) {
	cmd := PullCommand()
	if err := cmd.FlagSet.Parse([]string{"--package", "com.example"}); err != nil {
		t.Fatal(err)
	}
	err := cmd.Exec(context.Background(), nil)
	if err == nil {
		t.Fatal("expected error for missing --dir")
	}
}

func TestPullCommand_EmptyPackage(t *testing.T) {
	cmd := PullCommand()
	if err := cmd.FlagSet.Parse([]string{"--package", "  ", "--dir", "/tmp/test"}); err != nil {
		t.Fatal(err)
	}
	err := cmd.Exec(context.Background(), nil)
	if err == nil {
		t.Fatal("expected error for whitespace-only --package")
	}
}

func TestPullCommand_EmptyDir(t *testing.T) {
	cmd := PullCommand()
	if err := cmd.FlagSet.Parse([]string{"--package", "com.example", "--dir", "  "}); err != nil {
		t.Fatal(err)
	}
	err := cmd.Exec(context.Background(), nil)
	if err == nil {
		t.Fatal("expected error for whitespace-only --dir")
	}
}

func TestPullWritesFiles(t *testing.T) {
	// Create a mock server that returns listings
	mux := http.NewServeMux()

	// Mock edits.insert
	mux.HandleFunc("/androidpublisher/v3/applications/com.example/edits", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			resp := androidpublisher.AppEdit{Id: "edit-123"}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(resp)
			return
		}
	})

	// Mock edits.listings.list
	mux.HandleFunc("/androidpublisher/v3/applications/com.example/edits/edit-123/listings", func(w http.ResponseWriter, r *http.Request) {
		resp := androidpublisher.ListingsListResponse{
			Listings: []*androidpublisher.Listing{
				{
					Language:         "en-US",
					Title:            "My App",
					ShortDescription: "A short desc",
					FullDescription:  "A full desc",
					Video:            "https://youtube.com/watch?v=abc",
				},
				{
					Language:         "ja-JP",
					Title:            "My App JP",
					ShortDescription: "Short JP",
					FullDescription:  "Full JP",
				},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	})

	// Mock edits.delete (cleanup)
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte("{}"))
	})

	server := httptest.NewServer(mux)
	defer server.Close()

	dir := t.TempDir()

	api, err := androidpublisher.NewService(context.Background(), option.WithHTTPClient(server.Client()), option.WithEndpoint(server.URL))
	if err != nil {
		t.Fatal(err)
	}

	err = executePull(context.Background(), api, "com.example", dir, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Check files were created
	assertFileContent(t, filepath.Join(dir, "en-US", "title.txt"), "My App")
	assertFileContent(t, filepath.Join(dir, "en-US", "short_description.txt"), "A short desc")
	assertFileContent(t, filepath.Join(dir, "en-US", "full_description.txt"), "A full desc")
	assertFileContent(t, filepath.Join(dir, "en-US", "video_url.txt"), "https://youtube.com/watch?v=abc")
	assertFileContent(t, filepath.Join(dir, "ja-JP", "title.txt"), "My App JP")
	assertFileContent(t, filepath.Join(dir, "ja-JP", "short_description.txt"), "Short JP")
	assertFileContent(t, filepath.Join(dir, "ja-JP", "full_description.txt"), "Full JP")

	// video_url.txt should not exist for ja-JP (no video)
	if _, err := os.Stat(filepath.Join(dir, "ja-JP", "video_url.txt")); !os.IsNotExist(err) {
		t.Error("expected video_url.txt to not exist for ja-JP")
	}
}

func TestPullWritesFiles_FilteredLocales(t *testing.T) {
	mux := http.NewServeMux()

	mux.HandleFunc("/androidpublisher/v3/applications/com.example/edits", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			resp := androidpublisher.AppEdit{Id: "edit-123"}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(resp)
			return
		}
	})

	mux.HandleFunc("/androidpublisher/v3/applications/com.example/edits/edit-123/listings", func(w http.ResponseWriter, r *http.Request) {
		resp := androidpublisher.ListingsListResponse{
			Listings: []*androidpublisher.Listing{
				{
					Language:         "en-US",
					Title:            "My App",
					ShortDescription: "A short desc",
					FullDescription:  "A full desc",
				},
				{
					Language:         "ja-JP",
					Title:            "My App JP",
					ShortDescription: "Short JP",
					FullDescription:  "Full JP",
				},
				{
					Language:         "fr-FR",
					Title:            "Mon App",
					ShortDescription: "Court FR",
					FullDescription:  "Complet FR",
				},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	})

	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte("{}"))
	})

	server := httptest.NewServer(mux)
	defer server.Close()

	dir := t.TempDir()

	api, err := androidpublisher.NewService(context.Background(), option.WithHTTPClient(server.Client()), option.WithEndpoint(server.URL))
	if err != nil {
		t.Fatal(err)
	}

	locales := []string{"en-US", "ja-JP"}
	err = executePull(context.Background(), api, "com.example", dir, locales)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// en-US and ja-JP should exist
	assertFileContent(t, filepath.Join(dir, "en-US", "title.txt"), "My App")
	assertFileContent(t, filepath.Join(dir, "ja-JP", "title.txt"), "My App JP")

	// fr-FR should NOT exist
	if _, err := os.Stat(filepath.Join(dir, "fr-FR")); !os.IsNotExist(err) {
		t.Error("expected fr-FR directory to not exist when not in --locales filter")
	}
}

func TestPullCommand_LocalesFlagParsing(t *testing.T) {
	cmd := PullCommand()
	if err := cmd.FlagSet.Parse([]string{"--package", "com.example", "--dir", "/tmp/test", "--locales", "en-US,ja-JP"}); err != nil {
		t.Fatal(err)
	}
	// Just verify the flag parsed; actual execution needs API
	localesFlag := cmd.FlagSet.Lookup("locales")
	if localesFlag == nil {
		t.Fatal("expected --locales flag to exist")
	}
	val := localesFlag.Value.String()
	if !strings.Contains(val, "en-US") || !strings.Contains(val, "ja-JP") {
		t.Errorf("expected locales flag to contain en-US,ja-JP, got %q", val)
	}
}

func assertFileContent(t *testing.T, path, expected string) {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("failed to read %s: %v", path, err)
	}
	got := string(data)
	if got != expected {
		t.Errorf("file %s: expected %q, got %q", path, expected, got)
	}
}
