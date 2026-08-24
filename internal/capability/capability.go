// Package capability describes which Google Play workflows gplay can perform
// through documented APIs and which workflows require a manual Console handoff.
package capability

import (
	"fmt"
	"sort"
	"strings"
)

// Status is the supported execution mode for a capability.
type Status string

const (
	StatusOfficial    Status = "official"
	StatusManual      Status = "manual"
	StatusUnsupported Status = "unsupported"
)

// Capability is a policy-aware description of a Google Play workflow.
type Capability struct {
	ID          string `json:"id"`
	Intent      string `json:"intent"`
	Command     string `json:"command,omitempty"`
	Status      Status `json:"status"`
	Provider    string `json:"provider"`
	APIResource string `json:"api_resource,omitempty"`
	Stability   string `json:"stability"`
	Notes       string `json:"notes,omitempty"`
	NextAction  string `json:"next_action,omitempty"`
}

// Filter selects capabilities without invoking any provider or network API.
type Filter struct {
	Status   Status
	Provider string
	Query    string
}

// List returns the policy registry in stable ID order.
func List(filter Filter) ([]Capability, error) {
	if filter.Status != "" && !validStatus(filter.Status) {
		return nil, fmt.Errorf("invalid status %q: expected official, manual, or unsupported", filter.Status)
	}

	provider := strings.ToLower(strings.TrimSpace(filter.Provider))
	query := strings.ToLower(strings.TrimSpace(filter.Query))
	result := make([]Capability, 0, len(catalog))
	for _, item := range catalog {
		if filter.Status != "" && item.Status != filter.Status {
			continue
		}
		if provider != "" && strings.ToLower(item.Provider) != provider {
			continue
		}
		if query != "" && !strings.Contains(strings.ToLower(strings.Join([]string{
			item.ID, item.Intent, item.Command, item.Provider, item.APIResource, item.Notes,
		}, " ")), query) {
			continue
		}
		result = append(result, item)
	}
	sort.Slice(result, func(i, j int) bool { return result[i].ID < result[j].ID })
	return result, nil
}

func validStatus(status Status) bool {
	switch status {
	case StatusOfficial, StatusManual, StatusUnsupported:
		return true
	default:
		return false
	}
}

