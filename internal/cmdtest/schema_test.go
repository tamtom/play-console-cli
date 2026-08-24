package cmdtest_test

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestSchema_InspectsOfficialEndpointOffline(t *testing.T) {
	stdout, stderr, err := runCommand(t, "schema", "androidpublisher.orders.batchget")
	if err != nil {
		t.Fatalf("schema failed: %v\nstderr: %s", err, stderr)
	}

	var response struct {
		Count   int `json:"count"`
		Results []struct {
			API          string `json:"api"`
			ID           string `json:"id"`
			HTTPMethod   string `json:"http_method"`
			Path         string `json:"path"`
			ResponseType string `json:"response_type"`
			Parameters   []struct {
				Name     string `json:"name"`
				Location string `json:"location"`
				Repeated bool   `json:"repeated"`
			} `json:"parameters"`
		} `json:"results"`
	}
	if err := json.Unmarshal([]byte(stdout), &response); err != nil {
		t.Fatalf("decode schema output: %v\nstdout: %s", err, stdout)
	}
	if response.Count != 1 || len(response.Results) != 1 {
		t.Fatalf("result count = %d, want 1; stdout: %s", response.Count, stdout)
	}
	endpoint := response.Results[0]
	if endpoint.API != "androidpublisher" || endpoint.ID != "androidpublisher.orders.batchget" {
		t.Fatalf("unexpected endpoint identity: %#v", endpoint)
	}
	if endpoint.HTTPMethod != "GET" || endpoint.Path != "androidpublisher/v3/applications/{packageName}/orders:batchGet" {
		t.Fatalf("unexpected endpoint transport: %#v", endpoint)
	}
	if endpoint.ResponseType != "BatchGetOrdersResponse" {
		t.Fatalf("response type = %q", endpoint.ResponseType)
	}
	var foundOrderIDs bool
	for _, parameter := range endpoint.Parameters {
		if parameter.Name == "orderIds" {
			foundOrderIDs = parameter.Location == "query" && parameter.Repeated
		}
	}
	if !foundOrderIDs {
		t.Fatalf("orderIds query parameter missing from %#v", endpoint.Parameters)
	}
}

func TestSchema_QueriesEveryEmbeddedPlayAPI(t *testing.T) {
	stdout, stderr, err := runCommand(t, "schema", "--api", "playintegrity", "--method", "POST", "decodeIntegrityToken")
	if err != nil {
		t.Fatalf("schema failed: %v\nstderr: %s", err, stderr)
	}
	if !strings.Contains(stdout, `"id":"playintegrity.decodeIntegrityToken"`) {
		t.Fatalf("expected Play Integrity endpoint, got: %s", stdout)
	}
}

func TestSchema_InspectsRequestTypes(t *testing.T) {
	stdout, stderr, err := runCommand(t, "schema", "--api", "androidpublisher", "--type", "OrdersReviewRefundRequest")
	if err != nil {
		t.Fatalf("schema failed: %v\nstderr: %s", err, stderr)
	}
	if !strings.Contains(stdout, `"refundPreference"`) || !strings.Contains(stdout, `"pendingRefundToken"`) {
		t.Fatalf("expected request fields in schema output, got: %s", stdout)
	}
}

func TestSchema_RequiresQueryOrList(t *testing.T) {
	_, stderr, err := runCommand(t, "schema")
	if err == nil {
		t.Fatal("expected missing query error")
	}
	if !strings.Contains(stderr, "schema query is required") {
		t.Fatalf("unexpected stderr: %s", stderr)
	}
}

func TestDocsGenerate_UsesTheSameCatalogAsRootHelp(t *testing.T) {
	stdout, stderr, err := runCommand(t, "docs", "generate", "--output-file", "-")
	if err != nil {
		t.Fatalf("docs generate failed: %v\nstderr: %s", err, stderr)
	}
	for _, heading := range []string{"## gplay search", "## gplay schema"} {
		if !strings.Contains(stdout, heading) {
			t.Fatalf("generated docs missing %q", heading)
		}
	}
}

func TestCompletionGeneration_UsesTheSameCatalogAsRootHelp(t *testing.T) {
	for _, shell := range []string{"bash", "zsh", "fish", "powershell"} {
		t.Run(shell, func(t *testing.T) {
			stdout, stderr, err := runCommand(t, "completion", shell)
			if err != nil {
				t.Fatalf("completion %s failed: %v\nstderr: %s", shell, err, stderr)
			}
			for _, command := range []string{"search", "schema", "app-stores", "verification"} {
				if !strings.Contains(stdout, command) {
					t.Fatalf("%s completion missing %q", shell, command)
				}
			}
		})
	}
}
