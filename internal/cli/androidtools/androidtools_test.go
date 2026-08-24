package androidtools

import (
	"bytes"
	"context"
	"image"
	"image/color"
	"image/png"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/tamtom/play-console-cli/internal/cli/shared"
	"github.com/tamtom/play-console-cli/internal/localexec"
)

func preserveLookup(t *testing.T) {
	t.Helper()
	original := lookupExecutable
	t.Cleanup(func() { lookupExecutable = original })
}

func TestAndroidBuildRunsTestsLintAndBundleInOneInvocation(t *testing.T) {
	project := t.TempDir()
	wrapper := filepath.Join(project, "gradlew")
	if err := os.WriteFile(wrapper, []byte("#!/bin/sh\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	var request localexec.Request
	runner := localexec.RunnerFunc(func(_ context.Context, got localexec.Request) error {
		request = got
		artifact := filepath.Join(got.Dir, "app", "build", "outputs", "bundle", "release", "app-release.aab")
		if err := os.MkdirAll(filepath.Dir(artifact), 0o755); err != nil {
			return err
		}
		return os.WriteFile(artifact, []byte("aab"), 0o644)
	})
	ctx := localexec.ContextWithRunner(context.Background(), runner)
	ctx = shared.ContextWithIO(ctx, io.Discard, io.Discard)
	result, err := runBuild(ctx, buildOptions{Project: project, Module: "app", Variant: "release", ArtifactType: "aab"})
	if err != nil {
		t.Fatal(err)
	}
	wantTasks := []string{":app:testReleaseUnitTest", ":app:lintRelease", ":app:bundleRelease"}
	if !reflect.DeepEqual(result.Tasks, wantTasks) {
		t.Fatalf("tasks = %#v", result.Tasks)
	}
	if !reflect.DeepEqual(request.Args[:3], wantTasks) || request.Executable != wrapper || request.Dir != project {
		t.Fatalf("request = %#v", request)
	}
	if result.Mode != "built" || len(result.Artifacts) != 1 || result.Artifacts[0].SHA256 == "" {
		t.Fatalf("result = %#v", result)
	}
}

func TestAndroidBuildDryRunDoesNotExecute(t *testing.T) {
	project := t.TempDir()
	wrapper := filepath.Join(project, "gradlew")
	if err := os.WriteFile(wrapper, []byte("#!/bin/sh\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	ctx := shared.ContextWithDryRun(context.Background(), true)
	ctx = localexec.ContextWithRunner(ctx, localexec.RunnerFunc(func(context.Context, localexec.Request) error {
		t.Fatal("dry-run executed Gradle")
		return nil
	}))
	result, err := runBuild(ctx, buildOptions{Project: project, Module: "app", Variant: "release", ArtifactType: "aab"})
	if err != nil || result.Mode != "planned" {
		t.Fatalf("result=%#v err=%v", result, err)
	}
}

func TestVerifySignatureUsesDirectJarsignerInvocation(t *testing.T) {
	preserveLookup(t)
	lookupExecutable = func(name string) (string, error) { return "/tools/" + name, nil }
	artifact := filepath.Join(t.TempDir(), "app.aab")
	if err := os.WriteFile(artifact, []byte("signed"), 0o644); err != nil {
		t.Fatal(err)
	}
	var request localexec.Request
	ctx := localexec.ContextWithRunner(context.Background(), localexec.RunnerFunc(func(_ context.Context, got localexec.Request) error {
		request = got
		_, _ = io.WriteString(got.Stdout, "jar verified")
		return nil
	}))
	result, err := verifySignature(ctx, artifact)
	if err != nil {
		t.Fatal(err)
	}
	if request.Executable != "/tools/jarsigner" || result.Mode != "verified" || !result.Verified {
		t.Fatalf("request=%#v result=%#v", request, result)
	}
}

func TestInspectKeystorePassesOnlyPasswordEnvironmentName(t *testing.T) {
	preserveLookup(t)
	lookupExecutable = func(name string) (string, error) { return "/tools/" + name, nil }
	t.Setenv("UPLOAD_STORE_PASSWORD", "super-secret-value")
	keystore := filepath.Join(t.TempDir(), "upload.jks")
	if err := os.WriteFile(keystore, []byte("keystore"), 0o600); err != nil {
		t.Fatal(err)
	}
	var request localexec.Request
	ctx := localexec.ContextWithRunner(context.Background(), localexec.RunnerFunc(func(_ context.Context, got localexec.Request) error {
		request = got
		_, _ = io.WriteString(got.Stdout, "SHA256: AA:BB")
		return nil
	}))
	result, err := inspectKeystore(ctx, keystore, "upload", "UPLOAD_STORE_PASSWORD")
	if err != nil {
		t.Fatal(err)
	}
	joined := strings.Join(request.Args, " ")
	if strings.Contains(joined, "super-secret-value") || !strings.Contains(joined, "-storepass:env UPLOAD_STORE_PASSWORD") {
		t.Fatalf("unsafe args = %q", joined)
	}
	if result.Mode != "inspected" || !result.Verified {
		t.Fatalf("result = %#v", result)
	}
}

func TestCaptureScreenshotWritesValidatedMetadataLayout(t *testing.T) {
	preserveLookup(t)
	lookupExecutable = func(name string) (string, error) { return "/tools/" + name, nil }
	var pngData bytes.Buffer
	img := image.NewRGBA(image.Rect(0, 0, 1080, 1920))
	img.Set(0, 0, color.White)
	if err := png.Encode(&pngData, img); err != nil {
		t.Fatal(err)
	}
	ctx := localexec.ContextWithRunner(context.Background(), localexec.RunnerFunc(func(_ context.Context, request localexec.Request) error {
		if got := strings.Join(request.Args, " "); got != "-s emulator-5554 exec-out screencap -p" {
			t.Fatalf("adb args = %q", got)
		}
		_, err := request.Stdout.Write(pngData.Bytes())
		return err
	}))
	outputDir := t.TempDir()
	result, err := captureScreenshot(ctx, screenshotOptions{
		Serial: "emulator-5554", Locale: "en-US", ImageType: "phoneScreenshots",
		Name: "01-home", OutputDir: outputDir, Layout: "metadata",
	})
	if err != nil {
		t.Fatal(err)
	}
	wantPath := filepath.Join(outputDir, "en-US", "images", "phoneScreenshots", "01-home.png")
	if result.Mode != "captured" || result.Path != wantPath || result.Width != 1080 || result.Height != 1920 || result.SHA256 == "" {
		t.Fatalf("result = %#v", result)
	}
	if _, err := os.Stat(wantPath); err != nil {
		t.Fatal(err)
	}
}
