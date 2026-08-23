// Package bootstrap builds deterministic, offline plans for the initial Google
// Play app setup that cannot be completed through documented APIs.
package bootstrap

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
)

const consoleURL = "https://play.google.com/console/developers"

var packageNamePattern = regexp.MustCompile(`^[A-Za-z][A-Za-z0-9_]*(\.[A-Za-z][A-Za-z0-9_]*)+$`)

// Request contains the local inputs needed to build an initial-app plan.
type Request struct {
	PackageName  string
	AppName      string
	ArtifactPath string
}

// Artifact identifies the local app bundle used by the plan.
type Artifact struct {
	Path   string `json:"path"`
	SHA256 string `json:"sha256"`
	Size   int64  `json:"size_bytes"`
}

// Step is one offline, manual, or documented-API stage in a bootstrap plan.
type Step struct {
	ID           string   `json:"id"`
	Title        string   `json:"title"`
	Mode         string   `json:"mode"`
	Status       string   `json:"status"`
	Command      string   `json:"command,omitempty"`
	URL          string   `json:"url,omitempty"`
	Instructions []string `json:"instructions,omitempty"`
}

// Plan is an immutable description of work. It never executes any step.
type Plan struct {
	Version               int      `json:"version"`
	ID                    string   `json:"id"`
	Status                string   `json:"status"`
	PackageName           string   `json:"package_name"`
	AppName               string   `json:"app_name"`
	Artifact              Artifact `json:"artifact"`
	ExecutesChanges       bool     `json:"executes_changes"`
	UsesPrivateInterfaces bool     `json:"uses_private_interfaces"`
	RequiresManualConsole bool     `json:"requires_manual_console"`
	Steps                 []Step   `json:"steps"`
	NextAction            string   `json:"next_action"`
}

// BuildPlan validates local inputs and returns a deterministic plan. It does
// not authenticate, make network requests, open a browser, or mutate the app.
func BuildPlan(request Request) (*Plan, error) {
	packageName := strings.TrimSpace(request.PackageName)
	if !packageNamePattern.MatchString(packageName) {
		return nil, fmt.Errorf("--package must be a valid applicationId with at least two segments")
	}
	if isPlaceholderPackage(packageName) {
		return nil, fmt.Errorf("--package uses a reserved placeholder prefix")
	}
	appName := strings.TrimSpace(request.AppName)
	if appName == "" {
		return nil, fmt.Errorf("--name is required")
	}

	artifact, err := inspectArtifact(request.ArtifactPath)
	if err != nil {
		return nil, err
	}

	quotedArtifact := strconv.Quote(artifact.Path)
	plan := &Plan{
		Version:               1,
		Status:                "manual_action_required",
		PackageName:           packageName,
		AppName:               appName,
		Artifact:              artifact,
		ExecutesChanges:       false,
		UsesPrivateInterfaces: false,
		RequiresManualConsole: true,
		Steps: []Step{
			{
				ID:      "preflight-app-bundle",
				Title:   "Run offline app bundle checks",
				Mode:    "offline",
				Status:  "pending",
				Command: "gplay preflight --file " + quotedArtifact,
			},
			{
				ID:     "create-app-record",
				Title:  "Create the app record in Play Console",
				Mode:   "manual",
				Status: "pending",
				URL:    consoleURL,
				Instructions: []string{
					"Sign in directly to Play Console with the intended developer account.",
					"Create the app record and enter the requested app details yourself.",
					"Verify the displayed developer account and package name before continuing.",
				},
			},
			{
				ID:     "upload-first-app-bundle",
				Title:  "Upload the first app bundle in Play Console",
				Mode:   "manual",
				Status: "pending",
				URL:    consoleURL,
				Instructions: []string{
					"Use Play Console's own upload flow for the first app bundle.",
					"Confirm that the detected package name is " + packageName + ".",
					"Do not submit a production release as part of this plan.",
				},
			},
			{
				ID:     "configure-play-app-signing",
				Title:  "Review Play App Signing",
				Mode:   "manual",
				Status: "pending",
				URL:    consoleURL,
				Instructions: []string{
					"Review the current Play App Signing choices and key-management consequences.",
					"Make the signing choice manually as an authorized account owner.",
				},
			},
			{
				ID:     "review-legal-declarations",
				Title:  "Review required declarations and agreements",
				Mode:   "manual",
				Status: "pending",
				URL:    consoleURL,
				Instructions: []string{
					"Read the current text shown by Google before accepting anything.",
					"Only an authorized person should accept declarations or agreements.",
				},
			},
			{
				ID:      "continue-with-documented-api",
				Title:   "Use documented APIs for later releases",
				Mode:    "official",
				Status:  "pending",
				Command: "gplay publish track --package " + packageName + " --track internal --bundle /path/to/next-version.aab --status draft",
			},
		},
		NextAction: "Run the offline preflight command, then complete the manual Play Console steps yourself.",
	}

	id, err := planID(plan)
	if err != nil {
		return nil, err
	}
	plan.ID = id
	return plan, nil
}

func isPlaceholderPackage(packageName string) bool {
	for _, prefix := range []string{"com.example.", "com.android.", "android."} {
		if strings.HasPrefix(packageName, prefix) {
			return true
		}
	}
	return packageName == "com.example" || packageName == "com.android"
}

func inspectArtifact(path string) (Artifact, error) {
	path = strings.TrimSpace(path)
	if path == "" {
		return Artifact{}, fmt.Errorf("--aab is required")
	}
	absPath, err := filepath.Abs(path)
	if err != nil {
		return Artifact{}, fmt.Errorf("resolve --aab: %w", err)
	}
	if !strings.EqualFold(filepath.Ext(absPath), ".aab") {
		return Artifact{}, fmt.Errorf("--aab must point to an .aab file")
	}

	file, err := os.Open(absPath)
	if err != nil {
		return Artifact{}, fmt.Errorf("open --aab: %w", err)
	}
	defer file.Close()
	info, err := file.Stat()
	if err != nil {
		return Artifact{}, fmt.Errorf("inspect --aab: %w", err)
	}
	if !info.Mode().IsRegular() {
		return Artifact{}, fmt.Errorf("--aab must point to a regular file")
	}

	hash := sha256.New()
	if _, err := io.Copy(hash, file); err != nil {
		return Artifact{}, fmt.Errorf("hash --aab: %w", err)
	}
	return Artifact{
		Path:   absPath,
		SHA256: hex.EncodeToString(hash.Sum(nil)),
		Size:   info.Size(),
	}, nil
}

func planID(plan *Plan) (string, error) {
	data, err := json.Marshal(plan)
	if err != nil {
		return "", fmt.Errorf("encode bootstrap plan: %w", err)
	}
	sum := sha256.Sum256(data)
	return "sha256:" + hex.EncodeToString(sum[:]), nil
}
