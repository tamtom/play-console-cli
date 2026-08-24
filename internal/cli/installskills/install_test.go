package installskills

import (
	"bytes"
	"context"
	"errors"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/tamtom/play-console-cli/internal/cli/shared"
)

func preserveInstallerDependencies(t *testing.T) {
	t.Helper()
	oldHome := userHomeDirectory
	oldCheckout := checkoutPinnedSkills
	oldPins := skillPins
	oldRename := renameWithinRoot
	t.Cleanup(func() {
		userHomeDirectory = oldHome
		checkoutPinnedSkills = oldCheckout
		skillPins = oldPins
		renameWithinRoot = oldRename
	})
}

func runCommand(t *testing.T, ctx context.Context, args ...string) error {
	t.Helper()
	command := Command()
	if err := command.Parse(args); err != nil {
		t.Fatalf("parse: %v", err)
	}
	return command.Run(ctx)
}

func fixturePins(t *testing.T, sourceRoot string, contents map[string]string) []skillPin {
	t.Helper()
	pins := make([]skillPin, 0, len(contents))
	for name, content := range contents {
		dir := filepath.Join(sourceRoot, "skills", name)
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dir, "SKILL.md"), []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
		hash, err := gitTreeHash(dir)
		if err != nil {
			t.Fatal(err)
		}
		pins = append(pins, skillPin{Name: name, TreeHash: hash})
	}
	return pins
}

func TestSkillsSourceIsPinnedAndComplete(t *testing.T) {
	if !regexp.MustCompile(`^[0-9a-f]{40}$`).MatchString(skillsSourceCommit) {
		t.Fatalf("source commit = %q", skillsSourceCommit)
	}
	if skillsSourceRepositoryURL != "https://github.com/tamtom/gplay-cli-skills.git" {
		t.Fatalf("source repository = %q", skillsSourceRepositoryURL)
	}
	if len(skillPins) != 17 {
		t.Fatalf("skill count = %d, want 17", len(skillPins))
	}
	for _, pin := range skillPins {
		if !strings.HasPrefix(pin.Name, "gplay-") || !regexp.MustCompile(`^[0-9a-f]{40}$`).MatchString(pin.TreeHash) {
			t.Fatalf("invalid pin: %#v", pin)
		}
	}
}

func TestPreviewDoesNotCheckoutOrCreateDestination(t *testing.T) {
	preserveInstallerDependencies(t)
	home := t.TempDir()
	userHomeDirectory = func() (string, error) { return home, nil }
	checkoutPinnedSkills = func(context.Context, string) (string, error) {
		t.Fatal("preview attempted a network checkout")
		return "", nil
	}

	var stdout bytes.Buffer
	ctx := shared.ContextWithIO(context.Background(), &stdout, &bytes.Buffer{})
	if err := runCommand(t, ctx, "--preview"); err != nil {
		t.Fatalf("preview: %v", err)
	}
	if !strings.Contains(stdout.String(), `"mode":"preview"`) || !strings.Contains(stdout.String(), skillsSourceCommit) {
		t.Fatalf("preview output = %q", stdout.String())
	}
	if _, err := os.Stat(filepath.Join(home, ".agents")); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("preview created destination state: %v", err)
	}
}

