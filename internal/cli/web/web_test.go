package web

import (
	"context"
	"strings"
	"testing"
)

func TestWebCommand_Name(t *testing.T) {
	cmd := WebCommand()
	if cmd.Name != "web" {
		t.Errorf("Name = %q, want %q", cmd.Name, "web")
	}
}

func TestWebCommand_HasSubcommands(t *testing.T) {
	cmd := WebCommand()
	if len(cmd.Subcommands) == 0 {
		t.Error("expected subcommands")
	}
	found := false
	for _, sub := range cmd.Subcommands {
		if sub.Name == "open" {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected 'open' subcommand")
	}
}

func TestWebCommand_NoArgs_ReturnsHelp(t *testing.T) {
	cmd := WebCommand()
	err := cmd.Exec(context.Background(), nil)
	if err == nil {
		t.Error("expected error for no args")
	}
}

func TestOpenCommand_Name(t *testing.T) {
	cmd := OpenCommand()
	if cmd.Name != "open" {
		t.Errorf("Name = %q, want %q", cmd.Name, "open")
	}
}

func TestBuildPlayConsoleURL_Dashboard(t *testing.T) {
	url := buildPlayConsoleURL("com.example.app", "dashboard")
	want := "https://play.google.com/console/developers/app/com.example.app/app-dashboard"
	if url != want {
		t.Errorf("dashboard URL = %q, want %q", url, want)
	}
}

func TestBuildPlayConsoleURL_StoreListing(t *testing.T) {
	url := buildPlayConsoleURL("com.example.app", "store-listing")
	want := "https://play.google.com/console/developers/app/com.example.app/store-listing"
	if url != want {
		t.Errorf("store-listing URL = %q, want %q", url, want)
	}
}

func TestBuildPlayConsoleURL_Releases(t *testing.T) {
	url := buildPlayConsoleURL("com.example.app", "releases")
	want := "https://play.google.com/console/developers/app/com.example.app/tracks"
	if url != want {
		t.Errorf("releases URL = %q, want %q", url, want)
	}
}

func TestBuildPlayConsoleURL_Pricing(t *testing.T) {
	url := buildPlayConsoleURL("com.example.app", "pricing")
	want := "https://play.google.com/console/developers/app/com.example.app/pricing"
	if url != want {
		t.Errorf("pricing URL = %q, want %q", url, want)
	}
}

func TestBuildPlayConsoleURL_Testers(t *testing.T) {
	url := buildPlayConsoleURL("com.example.app", "testers")
	want := "https://play.google.com/console/developers/app/com.example.app/closed-testing"
	if url != want {
		t.Errorf("testers URL = %q, want %q", url, want)
	}
}

func TestBuildPlayConsoleURL_Reviews(t *testing.T) {
	url := buildPlayConsoleURL("com.example.app", "reviews")
	want := "https://play.google.com/console/developers/app/com.example.app/user-feedback/reviews"
	if url != want {
		t.Errorf("reviews URL = %q, want %q", url, want)
	}
}

func TestBuildPlayConsoleURL_Vitals(t *testing.T) {
	url := buildPlayConsoleURL("com.example.app", "vitals")
	want := "https://play.google.com/console/developers/app/com.example.app/vitals/overview"
	if url != want {
		t.Errorf("vitals URL = %q, want %q", url, want)
	}
}

func TestBuildPlayConsoleURL_Statistics(t *testing.T) {
	url := buildPlayConsoleURL("com.example.app", "statistics")
	want := "https://play.google.com/console/developers/app/com.example.app/statistics"
	if url != want {
		t.Errorf("statistics URL = %q, want %q", url, want)
	}
}

func TestBuildPlayConsoleURL_DefaultSection(t *testing.T) {
	url := buildPlayConsoleURL("com.example.app", "")
	want := "https://play.google.com/console/developers/app/com.example.app/app-dashboard"
	if url != want {
		t.Errorf("default URL = %q, want %q", url, want)
	}
}

func TestBuildPlayConsoleURL_UnknownSection(t *testing.T) {
	url := buildPlayConsoleURL("com.example.app", "nonexistent")
	// Unknown sections should fall back to dashboard
	want := "https://play.google.com/console/developers/app/com.example.app/app-dashboard"
	if url != want {
		t.Errorf("unknown section URL = %q, want %q", url, want)
	}
}

func TestValidSections(t *testing.T) {
	sections := validSections()
	if len(sections) == 0 {
		t.Error("expected at least one valid section")
	}
	expected := []string{"dashboard", "store-listing", "releases", "pricing", "testers", "reviews", "vitals", "statistics"}
	for _, s := range expected {
		found := false
		for _, vs := range sections {
			if vs == s {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("expected section %q in valid sections", s)
		}
	}
}

func TestOpenCommand_MissingPackage(t *testing.T) {
	cmd := OpenCommand()
	// Parse with no flags - package will be empty
	if err := cmd.FlagSet.Parse([]string{}); err != nil {
		t.Fatal(err)
	}
	// We need to override the opener to avoid actually opening a browser
	originalOpener := browserOpener
	defer func() { browserOpener = originalOpener }()
	browserOpener = func(url string) error { return nil }

	err := cmd.Exec(context.Background(), nil)
	if err == nil {
		t.Error("expected error for missing package")
	}
	if !strings.Contains(err.Error(), "package") {
		t.Errorf("error should mention package, got: %s", err.Error())
	}
}

func TestOpenCommand_InvalidSection(t *testing.T) {
	cmd := OpenCommand()
	if err := cmd.FlagSet.Parse([]string{"--package", "com.example.app", "--section", "invalid-section"}); err != nil {
		t.Fatal(err)
	}
	originalOpener := browserOpener
	defer func() { browserOpener = originalOpener }()
	browserOpener = func(url string) error { return nil }

	err := cmd.Exec(context.Background(), nil)
	if err == nil {
		t.Error("expected error for invalid section")
	}
	if !strings.Contains(err.Error(), "invalid section") {
		t.Errorf("error should mention invalid section, got: %s", err.Error())
	}
}

func TestOpenCommand_ValidSection(t *testing.T) {
	cmd := OpenCommand()
	if err := cmd.FlagSet.Parse([]string{"--package", "com.example.app", "--section", "vitals"}); err != nil {
		t.Fatal(err)
	}
	var openedURL string
	originalOpener := browserOpener
	defer func() { browserOpener = originalOpener }()
	browserOpener = func(url string) error {
		openedURL = url
		return nil
	}

	err := cmd.Exec(context.Background(), nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(openedURL, "vitals") {
		t.Errorf("opened URL should contain vitals, got: %s", openedURL)
	}
}

func TestOpenCommand_DefaultSection(t *testing.T) {
	cmd := OpenCommand()
	if err := cmd.FlagSet.Parse([]string{"--package", "com.example.app"}); err != nil {
		t.Fatal(err)
	}
	var openedURL string
	originalOpener := browserOpener
	defer func() { browserOpener = originalOpener }()
	browserOpener = func(url string) error {
		openedURL = url
		return nil
	}

	err := cmd.Exec(context.Background(), nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(openedURL, "app-dashboard") {
		t.Errorf("opened URL should contain app-dashboard, got: %s", openedURL)
	}
}
