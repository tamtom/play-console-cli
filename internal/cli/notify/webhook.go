package notify

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
)

// WebhookResult holds the response from a webhook POST.
type WebhookResult struct {
	Status     string `json:"status"`
	StatusCode int    `json:"status_code"`
	WebhookURL string `json:"webhook_url"`
	Format     string `json:"format"`
}

// HTTPDoer abstracts http.Client for testability.
type HTTPDoer interface {
	Do(req *http.Request) (*http.Response, error)
}

// ValidateWebhookURL checks that the URL is a valid http(s) URL.
func ValidateWebhookURL(raw string) error {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return fmt.Errorf("--webhook-url is required")
	}
	u, err := url.Parse(raw)
	if err != nil {
		return fmt.Errorf("invalid webhook URL: %w", err)
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return fmt.Errorf("webhook URL must use http or https scheme, got %q", u.Scheme)
	}
	if strings.TrimSpace(u.Host) == "" {
		return fmt.Errorf("webhook URL is missing a host")
	}
	return nil
}

// PostWebhook sends the payload as JSON to the webhook URL.
func PostWebhook(ctx context.Context, client HTTPDoer, webhookURL string, payload interface{}) (*WebhookResult, error) {
	body, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal payload: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, webhookURL, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("webhook request failed: %w", err)
	}
	defer resp.Body.Close()
	io.Copy(io.Discard, resp.Body)

	result := &WebhookResult{
		Status:     resp.Status,
		StatusCode: resp.StatusCode,
		WebhookURL: webhookURL,
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return result, fmt.Errorf("webhook returned non-success status: %s", resp.Status)
	}

	return result, nil
}

// MaskURL redacts all but the last 6 characters of the URL path for safe display.
// The leading "/" of the path is always preserved.
func MaskURL(raw string) string {
	u, err := url.Parse(raw)
	if err != nil {
		return "***"
	}
	path := u.Path
	// Strip leading slash for masking, then re-add it.
	trimmed := strings.TrimPrefix(path, "/")
	if len(trimmed) > 6 {
		trimmed = "***" + trimmed[len(trimmed)-6:]
	}
	if strings.HasPrefix(path, "/") {
		path = "/" + trimmed
	} else {
		path = trimmed
	}
	return fmt.Sprintf("%s://%s%s", u.Scheme, u.Host, path)
}
