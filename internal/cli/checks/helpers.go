package checks

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	checksapi "google.golang.org/api/checks/v1alpha"

	"github.com/tamtom/play-console-cli/internal/checksclient"
	"github.com/tamtom/play-console-cli/internal/config"
)

var errThresholdExceeded = errors.New("checks severity threshold exceeded")

var severityRank = map[string]int{
	"OPPORTUNITY": 1,
	"POTENTIAL":   2,
	"PRIORITY":    3,
}

var policyTypes = []string{
	"DANGEROUS_CONTENT",
	"PII_SOLICITING_RECITING",
	"HARASSMENT",
	"SEXUALLY_EXPLICIT",
	"HATE_SPEECH",
	"MEDICAL_INFO",
	"VIOLENCE_AND_GORE",
	"OBSCENITY_AND_PROFANITY",
}

func accountResource(account string) string {
	account = strings.Trim(strings.TrimSpace(account), "/")
	if strings.HasPrefix(account, "accounts/") {
		return account
	}
	return "accounts/" + account
}

func appResource(account, app string) string {
	app = strings.Trim(strings.TrimSpace(app), "/")
	if strings.HasPrefix(app, "accounts/") {
		return app
	}
	if strings.HasPrefix(app, "apps/") {
		return accountResource(account) + "/" + app
	}
	return accountResource(account) + "/apps/" + app
}

func reportResource(account, app, report string) string {
	report = strings.Trim(strings.TrimSpace(report), "/")
	if strings.HasPrefix(report, "accounts/") {
		return report
	}
	if strings.HasPrefix(report, "reports/") {
		return appResource(account, app) + "/" + report
	}
	return appResource(account, app) + "/reports/" + report
}

func operationResource(account, app, operation string) string {
	operation = strings.Trim(strings.TrimSpace(operation), "/")
	if strings.HasPrefix(operation, "accounts/") {
		return operation
	}
	if strings.HasPrefix(operation, "operations/") {
		return appResource(account, app) + "/" + operation
	}
	return appResource(account, app) + "/operations/" + operation
}

func repoResource(account, repo string) string {
	repo = strings.Trim(strings.TrimSpace(repo), "/")
	if strings.HasPrefix(repo, "accounts/") {
		return repo
	}
	if strings.HasPrefix(repo, "repos/") {
		return accountResource(account) + "/" + repo
	}
	return accountResource(account) + "/repos/" + repo
}

func repoScanResource(account, repo, scan string) string {
	scan = strings.Trim(strings.TrimSpace(scan), "/")
	if strings.HasPrefix(scan, "accounts/") {
		return scan
	}
	if strings.HasPrefix(scan, "scans/") {
		return repoResource(account, repo) + "/" + scan
	}
	return repoResource(account, repo) + "/scans/" + scan
}

func repoOperationResource(account, repo, operation string) string {
	operation = strings.Trim(strings.TrimSpace(operation), "/")
	if strings.HasPrefix(operation, "accounts/") {
		return operation
	}
	if strings.HasPrefix(operation, "operations/") {
		return repoResource(account, repo) + "/" + operation
	}
	return repoResource(account, repo) + "/operations/" + operation
}

func validatePageSize(pageSize int) error {
	if pageSize < 1 || pageSize > 50 {
		return fmt.Errorf("--page-size must be between 1 and 50")
	}
	return nil
}

func validateAppBinaryFileType(value string) error {
	switch strings.TrimSpace(value) {
	case "", "APP_BINARY_FILE_TYPE_UNSPECIFIED", "ANDROID_APK", "ANDROID_AAB", "IOS_IPA":
		return nil
	default:
		return fmt.Errorf("--binary-type must be one of ANDROID_APK, ANDROID_AAB, IOS_IPA")
	}
}

func validateSeverityThreshold(value string) error {
	value = strings.ToUpper(strings.TrimSpace(value))
	if value == "" {
		return nil
	}
	if _, ok := severityRank[value]; !ok {
		return fmt.Errorf("--severity-threshold must be one of PRIORITY, POTENTIAL, OPPORTUNITY")
	}
	return nil
}

