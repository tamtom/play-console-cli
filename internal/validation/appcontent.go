package validation

import (
	"fmt"
	"net/mail"
	"net/url"
	"sort"
	"strings"
	"unicode/utf8"
)

// AppContentInventory is a local, user-maintained record of Play Console
// declarations that are not fully readable through public Google APIs.
type AppContentInventory struct {
	PrivacyPolicyURL             string                  `json:"privacyPolicyUrl"`
	SupportEmail                 string                  `json:"supportEmail"`
	Ads                          string                  `json:"ads"`
	AppAccess                    string                  `json:"appAccess"`
	ReviewerInstructions         string                  `json:"reviewerInstructions,omitempty"`
	TargetAudience               []string                `json:"targetAudience"`
	ContentRatingStatus          string                  `json:"contentRatingStatus"`
	DataSafetyStatus             string                  `json:"dataSafetyStatus"`
	Category                     string                  `json:"category,omitempty"`
	Tags                         []string                `json:"tags,omitempty"`
	InitialCountries             []string                `json:"initialCountries,omitempty"`
	Declarations                 map[string]string       `json:"declarations,omitempty"`
	PolicyDeclarationsReviewed   *bool                   `json:"policyDeclarationsReviewed"`
	SensitivePermissions         []PermissionDeclaration `json:"sensitivePermissions,omitempty"`
	SensitivePermissionsReviewed *bool                   `json:"sensitivePermissionsReviewed"`
}

// PermissionDeclaration records whether a sensitive Android permission's Play
// declaration is complete. It does not claim to read Console-only state.
type PermissionDeclaration struct {
	Name   string `json:"name"`
	Status string `json:"status"`
}

