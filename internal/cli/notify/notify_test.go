package notify

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// --- Template / payload tests ---

func TestParseFormat(t *testing.T) {
	tests := []struct {
		input   string
		want    PayloadFormat
		wantErr bool
	}{
		{"slack", FormatSlack, false},
		{"Slack", FormatSlack, false},
		{"SLACK", FormatSlack, false},
		{"", FormatSlack, false},
		{"discord", FormatDiscord, false},
		{"Discord", FormatDiscord, false},
		{"generic", FormatGeneric, false},
		{"unknown", "", true},
		{"xml", "", true},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got, err := ParseFormat(tt.input)
			if tt.wantErr && err == nil {
				t.Fatalf("expected error for input %q", tt.input)
			}
			if !tt.wantErr && err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tt.want {
				t.Errorf("ParseFormat(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestBuildPayload_Slack(t *testing.T) {
	payload := BuildPayload(FormatSlack, "Deployed v1.2.3", "release", "com.example.app")
	sp, ok := payload.(SlackPayload)
	if !ok {
		t.Fatalf("expected SlackPayload, got %T", payload)
	}
	if sp.Text != "Deployed v1.2.3" {
		t.Errorf("text = %q, want %q", sp.Text, "Deployed v1.2.3")
	}
	if len(sp.Blocks) != 1 {
		t.Fatalf("expected 1 block, got %d", len(sp.Blocks))
	}
	if sp.Blocks[0].Type != "section" {
		t.Errorf("block type = %q, want %q", sp.Blocks[0].Type, "section")
	}
	if sp.Blocks[0].Text == nil {
		t.Fatal("block text is nil")
	}
	if sp.Blocks[0].Text.Type != "mrkdwn" {
		t.Errorf("text type = %q, want %q", sp.Blocks[0].Text.Type, "mrkdwn")
	}
	body := sp.Blocks[0].Text.Text
	if !strings.Contains(body, "[release]") {
		t.Errorf("expected body to contain [release], got %q", body)
	}
	if !strings.Contains(body, "`com.example.app`") {
		t.Errorf("expected body to contain `com.example.app`, got %q", body)
	}
	if !strings.Contains(body, "Deployed v1.2.3") {
		t.Errorf("expected body to contain message, got %q", body)
	}
}

func TestBuildPayload_SlackNoMetadata(t *testing.T) {
	payload := BuildPayload(FormatSlack, "Simple message", "", "")
	sp := payload.(SlackPayload)
	body := sp.Blocks[0].Text.Text
	if body != "Simple message" {
		t.Errorf("expected plain message, got %q", body)
	}
}

func TestBuildPayload_Discord(t *testing.T) {
	payload := BuildPayload(FormatDiscord, "New build", "release", "com.example.app")
	dp, ok := payload.(DiscordPayload)
	if !ok {
		t.Fatalf("expected DiscordPayload, got %T", payload)
	}
	if !strings.Contains(dp.Content, "New build") {
		t.Errorf("expected content to contain message, got %q", dp.Content)
	}
	if !strings.Contains(dp.Content, "[release]") {
		t.Errorf("expected content to contain [release], got %q", dp.Content)
	}
}

func TestBuildPayload_Generic(t *testing.T) {
	payload := BuildPayload(FormatGeneric, "Rollout complete", "rollout", "com.example.app")
	gp, ok := payload.(GenericPayload)
	if !ok {
		t.Fatalf("expected GenericPayload, got %T", payload)
	}
	if gp.Message != "Rollout complete" {
		t.Errorf("message = %q, want %q", gp.Message, "Rollout complete")
	}
	if gp.EventType != "rollout" {
		t.Errorf("event_type = %q, want %q", gp.EventType, "rollout")
	}
	if gp.Package != "com.example.app" {
		t.Errorf("package = %q, want %q", gp.Package, "com.example.app")
	}
	if gp.Timestamp == "" {
		t.Error("expected non-empty timestamp")
	}
}

func TestBuildPayload_GenericJSON(t *testing.T) {
	payload := BuildPayload(FormatGeneric, "msg", "release", "com.example")
	data, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("marshal error: %v", err)
	}
	var m map[string]interface{}
	if err := json.Unmarshal(data, &m); err != nil {
		t.Fatalf("unmarshal error: %v", err)
	}
	if m["event_type"] != "release" {
		t.Errorf("event_type = %v", m["event_type"])
	}
	if m["package"] != "com.example" {
		t.Errorf("package = %v", m["package"])
	}
	if m["message"] != "msg" {
		t.Errorf("message = %v", m["message"])
	}
}

// --- Webhook URL validation tests ---

func TestValidateWebhookURL(t *testing.T) {
	tests := []struct {
		name    string
		url     string
		wantErr string
	}{
		{"valid https", "https://hooks.slack.com/services/T00/B00/xxx", ""},
		{"valid http", "http://localhost:8080/webhook", ""},
		{"empty", "", "--webhook-url is required"},
		{"whitespace only", "   ", "--webhook-url is required"},
		{"no scheme", "hooks.slack.com/services/xxx", "webhook URL must use http or https"},
		{"ftp scheme", "ftp://example.com/hook", "webhook URL must use http or https"},
		{"no host", "https://", "webhook URL is missing a host"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateWebhookURL(tt.url)
			if tt.wantErr == "" {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				return
			}
			if err == nil {
				t.Fatal("expected error, got nil")
			}
			if !strings.Contains(err.Error(), tt.wantErr) {
				t.Errorf("error %q does not contain %q", err.Error(), tt.wantErr)
			}
		})
	}
}

// --- Webhook POST tests ---

func TestPostWebhook_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		ct := r.Header.Get("Content-Type")
		if ct != "application/json" {
			t.Errorf("Content-Type = %q, want application/json", ct)
		}
		body, _ := io.ReadAll(r.Body)
		var m map[string]interface{}
		if err := json.Unmarshal(body, &m); err != nil {
			t.Errorf("invalid JSON body: %v", err)
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("ok"))
	}))
	defer srv.Close()

	payload := BuildPayload(FormatSlack, "test", "", "")
	result, err := PostWebhook(context.Background(), srv.Client(), srv.URL, payload)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.StatusCode != 200 {
		t.Errorf("status code = %d, want 200", result.StatusCode)
	}
}

