package metadata

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"

	"google.golang.org/api/androidpublisher/v3"
	"google.golang.org/api/option"
)

func TestPushCommand_Name(t *testing.T) {
	cmd := PushCommand()
	if cmd.Name != "push" {
		t.Errorf("expected name %q, got %q", "push", cmd.Name)
	}
}

func TestPushCommand_MissingPackage(t *testing.T) {
	cmd := PushCommand()
	if err := cmd.FlagSet.Parse([]string{"--dir", "/tmp/test", "--confirm"}); err != nil {
		t.Fatal(err)
	}
	err := cmd.Exec(context.Background(), nil)
	if err == nil {
		t.Fatal("expected error for missing --package")
	}
}

func TestPushCommand_MissingDir(t *testing.T) {
	cmd := PushCommand()
	if err := cmd.FlagSet.Parse([]string{"--package", "com.example", "--confirm"}); err != nil {
		t.Fatal(err)
	}
	err := cmd.Exec(context.Background(), nil)
	if err == nil {
		t.Fatal("expected error for missing --dir")
	}
}

func TestPushCommand_MissingConfirm(t *testing.T) {
	cmd := PushCommand()
	if err := cmd.FlagSet.Parse([]string{"--package", "com.example", "--dir", "/tmp/test"}); err != nil {
		t.Fatal(err)
	}
	err := cmd.Exec(context.Background(), nil)
	if err == nil {
		t.Fatal("expected error for missing --confirm")
	}
	if !strings.Contains(err.Error(), "--confirm") {
		t.Errorf("error should mention --confirm, got: %s", err.Error())
	}
}

func TestPushCommand_EmptyPackage(t *testing.T) {
	cmd := PushCommand()
	if err := cmd.FlagSet.Parse([]string{"--package", "  ", "--dir", "/tmp/test", "--confirm"}); err != nil {
		t.Fatal(err)
	}
	err := cmd.Exec(context.Background(), nil)
	if err == nil {
		t.Fatal("expected error for whitespace-only --package")
	}
}

func TestPushReadsFilesAndCallsAPI(t *testing.T) {
	// Set up local metadata directory
	dir := t.TempDir()
	localeDir := filepath.Join(dir, "en-US")
	if err := os.MkdirAll(localeDir, 0o755); err != nil {
		t.Fatal(err)
	}
	writeFile(t, filepath.Join(localeDir, "title.txt"), "Updated Title")
	writeFile(t, filepath.Join(localeDir, "short_description.txt"), "Updated Short")
	writeFile(t, filepath.Join(localeDir, "full_description.txt"), "Updated Full")

	var updateCalled atomic.Int32

	mux := http.NewServeMux()

	// Mock edits.insert
	mux.HandleFunc("/androidpublisher/v3/applications/com.example/edits", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			resp := androidpublisher.AppEdit{Id: "edit-456"}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(resp)
			return
		}
	})

	// Mock edits.listings.list (return existing listings so we know what to update)
	mux.HandleFunc("/androidpublisher/v3/applications/com.example/edits/edit-456/listings", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			resp := androidpublisher.ListingsListResponse{
				Listings: []*androidpublisher.Listing{
					{
						Language:         "en-US",
						Title:            "Old Title",
						ShortDescription: "Old Short",
						FullDescription:  "Old Full",
					},
				},
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(resp)
			return
		}
	})

	// Mock edits.listings.update
	mux.HandleFunc("/androidpublisher/v3/applications/com.example/edits/edit-456/listings/en-US", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPut {
			updateCalled.Add(1)
			var listing androidpublisher.Listing
			if err := json.NewDecoder(r.Body).Decode(&listing); err != nil {
				t.Errorf("failed to decode listing body: %v", err)
			}
			if listing.Title != "Updated Title" {
				t.Errorf("expected title %q, got %q", "Updated Title", listing.Title)
			}
			if listing.ShortDescription != "Updated Short" {
				t.Errorf("expected short description %q, got %q", "Updated Short", listing.ShortDescription)
			}
			if listing.FullDescription != "Updated Full" {
				t.Errorf("expected full description %q, got %q", "Updated Full", listing.FullDescription)
			}
			resp := listing
			resp.Language = "en-US"
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(resp)
			return
		}
	})

	// Mock edits.commit
	mux.HandleFunc("/androidpublisher/v3/applications/com.example/edits/edit-456:commit", func(w http.ResponseWriter, r *http.Request) {
		resp := androidpublisher.AppEdit{Id: "edit-456"}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	})

	// Catch-all
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte("{}"))
	})

	server := httptest.NewServer(mux)
	defer server.Close()

	api, err := androidpublisher.NewService(context.Background(), option.WithHTTPClient(server.Client()), option.WithEndpoint(server.URL))
	if err != nil {
		t.Fatal(err)
	}

	err = executePush(context.Background(), api, "com.example", dir, nil, false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if updateCalled.Load() != 1 {
		t.Errorf("expected update to be called 1 time, got %d", updateCalled.Load())
	}
}