func reportExceedsThreshold(report *checksapi.GoogleChecksReportV1alphaReport, threshold string) bool {
	threshold = strings.ToUpper(strings.TrimSpace(threshold))
	if threshold == "" || report == nil {
		return false
	}
	minRank, ok := severityRank[threshold]
	if !ok {
		return false
	}
	for _, check := range report.Checks {
		if check == nil || check.State != "FAILED" {
			continue
		}
		if severityRank[strings.ToUpper(check.Severity)] >= minRank {
			return true
		}
	}
	return false
}

func classificationViolates(resp *checksapi.GoogleChecksAisafetyV1alphaClassifyContentResponse) bool {
	if resp == nil {
		return false
	}
	for _, result := range resp.PolicyResults {
		if result != nil && result.ViolationResult == "VIOLATIVE" {
			return true
		}
	}
	return false
}

func reportFromOperation(op *checksapi.Operation) (*checksapi.GoogleChecksReportV1alphaReport, error) {
	if op == nil || len(op.Response) == 0 {
		return nil, nil
	}
	var report checksapi.GoogleChecksReportV1alphaReport
	if err := json.Unmarshal(op.Response, &report); err != nil {
		return nil, fmt.Errorf("decode operation response as report: %w", err)
	}
	return &report, nil
}

func requireOperationDone(op *checksapi.Operation) error {
	if op == nil {
		return fmt.Errorf("operation response was empty")
	}
	if op.Error != nil {
		return fmt.Errorf("operation failed: %s", op.Error.Message)
	}
	if !op.Done {
		return fmt.Errorf("operation is still running")
	}
	return nil
}

func readTextArg(value string) (string, error) {
	value = strings.TrimSpace(value)
	if strings.HasPrefix(value, "@") {
		path := strings.TrimPrefix(value, "@")
		data, err := os.ReadFile(path)
		if err != nil {
			return "", fmt.Errorf("read text file: %w", err)
		}
		return string(data), nil
	}
	return value, nil
}

func mediaTypeForPath(path string) string {
	switch strings.ToLower(filepath.Ext(path)) {
	case ".apk":
		return "application/vnd.android.package-archive"
	case ".aab":
		return "application/octet-stream"
	case ".ipa":
		return "application/octet-stream"
	default:
		return "application/octet-stream"
	}
}

func parsePolicies(value string) ([]*checksapi.GoogleChecksAisafetyV1alphaClassifyContentRequestPolicyConfig, error) {
	var values []string
	if strings.TrimSpace(value) == "" || strings.EqualFold(strings.TrimSpace(value), "all") {
		values = policyTypes
	} else {
		values = strings.Split(value, ",")
	}
	allowed := make(map[string]bool, len(policyTypes))
	for _, policy := range policyTypes {
		allowed[policy] = true
	}
	policies := make([]*checksapi.GoogleChecksAisafetyV1alphaClassifyContentRequestPolicyConfig, 0, len(values))
	for _, raw := range values {
		policy := strings.ToUpper(strings.TrimSpace(raw))
		if policy == "" {
			continue
		}
		if !allowed[policy] {
			return nil, fmt.Errorf("unsupported policy %q", policy)
		}
		policies = append(policies, &checksapi.GoogleChecksAisafetyV1alphaClassifyContentRequestPolicyConfig{
			PolicyType: policy,
		})
	}
	if len(policies) == 0 {
		return nil, fmt.Errorf("--policies must include at least one policy")
	}
	return policies, nil
}

func formatWaitTimeout(d time.Duration) string {
	if d <= 0 {
		return ""
	}
	return fmt.Sprintf("%.0fs", d.Seconds())
}

func resolveAccountForCommand(flagValue string) (string, error) {
	cfg, err := config.Load()
	if err != nil && !errors.Is(err, config.ErrNotFound) {
		return "", fmt.Errorf("load config: %w", err)
	}
	return checksclient.ResolveAccount(flagValue, cfg), nil
}