func TestPostWebhook_ServerError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("internal error"))
	}))
	defer srv.Close()

	payload := BuildPayload(FormatGeneric, "test", "", "")
	result, err := PostWebhook(context.Background(), srv.Client(), srv.URL, payload)
	if err == nil {
		t.Fatal("expected error for 500 response")
	}
	if !strings.Contains(err.Error(), "non-success") {
		t.Errorf("error = %q, expected to contain 'non-success'", err.Error())
	}
	if result == nil {
		t.Fatal("expected non-nil result even on error")
	}
	if result.StatusCode != 500 {
		t.Errorf("status code = %d, want 500", result.StatusCode)
	}
}

func TestPostWebhook_VerifiesPayloadFormat(t *testing.T) {
	var receivedBody []byte
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedBody, _ = io.ReadAll(r.Body)
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	// Test Slack payload
	payload := BuildPayload(FormatSlack, "hello", "release", "com.app")
	_, err := PostWebhook(context.Background(), srv.Client(), srv.URL, payload)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	var slackMsg SlackPayload
	if err := json.Unmarshal(receivedBody, &slackMsg); err != nil {
		t.Fatalf("failed to unmarshal slack payload: %v", err)
	}
	if slackMsg.Text != "hello" {
		t.Errorf("slack text = %q, want %q", slackMsg.Text, "hello")
	}

	// Test Generic payload
	payload = BuildPayload(FormatGeneric, "hello", "release", "com.app")
	_, err = PostWebhook(context.Background(), srv.Client(), srv.URL, payload)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	var genericMsg GenericPayload
	if err := json.Unmarshal(receivedBody, &genericMsg); err != nil {
		t.Fatalf("failed to unmarshal generic payload: %v", err)
	}
	if genericMsg.EventType != "release" {
		t.Errorf("event_type = %q, want %q", genericMsg.EventType, "release")
	}
}

// --- MaskURL tests ---