// ValidateAppContent validates an offline inventory. Missing or pending
// mandatory declarations are errors; merchandising fields are advisory.
func ValidateAppContent(inventory AppContentInventory) []CheckResult {
	var results []CheckResult
	privacyURL := strings.TrimSpace(inventory.PrivacyPolicyURL)
	if !validHTTPSURL(privacyURL) {
		results = append(results, appContentResult(
			"app-content-privacy-policy-url-invalid", SeverityError, "privacyPolicyUrl",
			"Privacy policy URL is missing or is not a public HTTPS URL.",
			"Provide an HTTPS privacy policy URL with a valid host.",
		))
	} else {
		results = append(results, appContentResult("app-content-privacy-policy-ready", SeverityInfo, "privacyPolicyUrl", "Privacy policy URL is inventoried.", ""))
	}

	email := strings.TrimSpace(inventory.SupportEmail)
	parsed, emailErr := mail.ParseAddress(email)
	if emailErr != nil || parsed.Address != email {
		results = append(results, appContentResult(
			"app-content-support-email-invalid", SeverityError, "supportEmail",
			"Support email is missing or invalid.", "Provide a plain support email address.",
		))
	} else {
		results = append(results, appContentResult("app-content-support-email-ready", SeverityInfo, "supportEmail", "Support email is inventoried.", ""))
	}

	switch normalizedStatus(inventory.Ads) {
	case "yes", "no":
		results = append(results, appContentResult("app-content-ads-declared", SeverityInfo, "ads", "Ads declaration is inventoried.", ""))
	default:
		results = append(results, appContentResult(
			"app-content-ads-unresolved", SeverityError, "ads",
			"Ads declaration must explicitly be yes or no.", "Set ads to \"yes\" or \"no\" after confirming the app and its SDKs.",
		))
	}

	switch normalizedStatus(inventory.AppAccess) {
	case "all-accessible":
		results = append(results, appContentResult("app-content-app-access-declared", SeverityInfo, "appAccess", "App access is inventoried as fully accessible.", ""))
	case "restricted":
		if strings.TrimSpace(inventory.ReviewerInstructions) == "" {
			results = append(results, appContentResult(
				"app-content-reviewer-instructions-missing", SeverityError, "reviewerInstructions",
				"Restricted app access has no reviewer instructions.", "Provide reproducible reviewer access instructions and test credentials in Play Console.",
			))
		} else {
			results = append(results, appContentResult("app-content-reviewer-instructions-ready", SeverityInfo, "reviewerInstructions", "Reviewer access instructions are inventoried.", ""))
		}
	default:
		results = append(results, appContentResult(
			"app-content-app-access-unresolved", SeverityError, "appAccess",
			"App access declaration is unresolved.", "Set appAccess to \"all-accessible\" or \"restricted\".",
		))
	}

	if len(nonEmptyValues(inventory.TargetAudience)) == 0 {
		results = append(results, appContentResult(
			"app-content-target-audience-missing", SeverityError, "targetAudience",
			"Target audience is not inventoried.", "Record the target age groups selected in Play Console.",
		))
	} else {
		results = append(results, appContentResult("app-content-target-audience-ready", SeverityInfo, "targetAudience", "Target audience is inventoried.", ""))
	}

	results = append(results, completionStatusResult(
		"content-rating", "contentRatingStatus", inventory.ContentRatingStatus,
		"Complete the content rating questionnaire in Play Console.",
	))
	results = append(results, completionStatusResult(
		"data-safety", "dataSafetyStatus", inventory.DataSafetyStatus,
		"Complete Data Safety using the official applications.dataSafety API or Play Console.",
	))

	if strings.TrimSpace(inventory.Category) == "" {
		results = append(results, appContentResult("app-content-category-missing", SeverityWarning, "category", "Store category is not inventoried.", "Choose the app category before launch."))
	}
	if len(nonEmptyValues(inventory.Tags)) > 5 {
		results = append(results, appContentResult("app-content-tags-too-many", SeverityWarning, "tags", "More than five store tags are inventoried.", "Review the current Play Console tag limit and keep only the most relevant tags."))
	}
	if len(nonEmptyValues(inventory.InitialCountries)) == 0 {
		results = append(results, appContentResult("app-content-initial-availability-missing", SeverityWarning, "initialCountries", "Initial country availability is not inventoried.", "Record the countries or regions intended for the initial release."))
	}

	if inventory.PolicyDeclarationsReviewed == nil || !*inventory.PolicyDeclarationsReviewed {
		results = append(results, appContentResult(
			"app-content-policy-review-missing", SeverityError, "policyDeclarationsReviewed",
			"Policy declarations have not been explicitly reviewed.",
			"Set policyDeclarationsReviewed to true after reviewing every applicable Play declaration.",
		))
	}
	for _, required := range []string{"financial-features", "health", "news"} {
		if _, present := inventory.Declarations[required]; !present {
			results = append(results, appContentResult(
				"app-content-required-declaration-missing", SeverityError, "declarations."+required,
				fmt.Sprintf("%s declaration is not inventoried.", required),
				"Record this declaration as complete or not-applicable after review.",
			))
		}
	}

	declarationNames := make([]string, 0, len(inventory.Declarations))
	for name := range inventory.Declarations {
		declarationNames = append(declarationNames, name)
	}
	sort.Strings(declarationNames)
	for _, name := range declarationNames {
		status := normalizedStatus(inventory.Declarations[name])
		if status != "complete" && status != "not-applicable" {
			result := appContentResult(
				"app-content-declaration-pending", SeverityError, "declarations."+name,
				fmt.Sprintf("%s declaration is unresolved.", name),
				"Complete the declaration in Play Console or explicitly mark it not-applicable after review.",
			)
			results = append(results, result)
		}
	}

	if inventory.SensitivePermissionsReviewed == nil || !*inventory.SensitivePermissionsReviewed {
		results = append(results, appContentResult(
			"app-content-sensitive-permissions-review-missing", SeverityError, "sensitivePermissionsReviewed",
			"Sensitive permissions have not been explicitly reviewed.",
			"Set sensitivePermissionsReviewed to true after comparing the release artifact permissions with Play declaration requirements.",
		))
	}
	permissions := append([]PermissionDeclaration(nil), inventory.SensitivePermissions...)
	sort.Slice(permissions, func(i, j int) bool { return permissions[i].Name < permissions[j].Name })
	for _, permission := range permissions {
		name := strings.TrimSpace(permission.Name)
		status := normalizedStatus(permission.Status)
		if name == "" || (status != "complete" && status != "not-applicable") {
			results = append(results, appContentResult(
				"app-content-sensitive-permission-pending", SeverityError, "sensitivePermissions",
				fmt.Sprintf("Sensitive permission %q has no completed declaration.", name),
				"Confirm whether the permission is used and complete its Play policy declaration before release.",
			))
		}
	}
	return results
}