func TestPushDryRun_NoAPICalls(t *testing.T) {
	dir := t.TempDir()
	localeDir := filepath.Join(dir, "en-US")
	if err := os.MkdirAll(localeDir, 0o755); err != nil {
		t.Fatal(err)
	}
	writeFile(t, filepath.Join(localeDir, "title.txt"), "Title")
	writeFile(t, filepath.Join(localeDir, "short_description.txt"), "Short")
	writeFile(t, filepath.Join(localeDir, "full_description.txt"), "Full")

	var apiCalls atomic.Int32

	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		apiCalls.Add(1)
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte("{}"))
	})

	server := httptest.NewServer(mux)
	defer server.Close()

	api, err := androidpublisher.NewService(context.Background(), option.WithHTTPClient(server.Client()), option.WithEndpoint(server.URL))
	if err != nil {
		t.Fatal(err)
	}

	err = executePush(context.Background(), api, "com.example", dir, nil, true)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if apiCalls.Load() != 0 {
		t.Errorf("expected no API calls in dry-run mode, got %d", apiCalls.Load())
	}
}

func TestPushCommand_LocaleFilter(t *testing.T) {
	dir := t.TempDir()
	for _, locale := range []string{"en-US", "ja-JP"} {
		localeDir := filepath.Join(dir, locale)
		if err := os.MkdirAll(localeDir, 0o755); err != nil {
			t.Fatal(err)
		}
		writeFile(t, filepath.Join(localeDir, "title.txt"), "Title "+locale)
		writeFile(t, filepath.Join(localeDir, "short_description.txt"), "Short "+locale)
		writeFile(t, filepath.Join(localeDir, "full_description.txt"), "Full "+locale)
	}

	var updatedLocales []string

	mux := http.NewServeMux()
	mux.HandleFunc("/androidpublisher/v3/applications/com.example/edits", func(w http.ResponseWriter, r *http.Request) {
		resp := androidpublisher.AppEdit{Id: "edit-789"}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	})
	mux.HandleFunc("/androidpublisher/v3/applications/com.example/edits/edit-789/listings", func(w http.ResponseWriter, r *http.Request) {
		resp := androidpublisher.ListingsListResponse{
			Listings: []*androidpublisher.Listing{
				{Language: "en-US", Title: "Old", ShortDescription: "Old", FullDescription: "Old"},
				{Language: "ja-JP", Title: "Old", ShortDescription: "Old", FullDescription: "Old"},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	})
	mux.HandleFunc("/androidpublisher/v3/applications/com.example/edits/edit-789/listings/en-US", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPut {
			updatedLocales = append(updatedLocales, "en-US")
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(androidpublisher.Listing{Language: "en-US"})
		}
	})
	mux.HandleFunc("/androidpublisher/v3/applications/com.example/edits/edit-789/listings/ja-JP", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPut {
			updatedLocales = append(updatedLocales, "ja-JP")
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(androidpublisher.Listing{Language: "ja-JP"})
		}
	})
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte("{}"))
	})

	server := httptest.NewServer(mux)
	defer server.Close()

	api, err := androidpublisher.NewService(context.Background(), option.WithHTTPClient(server.Client()), option.WithEndpoint(server.URL))
	if err != nil {
		t.Fatal(err)
	}

	err = executePush(context.Background(), api, "com.example", dir, []string{"en-US"}, false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(updatedLocales) != 1 || updatedLocales[0] != "en-US" {
		t.Errorf("expected only en-US to be updated, got %v", updatedLocales)
	}
}

func TestPushCommand_NoMetadataFiles(t *testing.T) {
	dir := t.TempDir()
	// Empty dir - no locale subdirs

	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte("{}"))
	})

	server := httptest.NewServer(mux)
	defer server.Close()

	api, err := androidpublisher.NewService(context.Background(), option.WithHTTPClient(server.Client()), option.WithEndpoint(server.URL))
	if err != nil {
		t.Fatal(err)
	}

	err = executePush(context.Background(), api, "com.example", dir, nil, false)
	if err == nil {
		t.Fatal("expected error for directory with no metadata files")
	}
}