func TestMaskURL(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"https://hooks.slack.com/services/T00/B00/xxxyyy", "https://hooks.slack.com/***xxxyyy"},
		{"https://example.com/a", "https://example.com/a"},
		{"https://example.com/abcdef", "https://example.com/abcdef"},
		{"https://example.com/abcdefg", "https://example.com/***bcdefg"},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := MaskURL(tt.input)
			if got != tt.want {
				t.Errorf("MaskURL(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

// --- Send command integration tests ---

type mockDoer struct {
	handler func(req *http.Request) (*http.Response, error)
}

func (m *mockDoer) Do(req *http.Request) (*http.Response, error) {
	return m.handler(req)
}

func newMockDoer(statusCode int, body string) *mockDoer {
	return &mockDoer{
		handler: func(req *http.Request) (*http.Response, error) {
			return &http.Response{
				StatusCode: statusCode,
				Status:     http.StatusText(statusCode),
				Body:       io.NopCloser(strings.NewReader(body)),
				Header:     make(http.Header),
			}, nil
		},
	}
}

func TestRunSend_MissingWebhookURL(t *testing.T) {
	err := runSend(context.Background(), sendOpts{
		webhookURL: "",
		message:    "hello",
		format:     "slack",
		outputFlag: "json",
		client:     newMockDoer(200, "ok"),
	})
	if err == nil {
		t.Fatal("expected error for missing webhook URL")
	}
	if !strings.Contains(err.Error(), "--webhook-url is required") {
		t.Errorf("error = %q, want to contain '--webhook-url is required'", err.Error())
	}
}

func TestRunSend_MissingMessage(t *testing.T) {
	err := runSend(context.Background(), sendOpts{
		webhookURL: "https://hooks.slack.com/services/T00/B00/xxx",
		message:    "",
		format:     "slack",
		outputFlag: "json",
		client:     newMockDoer(200, "ok"),
	})
	if err == nil {
		t.Fatal("expected error for missing message")
	}
	if !strings.Contains(err.Error(), "--message is required") {
		t.Errorf("error = %q, want to contain '--message is required'", err.Error())
	}
}

func TestRunSend_InvalidFormat(t *testing.T) {
	err := runSend(context.Background(), sendOpts{
		webhookURL: "https://hooks.slack.com/services/T00/B00/xxx",
		message:    "hello",
		format:     "xml",
		outputFlag: "json",
		client:     newMockDoer(200, "ok"),
	})
	if err == nil {
		t.Fatal("expected error for invalid format")
	}
	if !strings.Contains(err.Error(), "unsupported format") {
		t.Errorf("error = %q, want to contain 'unsupported format'", err.Error())
	}
}

func TestRunSend_InvalidWebhookURL(t *testing.T) {
	err := runSend(context.Background(), sendOpts{
		webhookURL: "ftp://example.com/hook",
		message:    "hello",
		format:     "slack",
		outputFlag: "json",
		client:     newMockDoer(200, "ok"),
	})
	if err == nil {
		t.Fatal("expected error for invalid webhook URL scheme")
	}
	if !strings.Contains(err.Error(), "http or https") {
		t.Errorf("error = %q, want to contain 'http or https'", err.Error())
	}
}

func TestRunSend_InvalidOutputFlags(t *testing.T) {
	err := runSend(context.Background(), sendOpts{
		webhookURL: "https://hooks.slack.com/services/T00/B00/xxx",
		message:    "hello",
		format:     "slack",
		outputFlag: "table",
		pretty:     true,
		client:     newMockDoer(200, "ok"),
	})
	if err == nil {
		t.Fatal("expected error for --pretty with table output")
	}
	if !strings.Contains(err.Error(), "--pretty is only valid with JSON") {
		t.Errorf("error = %q, want to contain '--pretty is only valid with JSON'", err.Error())
	}
}

func TestRunSend_Success(t *testing.T) {
	// Capture stdout for output verification
	var capturedBody []byte
	mock := &mockDoer{
		handler: func(req *http.Request) (*http.Response, error) {
			capturedBody, _ = io.ReadAll(req.Body)
			return &http.Response{
				StatusCode: 200,
				Status:     "200 OK",
				Body:       io.NopCloser(strings.NewReader("ok")),
				Header:     make(http.Header),
			}, nil
		},
	}

	err := runSend(context.Background(), sendOpts{
		webhookURL:  "https://hooks.slack.com/services/T00/B00/xxx",
		message:     "Build deployed",
		format:      "slack",
		eventType:   "release",
		packageName: "com.example.app",
		outputFlag:  "json",
		client:      mock,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify payload sent
	var sp SlackPayload
	if err := json.Unmarshal(capturedBody, &sp); err != nil {
		t.Fatalf("failed to unmarshal sent payload: %v", err)
	}
	if sp.Text != "Build deployed" {
		t.Errorf("sent text = %q, want %q", sp.Text, "Build deployed")
	}
}

func TestRunSend_WebhookError(t *testing.T) {
	err := runSend(context.Background(), sendOpts{
		webhookURL: "https://hooks.slack.com/services/T00/B00/xxx",
		message:    "hello",
		format:     "slack",
		outputFlag: "json",
		client:     newMockDoer(403, "forbidden"),
	})
	if err == nil {
		t.Fatal("expected error for 403 response")
	}
	if !strings.Contains(err.Error(), "notification failed") {
		t.Errorf("error = %q, want to contain 'notification failed'", err.Error())
	}
}

func TestRunSend_AllFormats(t *testing.T) {
	formats := []string{"slack", "discord", "generic"}
	for _, format := range formats {
		t.Run(format, func(t *testing.T) {
			var capturedBody []byte
			mock := &mockDoer{
				handler: func(req *http.Request) (*http.Response, error) {
					capturedBody, _ = io.ReadAll(req.Body)
					return &http.Response{
						StatusCode: 200,
						Status:     "200 OK",
						Body:       io.NopCloser(strings.NewReader("ok")),
						Header:     make(http.Header),
					}, nil
				},
			}

			err := runSend(context.Background(), sendOpts{
				webhookURL:  "https://example.com/webhook",
				message:     "test message",
				format:      format,
				eventType:   "test",
				packageName: "com.test",
				outputFlag:  "json",
				client:      mock,
			})
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if len(capturedBody) == 0 {
				t.Fatal("expected non-empty body")
			}

			// Verify valid JSON
			var raw json.RawMessage
			if err := json.Unmarshal(capturedBody, &raw); err != nil {
				t.Errorf("sent body is not valid JSON: %v", err)
			}
		})
	}
}

// --- NotifyCommand structure test ---

func TestNotifyCommand_HasSubcommands(t *testing.T) {
	cmd := NotifyCommand()
	if cmd.Name != "notify" {
		t.Errorf("name = %q, want %q", cmd.Name, "notify")
	}
	if len(cmd.Subcommands) == 0 {
		t.Fatal("expected subcommands")
	}
	found := false
	for _, sub := range cmd.Subcommands {
		if sub.Name == "send" {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected 'send' subcommand")
	}
}

func TestNotifyCommand_NoArgsReturnsHelp(t *testing.T) {
	cmd := NotifyCommand()
	err := cmd.Exec(context.Background(), nil)
	if err == nil {
		t.Fatal("expected error when no args")
	}
}

// --- End-to-end test with httptest server ---

func TestSendEndToEnd_SlackWebhook(t *testing.T) {
	var receivedPayload bytes.Buffer
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		io.Copy(&receivedPayload, r.Body)
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	err := runSend(context.Background(), sendOpts{
		webhookURL:  srv.URL,
		message:     "Release v2.0 is live!",
		format:      "slack",
		eventType:   "release",
		packageName: "com.example.myapp",
		outputFlag:  "json",
		client:      srv.Client(),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var sp SlackPayload
	if err := json.Unmarshal(receivedPayload.Bytes(), &sp); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}
	if sp.Text != "Release v2.0 is live!" {
		t.Errorf("text = %q", sp.Text)
	}
	blockText := sp.Blocks[0].Text.Text
	if !strings.Contains(blockText, "[release]") {
		t.Errorf("block text missing event type: %q", blockText)
	}
	if !strings.Contains(blockText, "`com.example.myapp`") {
		t.Errorf("block text missing package: %q", blockText)
	}
}

func TestSendEndToEnd_GenericWebhook(t *testing.T) {
	var receivedPayload bytes.Buffer
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		io.Copy(&receivedPayload, r.Body)
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	err := runSend(context.Background(), sendOpts{
		webhookURL:  srv.URL,
		message:     "Rollout at 50%",
		format:      "generic",
		eventType:   "rollout",
		packageName: "com.example.myapp",
		outputFlag:  "json",
		client:      srv.Client(),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var gp GenericPayload
	if err := json.Unmarshal(receivedPayload.Bytes(), &gp); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}
	if gp.Message != "Rollout at 50%" {
		t.Errorf("message = %q", gp.Message)
	}
	if gp.EventType != "rollout" {
		t.Errorf("event_type = %q", gp.EventType)
	}
	if gp.Package != "com.example.myapp" {
		t.Errorf("package = %q", gp.Package)
	}
}