// ValidateListingQuality adds advisory push-time checks beyond hard character
// limits. Results use stable IDs so CI can make explicit policy choices.
func ValidateListingQuality(locale string, fields map[string]string) []CheckResult {
	var results []CheckResult
	placeholderTerms := []string{"lorem ipsum", "todo", "tbd", "test app", "description here", "your app"}
	for _, field := range []string{"title", "short_description", "full_description"} {
		value := strings.TrimSpace(fields[field])
		lower := strings.ToLower(value)
		for _, term := range placeholderTerms {
			if strings.Contains(lower, term) {
				results = append(results, CheckResult{
					ID: "metadata-placeholder", Severity: SeverityWarning, Locale: locale, Field: field,
					Message:     fmt.Sprintf("%s appears to contain placeholder copy.", field),
					Remediation: "Replace placeholder copy with final store-facing text.",
				})
				break
			}
		}
	}
	if value := strings.TrimSpace(fields["title"]); value != "" && utf8.RuneCountInString(value) < 2 {
		results = append(results, qualityLengthResult(locale, "title", "title-too-short", 2))
	}
	if value := strings.TrimSpace(fields["short_description"]); value != "" && utf8.RuneCountInString(value) < 10 {
		results = append(results, qualityLengthResult(locale, "short_description", "short-description-too-short", 10))
	}
	if value := strings.TrimSpace(fields["full_description"]); value != "" && utf8.RuneCountInString(value) < 80 {
		results = append(results, qualityLengthResult(locale, "full_description", "full-description-too-short", 80))
	}
	if video := strings.TrimSpace(fields["video"]); video != "" && !validHTTPURL(video) {
		results = append(results, CheckResult{
			ID: "video-url-invalid", Severity: SeverityWarning, Locale: locale, Field: "video",
			Message: "Video URL is not a valid HTTP(S) URL.", Remediation: "Provide a public Play-supported video URL or remove it.",
		})
	}
	return results
}

func completionStatusResult(name, field, status, remediation string) CheckResult {
	if normalizedStatus(status) == "complete" {
		return appContentResult("app-content-"+name+"-complete", SeverityInfo, field, strings.ReplaceAll(name, "-", " ")+" is inventoried as complete.", "")
	}
	return appContentResult("app-content-"+name+"-pending", SeverityError, field, strings.ReplaceAll(name, "-", " ")+" is not complete.", remediation)
}

func appContentResult(id string, severity Severity, field, message, remediation string) CheckResult {
	return CheckResult{ID: id, Severity: severity, Field: field, Message: message, Remediation: remediation}
}

func qualityLengthResult(locale, field, id string, minimum int) CheckResult {
	return CheckResult{
		ID: id, Severity: SeverityWarning, Locale: locale, Field: field,
		Message:     fmt.Sprintf("%s is unusually short.", field),
		Remediation: fmt.Sprintf("Use at least %d meaningful characters unless the short copy is intentional.", minimum),
	}
}

func validHTTPSURL(value string) bool {
	parsed, err := url.ParseRequestURI(value)
	return err == nil && parsed.Scheme == "https" && parsed.Host != "" && !strings.EqualFold(parsed.Hostname(), "localhost")
}

func validHTTPURL(value string) bool {
	parsed, err := url.ParseRequestURI(value)
	return err == nil && (parsed.Scheme == "http" || parsed.Scheme == "https") && parsed.Host != ""
}

func normalizedStatus(value string) string {
	return strings.ToLower(strings.TrimSpace(value))
}

func nonEmptyValues(values []string) []string {
	result := make([]string, 0, len(values))
	for _, value := range values {
		if trimmed := strings.TrimSpace(value); trimmed != "" {
			result = append(result, trimmed)
		}
	}
	return result
}