func TestInstallRefusesExistingSkillBeforeCheckout(t *testing.T) {
	preserveInstallerDependencies(t)
	home := t.TempDir()
	destination := filepath.Join(home, "skills")
	if err := os.MkdirAll(filepath.Join(destination, skillPins[0].Name), 0o755); err != nil {
		t.Fatal(err)
	}
	oldPath := filepath.Join(destination, skillPins[0].Name, "SKILL.md")
	if err := os.WriteFile(oldPath, []byte("old\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	checkoutPinnedSkills = func(context.Context, string) (string, error) {
		t.Fatal("conflict attempted a network checkout")
		return "", nil
	}

	err := runCommand(t, context.Background(), "--dest", destination)
	if err == nil || !strings.Contains(err.Error(), "already exist") || !strings.Contains(err.Error(), "--force") {
		t.Fatalf("error = %v", err)
	}
	data, readErr := os.ReadFile(oldPath)
	if readErr != nil || string(data) != "old\n" {
		t.Fatalf("existing skill changed: data=%q err=%v", data, readErr)
	}
}

func TestInstallVerifiesAndCopiesPinnedPack(t *testing.T) {
	preserveInstallerDependencies(t)
	workspaceSource := t.TempDir()
	skillPins = fixturePins(t, workspaceSource, map[string]string{
		"gplay-alpha": "---\nname: gplay-alpha\n---\n",
		"gplay-beta":  "---\nname: gplay-beta\n---\n",
	})
	checkoutPinnedSkills = func(context.Context, string) (string, error) { return workspaceSource, nil }
	destination := filepath.Join(t.TempDir(), "skills")
	var stdout bytes.Buffer
	ctx := shared.ContextWithIO(context.Background(), &stdout, &bytes.Buffer{})

	if err := runCommand(t, ctx, "--dest", destination); err != nil {
		t.Fatalf("install: %v", err)
	}
	for _, pin := range skillPins {
		if _, err := os.Stat(filepath.Join(destination, pin.Name, "SKILL.md")); err != nil {
			t.Fatalf("%s not installed: %v", pin.Name, err)
		}
	}
	if !strings.Contains(stdout.String(), `"mode":"installed"`) || !strings.Contains(stdout.String(), `"installedCount":2`) {
		t.Fatalf("receipt = %q", stdout.String())
	}
}

func TestInstallRejectsTreeHashMismatchWithoutChangingDestination(t *testing.T) {
	preserveInstallerDependencies(t)
	workspaceSource := t.TempDir()
	skillPins = fixturePins(t, workspaceSource, map[string]string{"gplay-alpha": "reviewed\n"})
	skillPins[0].TreeHash = strings.Repeat("0", 40)
	checkoutPinnedSkills = func(context.Context, string) (string, error) { return workspaceSource, nil }
	destination := filepath.Join(t.TempDir(), "skills")

	err := runCommand(t, context.Background(), "--dest", destination)
	if err == nil || !strings.Contains(err.Error(), "tree hash") {
		t.Fatalf("error = %v", err)
	}
	entries, readErr := os.ReadDir(destination)
	if readErr != nil {
		t.Fatal(readErr)
	}
	if len(entries) != 0 {
		t.Fatalf("destination changed: %v", entries)
	}
}

func TestForceInstallRollsBackAllSkillsOnPartialFailure(t *testing.T) {
	preserveInstallerDependencies(t)
	workspaceSource := t.TempDir()
	skillPins = fixturePins(t, workspaceSource, map[string]string{
		"gplay-alpha": "new alpha\n",
		"gplay-beta":  "new beta\n",
	})
	checkoutPinnedSkills = func(context.Context, string) (string, error) { return workspaceSource, nil }
	destination := filepath.Join(t.TempDir(), "skills")
	for _, pin := range skillPins {
		path := filepath.Join(destination, pin.Name)
		if err := os.MkdirAll(path, 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(path, "SKILL.md"), []byte("old "+pin.Name+"\n"), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	originalRename := renameWithinRoot
	failed := false
	renameWithinRoot = func(root *os.Root, oldName, newName string) error {
		if !failed && strings.Contains(oldName, ".gplay-stage-") && filepath.Base(oldName) == "gplay-beta" {
			failed = true
			return errors.New("injected rename failure")
		}
		return originalRename(root, oldName, newName)
	}
	err := runCommand(t, context.Background(), "--dest", destination, "--force")
	if err == nil || !strings.Contains(err.Error(), "injected rename failure") {
		t.Fatalf("error = %v", err)
	}
	for _, pin := range skillPins {
		data, readErr := os.ReadFile(filepath.Join(destination, pin.Name, "SKILL.md"))
		if readErr != nil || string(data) != "old "+pin.Name+"\n" {
			t.Fatalf("%s not rolled back: data=%q err=%v", pin.Name, data, readErr)
		}
	}
	entries, readErr := os.ReadDir(destination)
	if readErr != nil {
		t.Fatal(readErr)
	}
	for _, entry := range entries {
		if strings.HasPrefix(entry.Name(), ".gplay-") {
			t.Fatalf("transaction artifact was not cleaned up: %s", entry.Name())
		}
	}
}