var catalog = []Capability{
	{
		ID:         "app.create",
		Intent:     "Create the initial app record",
		Command:    "gplay bootstrap plan",
		Status:     StatusManual,
		Provider:   "play-console",
		Stability:  "stable",
		Notes:      "No documented Android Publisher API creates a standard public app record.",
		NextAction: "Create the app manually in Play Console, then continue with documented APIs.",
	},
	{
		ID:         "app.first_artifact_upload",
		Intent:     "Upload the first app bundle",
		Command:    "gplay bootstrap plan",
		Status:     StatusManual,
		Provider:   "play-console",
		Stability:  "stable",
		Notes:      "The first artifact establishes the app before edit-based API workflows are available.",
		NextAction: "Upload the first AAB manually in Play Console.",
	},
	{
		ID:         "app.legal_consents",
		Intent:     "Review and accept Play Console declarations and agreements",
		Command:    "gplay bootstrap plan",
		Status:     StatusManual,
		Provider:   "play-console",
		Stability:  "stable",
		Notes:      "Legal declarations require an authorized person to review current Console text.",
		NextAction: "Complete each declaration manually in Play Console.",
	},
	{
		ID:          "app.release",
		Intent:      "Upload and publish later releases",
		Command:     "gplay publish",
		Status:      StatusOfficial,
		Provider:    "android-publisher-api",
		APIResource: "edits, bundles, tracks",
		Stability:   "stable",
	},
	{
		ID:          "app.store_listing",
		Intent:      "Manage store listing metadata and images",
		Command:     "gplay metadata",
		Status:      StatusOfficial,
		Provider:    "android-publisher-api",
		APIResource: "edits.listings, edits.images",
		Stability:   "stable",
	},
	{
		ID:         "app.store_listing_experiments",
		Intent:     "Create, monitor, and apply store-listing experiments",
		Command:    "gplay experiments",
		Status:     StatusManual,
		Provider:   "mixed",
		Stability:  "stable",
		Notes:      "The public Android Publisher discovery document has no experiment lifecycle or results resource. Applying a manually selected winner uses official edits.listings and edits.images through the resumable sync engine.",
		NextAction: "Run gplay experiments support, manage the experiment in Play Console, then apply the human-selected winner with gplay experiments apply-winner.",
	},
	{
		ID:          "app.reviews",
		Intent:      "Read and reply to reviews",
		Command:     "gplay reviews",
		Status:      StatusOfficial,
		Provider:    "android-publisher-api",
		APIResource: "reviews",
		Stability:   "stable",
	},
	{
		ID:          "app.vitals",
		Intent:      "Inspect crashes, ANRs, and performance metrics",
		Command:     "gplay vitals",
		Status:      StatusOfficial,
		Provider:    "play-developer-reporting-api",
		APIResource: "apps, vitals",
		Stability:   "stable",
	},
	{
		ID: "app.reporting_metric_sets", Intent: "Describe and query every Play Developer Reporting metric set", Command: "gplay vitals metric-sets", Status: StatusOfficial,
		Provider: "play-developer-reporting-api", APIResource: "apps.fetchReleaseFilterOptions, vitals.*.get/query", Stability: "stable",
	},
	{
		ID: "app.checks_repo_scans", Intent: "Generate and inspect Checks repository scans", Command: "gplay checks repo-scans", Status: StatusOfficial,
		Provider: "checks-api", APIResource: "accounts.repos.scans, accounts.repos.operations", Stability: "stable", Notes: "Generation sends only the explicitly supplied analysis and SCM metadata JSON.",
	},
	{
		ID: "app.play_integrity", Intent: "Decode integrity tokens and manage restricted Device Recall state", Command: "gplay integrity", Status: StatusOfficial,
		Provider: "play-integrity-api", APIResource: "v1.decodeIntegrityToken, v1.decodePcIntegrityToken, deviceRecall.write", Stability: "stable", Notes: "Device Recall writes are restricted to security, fraud, and abuse prevention.",
	},
	{
		ID: "app.developer_id_status", Intent: "Check verified-developer package and certificate registration", Command: "gplay verification status", Status: StatusOfficial,
		Provider: "android-developer-id-status-api", APIResource: "packages.packageRegistrationStatus.check", Stability: "stable",
	},
	{
		ID: "app.third_party_store", Intent: "Submit and inspect apps for a registered third-party app store", Command: "gplay app-stores", Status: StatusOfficial,
		Provider: "android-publisher-api", APIResource: "appstoreappsreview, appstorecatalog", Stability: "stable", Notes: "Only for organizations enrolled in Google's third-party app-store program.",
	},
	{
		ID: "app.enterprise_kms_signing", Intent: "Enroll or rotate enterprise self-hosted Cloud KMS app-signing keys", Command: "gplay app-signing", Status: StatusOfficial,
		Provider: "android-publisher-api", APIResource: "appsigning.enrollApp, appsigning.rotateAppSigningKey", Stability: "stable", Notes: "Only for enterprise self-hosted Cloud KMS custody; exact package confirmation is required.",
	},
	{
		ID: "app.standard_signing_enrollment", Intent: "Enroll an ordinary app in Google-managed Play App Signing", Command: "gplay bootstrap plan", Status: StatusManual,
		Provider: "play-console", Stability: "stable", Notes: "The enterprise self-hosted-KMS API does not apply to standard Google-managed enrollment.", NextAction: "Complete signing enrollment manually in Play Console.",
	},
	{
		ID:         "console.private_automation",
		Intent:     "Drive private Play Console endpoints or authenticated UI flows",
		Status:     StatusUnsupported,
		Provider:   "none",
		Stability:  "stable",
		Notes:      "Intentionally excluded: gplay does not use private RPCs, browser cookies, or automated legal acceptance.",
		NextAction: "Use a documented API or complete the task manually in Play Console.",
	},
}
