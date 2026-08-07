package webclient

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
)

// DefaultAppsBaseURL is Play Console's apps backend. Unlike the console
// origin this is a OnePlatform API host: REST paths carrying JSON+protobuf,
// authenticated with the session's SAPISIDHASH plus a public web API key. The
// Play Console UI uses these endpoints; it does not use batchexecute at all.
const DefaultAppsBaseURL = "https://playconsoleapps-pa.clients6.google.com"

// appsAPIKeyEnv overrides the public API key discovered from console startup
// data.
const appsAPIKeyEnv = "GPLAY_WEB_API_KEY"

// maxAppPages bounds pagination so a misbehaving nextPageToken cannot spin
// forever.
const maxAppPages = 100

// App is one entry of the Play Console app list.
type App struct {
	AppID       string `json:"appId"`
	PackageName string `json:"packageName"`
	DisplayName string `json:"displayName"`
	DeveloperID string `json:"developerId,omitempty"`
	Language    string `json:"defaultLanguage,omitempty"`
}

// appSummariesWire mirrors ListAppSummariesResponse. Field numbers come from
// the console's own schema, so the odd JSON tags are the wire contract.
type appSummariesWire struct {
	Apps          []appSummaryWire `json:"1"`
	NextPageToken string           `json:"2"`
}

type appSummaryWire struct {
	Reference struct {
		Developer struct {
			Value string `json:"1"`
		} `json:"1"`
		App struct {
			Value string `json:"1"`
		} `json:"2"`
	} `json:"1"`
	DisplayName string `json:"2"`
	PackageName string `json:"5"`
	Language    string `json:"16"`
}

// apiKey returns the environment override or discovers the public key from the
// console's startup data.
func (c *Client) apiKey(ctx context.Context) (string, error) {
	if v := strings.TrimSpace(os.Getenv(appsAPIKeyEnv)); v != "" {
		return v, nil
	}
	c.appsAPIKeyMu.Lock()
	defer c.appsAPIKeyMu.Unlock()
	if c.appsAPIKey != "" {
		return c.appsAPIKey, nil
	}
	body, _, err := c.getConsole(ctx)
	if err != nil {
		return "", fmt.Errorf("discovering Play Console API key: %w", err)
	}
	data, err := parseStartupData(body)
	if err != nil {
		return "", fmt.Errorf("discovering Play Console API key: %w (set %s to override)", err, appsAPIKeyEnv)
	}
	c.appsAPIKey = strings.TrimSpace(data.Boot.APIKey)
	if c.appsAPIKey == "" {
		return "", fmt.Errorf("no Play Console API key found in startupData; set %s to override", appsAPIKeyEnv)
	}
	return c.appsAPIKey, nil
}

// ListApps returns every app in the developer account, following pagination.
// This is the only way to enumerate apps for a brand-new app: the official
// Publisher API has no list endpoint and the Reporting API only surfaces apps
// that already have reporting data.
func (c *Client) ListApps(ctx context.Context, developerID string) ([]App, error) {
	developerID = strings.TrimSpace(developerID)
	if developerID == "" {
		return nil, errors.New("developer ID is required")
	}
	var (
		apps      []App
		pageToken string
	)
	for page := 0; page < maxAppPages; page++ {
		batch, next, err := c.listAppsPage(ctx, developerID, pageToken)
		if err != nil {
			return nil, err
		}
		apps = append(apps, batch...)
		if next == "" {
			return apps, nil
		}
		pageToken = next
	}
	return apps, nil
}

