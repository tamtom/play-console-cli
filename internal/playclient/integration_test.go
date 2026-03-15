//go:build integration

package playclient

import (
	"context"
	"os"
	"testing"
	"time"
)

const integrationPackage = "com.itdeveapps.stepsshare"

// newIntegrationService creates an authenticated service for integration tests.
// It skips the test if credentials are not available.
func newIntegrationService(t *testing.T) *Service {
	t.Helper()

	if v := os.Getenv("GPLAY_INTEGRATION_TEST"); v != "1" && v != "true" {
		t.Skip("skipping integration test; set GPLAY_INTEGRATION_TEST=1")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	t.Cleanup(cancel)

	service, err := NewService(ctx)
	if err != nil {
		t.Fatalf("creating service: %v", err)
	}
	return service
}

// createAndCleanupEdit creates an edit and registers cleanup to delete it.
func createAndCleanupEdit(t *testing.T, service *Service) string {
	t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	edit, err := service.API.Edits.Insert(integrationPackage, nil).Context(ctx).Do()
	if err != nil {
		t.Fatalf("creating edit: %v", err)
	}

	t.Cleanup(func() {
		delCtx, delCancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer delCancel()
		_ = service.API.Edits.Delete(integrationPackage, edit.Id).Context(delCtx).Do()
	})

	return edit.Id
}

// --- Edit lifecycle ---

func TestIntegration_EditsCreateAndDelete(t *testing.T) {
	service := newIntegrationService(t)

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	edit, err := service.API.Edits.Insert(integrationPackage, nil).Context(ctx).Do()
	if err != nil {
		t.Fatalf("edits create: %v", err)
	}
	if edit.Id == "" {
		t.Fatal("expected non-empty edit ID")
	}
	t.Logf("created edit: %s (expires: %s)", edit.Id, edit.ExpiryTimeSeconds)

	if err = service.API.Edits.Delete(integrationPackage, edit.Id).Context(ctx).Do(); err != nil {
		t.Fatalf("edits delete: %v", err)
	}
}

// --- Tracks ---

func TestIntegration_TracksList(t *testing.T) {
	service := newIntegrationService(t)
	editID := createAndCleanupEdit(t, service)

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	resp, err := service.API.Edits.Tracks.List(integrationPackage, editID).Context(ctx).Do()
	if err != nil {
		t.Fatalf("tracks list: %v", err)
	}

	if len(resp.Tracks) == 0 {
		t.Fatal("expected at least one track")
	}

	// Verify known tracks exist
	trackNames := make(map[string]bool)
	for _, track := range resp.Tracks {
		trackNames[track.Track] = true
		t.Logf("track: %s (releases: %d)", track.Track, len(track.Releases))
	}

	if !trackNames["production"] {
		t.Error("expected production track")
	}
}

func TestIntegration_TracksGet(t *testing.T) {
	service := newIntegrationService(t)
	editID := createAndCleanupEdit(t, service)

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	track, err := service.API.Edits.Tracks.Get(integrationPackage, editID, "production").Context(ctx).Do()
	if err != nil {
		t.Fatalf("tracks get: %v", err)
	}

	if track.Track != "production" {
		t.Errorf("expected track name 'production', got %q", track.Track)
	}
	if len(track.Releases) == 0 {
		t.Error("expected at least one release on production")
	}

	for _, r := range track.Releases {
		t.Logf("release: %s status=%s versionCodes=%v", r.Name, r.Status, r.VersionCodes)
	}
}

// --- Listings ---

func TestIntegration_ListingsList(t *testing.T) {
	service := newIntegrationService(t)
	editID := createAndCleanupEdit(t, service)

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	resp, err := service.API.Edits.Listings.List(integrationPackage, editID).Context(ctx).Do()
	if err != nil {
		t.Fatalf("listings list: %v", err)
	}

	if len(resp.Listings) == 0 {
		t.Fatal("expected at least one listing")
	}

	var foundEnUS bool
	for _, listing := range resp.Listings {
		t.Logf("listing: %s title=%q", listing.Language, listing.Title)
		if listing.Language == "en-US" {
			foundEnUS = true
			if listing.Title == "" {
				t.Error("en-US listing should have a title")
			}
			if listing.FullDescription == "" {
				t.Error("en-US listing should have a full description")
			}
		}
	}

	if !foundEnUS {
		t.Error("expected en-US listing")
	}
}

func TestIntegration_ListingsGet(t *testing.T) {
	service := newIntegrationService(t)
	editID := createAndCleanupEdit(t, service)

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	listing, err := service.API.Edits.Listings.Get(integrationPackage, editID, "en-US").Context(ctx).Do()
	if err != nil {
		t.Fatalf("listings get en-US: %v", err)
	}

	if listing.Language != "en-US" {
		t.Errorf("expected language 'en-US', got %q", listing.Language)
	}
	if listing.Title == "" {
		t.Error("expected non-empty title")
	}
	t.Logf("title: %q, shortDesc: %q", listing.Title, listing.ShortDescription)
}

// --- Reviews ---

func TestIntegration_ReviewsList(t *testing.T) {
	service := newIntegrationService(t)
	// Reviews don't need an edit

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	resp, err := service.API.Reviews.List(integrationPackage).Context(ctx).Do()
	if err != nil {
		t.Fatalf("reviews list: %v", err)
	}

	// App may have no reviews — that's OK, just verify the call works
	t.Logf("reviews count: %d", len(resp.Reviews))

	for _, review := range resp.Reviews {
		if review.AuthorName == "" && len(review.Comments) == 0 {
			t.Error("review should have an author or comments")
		}
	}
}

// --- Error handling ---

func TestIntegration_InvalidPackage_Returns404(t *testing.T) {
	service := newIntegrationService(t)

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	_, err := service.API.Edits.Insert("com.nonexistent.fake.package.name", nil).Context(ctx).Do()
	if err == nil {
		t.Fatal("expected error for invalid package, got nil")
	}
	t.Logf("expected error: %v", err)
}

func TestIntegration_InvalidEditID_ReturnsError(t *testing.T) {
	service := newIntegrationService(t)

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	_, err := service.API.Edits.Tracks.List(integrationPackage, "invalid-edit-id-999").Context(ctx).Do()
	if err == nil {
		t.Fatal("expected error for invalid edit ID, got nil")
	}
	t.Logf("expected error: %v", err)
}

func TestIntegration_InvalidTrack_ReturnsError(t *testing.T) {
	service := newIntegrationService(t)
	editID := createAndCleanupEdit(t, service)

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	_, err := service.API.Edits.Tracks.Get(integrationPackage, editID, "nonexistent-track").Context(ctx).Do()
	if err == nil {
		t.Fatal("expected error for invalid track name, got nil")
	}
	t.Logf("expected error: %v", err)
}
