package playclient

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	"google.golang.org/api/googleapi"
)

// UpdateAppStoreHostedApp preserves the current discovery request and response.
// The generated SDK lacks activeApkSets.versionCode/alreadyPublishedOnPlay and
// response.updateId; raw JSON also retains explicit false policy answers.
func (s *Service) UpdateAppStoreHostedApp(ctx context.Context, store string, body json.RawMessage) (map[string]json.RawMessage, error) {
	if s.HTTPClient == nil {
		return nil, fmt.Errorf("missing authenticated HTTP client")
	}
	endpoint := strings.TrimRight(s.API.BasePath, "/") + "/androidpublisher/v3/appstore/" + url.PathEscape(store) + "/apps:update"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := s.HTTPClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if err := googleapi.CheckResponse(resp); err != nil {
		return nil, err
	}
	var result map[string]json.RawMessage
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil && err != io.EOF {
		return nil, err
	}
	return result, nil
}