// listAppsPage fetches one page of app summaries.
func (c *Client) listAppsPage(ctx context.Context, developerID, pageToken string) ([]App, string, error) {
	endpoint := c.appsBaseURL + "/v1/developers/" + url.PathEscape(developerID) + "/appSummaries"
	if pageToken != "" {
		endpoint += "?" + url.Values{"pageToken": {pageToken}}.Encode()
	}
	req, err := c.apiRequest(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, "", err
	}
	resp, err := c.http.Do(req)
	if err != nil {
		return nil, "", fmt.Errorf("listing apps: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden {
		_, _ = io.Copy(io.Discard, resp.Body)
		return nil, "", fmt.Errorf("%w: listing apps for developer %s returned HTTP %d (session expired, or this account cannot access that developer — run gplay web auth login)", ErrAuth, developerID, resp.StatusCode)
	}
	if resp.StatusCode != http.StatusOK {
		_, _ = io.Copy(io.Discard, resp.Body)
		return nil, "", fmt.Errorf("listing apps returned HTTP %d", resp.StatusCode)
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, "", err
	}

	var wire appSummariesWire
	if err := json.Unmarshal(body, &wire); err != nil {
		return nil, "", fmt.Errorf("parsing app list response: %w", err)
	}
	apps := make([]App, 0, len(wire.Apps))
	for _, a := range wire.Apps {
		apps = append(apps, App{
			AppID:       a.Reference.App.Value,
			PackageName: a.PackageName,
			DisplayName: a.DisplayName,
			DeveloperID: a.Reference.Developer.Value,
			Language:    a.Language,
		})
	}
	return apps, wire.NextPageToken, nil
}

// apiRequest builds an authenticated request to a Play Console -pa backend.
//
// Two details are load-bearing and were established against the live API:
// the cookie jar keys the session to play.google.com so it never serves these
// .google.com account cookies to an API host (hence the manual Cookie header),
// and Referer must be present because the web API key is referrer-restricted.
// X-Goog-AuthUser is deliberately never sent: it makes the backend reject the
// session cookie outright, even when it names the correct account index.
func (c *Client) apiRequest(ctx context.Context, method, endpoint string, body io.Reader) (*http.Request, error) {
	req, err := http.NewRequestWithContext(ctx, method, endpoint, body)
	if err != nil {
		return nil, err
	}
	key, err := c.apiKey(ctx)
	if err != nil {
		return nil, err
	}
	if header := c.cookieHeader(); header != "" {
		req.Header.Set("Cookie", header)
	}
	if h := c.authHeader(); h != "" {
		req.Header.Set("Authorization", h)
	}
	req.Header.Set("X-Goog-Api-Key", key)
	req.Header.Set("Content-Type", "application/json+protobuf")
	req.Header.Set("Origin", authOrigin)
	req.Header.Set("Referer", authOrigin+"/")
	return req, nil
}

// developerPath builds a custom-method URL: /v1/developers/<id>:<verb>.
func (c *Client) developerPath(developerID, verb string) string {
	return c.appsBaseURL + "/v1/developers/" + url.PathEscape(developerID) + ":" + verb
}

// PackageAvailability is the console's package-name availability enum.
type PackageAvailability int

const (
	PackageAvailabilityUnspecified PackageAvailability = 0
	PackageAvailabilityAvailable   PackageAvailability = 1
	PackageAvailabilityPlayOwned   PackageAvailability = 2
	PackageAvailabilityPlayOnly    PackageAvailability = 3
	PackageAvailabilityUnavailable PackageAvailability = 4
)

// Available reports whether a new app may claim this package name.
func (a PackageAvailability) Available() bool { return a == PackageAvailabilityAvailable }

func (a PackageAvailability) String() string {
	switch a {
	case PackageAvailabilityAvailable:
		return "available"
	case PackageAvailabilityPlayOwned:
		return "owned by you on Play"
	case PackageAvailabilityPlayOnly:
		return "reserved on Play"
	case PackageAvailabilityUnavailable:
		return "already taken"
	default:
		return "unknown"
	}
}

// CheckPackageName reports whether a package name can be claimed. It is a
// read-only call, so it is safe to run before the irreversible create.
func (c *Client) CheckPackageName(ctx context.Context, developerID, packageName string) (PackageAvailability, error) {
	developerID = strings.TrimSpace(developerID)
	packageName = strings.TrimSpace(packageName)
	if developerID == "" || packageName == "" {
		return 0, errors.New("developer ID and package name are required")
	}
	endpoint := c.developerPath(developerID, "checkPackageNameAvailability") +
		"?" + url.Values{"packageName": {packageName}}.Encode()
	req, err := c.apiRequest(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return 0, err
	}
	body, err := c.doAPI(req, "checking package name")
	if err != nil {
		return 0, err
	}
	var wire struct {
		Availability int `json:"1"`
	}
	if err := json.Unmarshal(body, &wire); err != nil {
		return 0, fmt.Errorf("parsing availability response: %w", err)
	}
	return PackageAvailability(wire.Availability), nil
}

// AppKind is the Play Console "app or game" choice.
type AppKind string

const (
	AppKindApp  AppKind = "app"
	AppKindGame AppKind = "game"
)

// appType maps the kind onto the console's APP_TYPE enum.
func (k AppKind) appType() int {
	if k == AppKindGame {
		return 2 // APP_TYPE_GAME
	}
	return 1 // APP_TYPE_APP
}

// playAppClassificationStandard is PLAY_APP_CLASSIFICATION_STANDARD, the
// classification the console sends for an ordinary app.
const playAppClassificationStandard = 1

// CreateAppRequest describes a Play Console "Create app" submission. The
// fields mirror the console dialog one-for-one.
type CreateAppRequest struct {
	DeveloperID     string
	Title           string
	DefaultLanguage string
	Kind            AppKind
	Paid            bool
	PackageName     string
	// MeetsGuidelines and USLawCompliant are the two legal declarations the
	// console requires; both must be affirmed by the operator.
	MeetsGuidelines bool
	USLawCompliant  bool
}

func (r CreateAppRequest) validate() error {
	switch {
	case strings.TrimSpace(r.DeveloperID) == "":
		return errors.New("developer ID is required")
	case strings.TrimSpace(r.Title) == "":
		return errors.New("app title is required")
	case strings.TrimSpace(r.DefaultLanguage) == "":
		return errors.New("default language is required")
	case strings.TrimSpace(r.PackageName) == "":
		return errors.New("package name is required")
	case r.Kind != AppKindApp && r.Kind != AppKindGame:
		return fmt.Errorf("kind must be %q or %q", AppKindApp, AppKindGame)
	case !r.MeetsGuidelines || !r.USLawCompliant:
		return errors.New("both declarations must be accepted: app meets Developer Program Policies, and US export law compliance")
	}
	return nil
}

// CreateApp creates a new app and returns its identifiers. The request body
// uses the console's numeric protobuf field keys.
func (c *Client) CreateApp(ctx context.Context, req CreateAppRequest) (*App, error) {
	if err := req.validate(); err != nil {
		return nil, err
	}
	payload, err := json.Marshal(map[string]any{
		// developer_id is a DeveloperId message, not a bare string: the path
		// template {developer_id.value} shows the value nests in field 1.
		"1": map[string]any{"1": strings.TrimSpace(req.DeveloperID)},
		"2": strings.TrimSpace(req.Title),
		"3": strings.TrimSpace(req.DefaultLanguage),
		"4": req.Kind.appType(),
		"5": req.Paid,
		"6": req.MeetsGuidelines,
		"7": req.USLawCompliant,
		"8": playAppClassificationStandard,
		// PlatformDistributionSettings: distribute on Android, not Windows.
		"9":  map[string]any{"1": true, "2": false},
		"11": strings.TrimSpace(req.PackageName),
	})
	if err != nil {
		return nil, fmt.Errorf("encoding create request: %w", err)
	}

	endpoint := c.developerPath(req.DeveloperID, "createAppV2")
	httpReq, err := c.apiRequest(ctx, http.MethodPost, endpoint, bytes.NewReader(payload))
	if err != nil {
		return nil, err
	}
	body, err := c.doAPI(httpReq, "creating app")
	if err != nil {
		return nil, err
	}
	var wire struct {
		Reference struct {
			Developer struct {
				Value string `json:"1"`
			} `json:"1"`
			App struct {
				Value string `json:"1"`
			} `json:"2"`
		} `json:"1"`
	}
	if err := json.Unmarshal(body, &wire); err != nil {
		return nil, fmt.Errorf("parsing create response: %w", err)
	}
	return &App{
		AppID:       wire.Reference.App.Value,
		DeveloperID: wire.Reference.Developer.Value,
		PackageName: strings.TrimSpace(req.PackageName),
		DisplayName: strings.TrimSpace(req.Title),
		Language:    strings.TrimSpace(req.DefaultLanguage),
	}, nil
}

// doAPI executes a built request and returns the body, mapping auth-class
// failures onto ErrAuth.
func (c *Client) doAPI(req *http.Request, action string) ([]byte, error) {
	resp, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", action, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden {
		_, _ = io.Copy(io.Discard, resp.Body)
		return nil, fmt.Errorf("%w: %s returned HTTP %d (run gplay web auth login)", ErrAuth, action, resp.StatusCode)
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		snippet := string(body)
		if len(snippet) > 300 {
			snippet = snippet[:300]
		}
		return nil, fmt.Errorf("%s returned HTTP %d: %s", action, resp.StatusCode, snippet)
	}
	return body, nil
}
