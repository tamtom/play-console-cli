package auth

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/tamtom/play-console-cli/internal/config"
)

type fixResult struct {
	Name    string `json:"name"`
	Status  string `json:"status"` // "fixed", "dry_run", "failed", "manual_action_required", "skipped"
	Message string `json:"message"`
}

func attemptFixes(_ authReport, apply bool) []fixResult {
	var fixes []fixResult

	// Fix 1: Missing config directory
	configPath, err := config.GlobalPath()
	if err == nil {
		configDir := filepath.Dir(configPath)
		if _, statErr := os.Stat(configDir); os.IsNotExist(statErr) {
			if apply {
				if mkErr := os.MkdirAll(configDir, 0o700); mkErr == nil {
					fixes = append(fixes, fixResult{
						Name:    "config_directory",
						Status:  "fixed",
						Message: fmt.Sprintf("Created %s", configDir),
					})
				} else {
					fixes = append(fixes, fixResult{
						Name:    "config_directory",
						Status:  "failed",
						Message: fmt.Sprintf("Failed to create %s: %v", configDir, mkErr),
					})
				}
			} else {
				fixes = append(fixes, fixResult{
					Name:    "config_directory",
					Status:  "dry_run",
					Message: fmt.Sprintf("Would create %s", configDir),
				})
			}
		}
	}

	// Fix 2: Missing config file
	if err == nil {
		if _, statErr := os.Stat(configPath); os.IsNotExist(statErr) {
			if apply {
				dir := filepath.Dir(configPath)
				if mkErr := os.MkdirAll(dir, 0o700); mkErr != nil {
					fixes = append(fixes, fixResult{
						Name:    "config_file",
						Status:  "failed",
						Message: fmt.Sprintf("Failed to create directory %s: %v", dir, mkErr),
					})
				} else if saveErr := config.SaveAt(configPath, &config.Config{}); saveErr == nil {
					fixes = append(fixes, fixResult{
						Name:    "config_file",
						Status:  "fixed",
						Message: fmt.Sprintf("Created default config at %s", configPath),
					})
				} else {
					fixes = append(fixes, fixResult{
						Name:    "config_file",
						Status:  "failed",
						Message: fmt.Sprintf("Failed to create config at %s: %v", configPath, saveErr),
					})
				}
			} else {
				fixes = append(fixes, fixResult{
					Name:    "config_file",
					Status:  "dry_run",
					Message: fmt.Sprintf("Would create default config at %s", configPath),
				})
			}
		}
	}

	// Fix 3: Service account guidance
	if envSA := os.Getenv("GPLAY_SERVICE_ACCOUNT"); envSA != "" {
		if _, err := os.Stat(envSA); err == nil {
			fixes = append(fixes, fixResult{
				Name:    "service_account",
				Status:  "manual_action_required",
				Message: fmt.Sprintf("Run: gplay auth login --service-account %s", envSA),
			})
		}
	}

	return fixes
}

func printFixes(fixes []fixResult) {
	if len(fixes) == 0 {
		fmt.Println("No fixes available.")
		return
	}
	fmt.Println("\nFix Results:")
	for _, f := range fixes {
		fmt.Printf("  [%s] %s: %s\n", f.Status, f.Name, f.Message)
	}
}
