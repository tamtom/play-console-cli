package oauth

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
)

const (
	// Google OAuth endpoints
	authURL  = "https://accounts.google.com/o/oauth2/v2/auth"
	tokenURL = "https://oauth2.googleapis.com/token"

	// Default OAuth client (Google's "installed app" client for CLI tools)
	// Users can override with their own client ID/secret
	defaultClientID     = "764086051850-6qr4p6gpi6hn506pt8ejuq83di341hur.apps.googleusercontent.com"
	defaultClientSecret = "d-FL95Q19q7MQmFpd7hHD0Ty"
)

var scopes = []string{"https://www.googleapis.com/auth/androidpublisher"}

// BrowserFlowOptions configures the browser OAuth flow.
type BrowserFlowOptions struct {
	ClientID     string
	ClientSecret string
	Timeout      time.Duration
}

// BrowserFlowResult contains the result of a successful browser OAuth flow.
type BrowserFlowResult struct {
	Token       *oauth2.Token
	TokenPath   string
	ClientID    string
	ClientSecret string
}

// RunBrowserFlow starts the OAuth browser flow and returns credentials.
func RunBrowserFlow(ctx context.Context, profileName string, opts BrowserFlowOptions) (*BrowserFlowResult, error) {
	clientID := opts.ClientID
	clientSecret := opts.ClientSecret
	if clientID == "" {
		clientID = defaultClientID
	}
	if clientSecret == "" {
		clientSecret = defaultClientSecret
	}

	// Find available port
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return nil, fmt.Errorf("failed to start local server: %w", err)
	}
	port := listener.Addr().(*net.TCPAddr).Port
	redirectURI := fmt.Sprintf("http://127.0.0.1:%d/callback", port)

	// Generate PKCE verifier and challenge
	verifier, challenge, err := generatePKCE()
	if err != nil {
		listener.Close()
		return nil, fmt.Errorf("failed to generate PKCE: %w", err)
	}

	// Generate state for CSRF protection
	state, err := generateState()
	if err != nil {
		listener.Close()
		return nil, fmt.Errorf("failed to generate state: %w", err)
	}

	// Build authorization URL
	authURL := buildAuthURL(clientID, redirectURI, state, challenge)

	// Channel to receive auth result
	resultCh := make(chan authResult, 1)

	// Start local server
	server := &http.Server{
		Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			handleCallback(w, r, state, resultCh)
		}),
	}

	go func() {
		_ = server.Serve(listener)
	}()

	defer func() {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		_ = server.Shutdown(ctx)
	}()

	// Open browser
	fmt.Println("Opening browser for Google authentication...")
	fmt.Printf("If the browser doesn't open, visit this URL:\n%s\n\n", authURL)
	if err := openBrowser(authURL); err != nil {
		fmt.Printf("Failed to open browser: %v\n", err)
	}

	// Wait for callback
	timeout := opts.Timeout
	if timeout == 0 {
		timeout = 5 * time.Minute
	}

	select {
	case result := <-resultCh:
		if result.err != nil {
			return nil, result.err
		}

		// Exchange code for token
		token, err := exchangeCode(ctx, clientID, clientSecret, redirectURI, result.code, verifier)
		if err != nil {
			return nil, fmt.Errorf("failed to exchange code: %w", err)
		}

		// Save token
		tokenPath, err := saveToken(profileName, token)
		if err != nil {
			return nil, fmt.Errorf("failed to save token: %w", err)
		}

		return &BrowserFlowResult{
			Token:        token,
			TokenPath:    tokenPath,
			ClientID:     clientID,
			ClientSecret: clientSecret,
		}, nil

	case <-time.After(timeout):
		return nil, fmt.Errorf("authentication timed out after %v", timeout)

	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

type authResult struct {
	code string
	err  error
}

func handleCallback(w http.ResponseWriter, r *http.Request, expectedState string, resultCh chan<- authResult) {
	query := r.URL.Query()

	// Check for error
	if errParam := query.Get("error"); errParam != "" {
		errDesc := query.Get("error_description")
		resultCh <- authResult{err: fmt.Errorf("OAuth error: %s - %s", errParam, errDesc)}
		http.Error(w, "Authentication failed: "+errDesc, http.StatusBadRequest)
		return
	}

	// Verify state
	if query.Get("state") != expectedState {
		resultCh <- authResult{err: fmt.Errorf("invalid state parameter")}
		http.Error(w, "Invalid state parameter", http.StatusBadRequest)
		return
	}

	// Get authorization code
	code := query.Get("code")
	if code == "" {
		resultCh <- authResult{err: fmt.Errorf("missing authorization code")}
		http.Error(w, "Missing authorization code", http.StatusBadRequest)
		return
	}

	// Success page
	w.Header().Set("Content-Type", "text/html")
	fmt.Fprint(w, `<!DOCTYPE html>
<html>
<head>
    <title>gplay - Authentication Successful</title>
    <style>
        body {
            font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif;
            display: flex;
            justify-content: center;
            align-items: center;
            height: 100vh;
            margin: 0;
            background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
        }
        .card {
            background: white;
            padding: 40px 60px;
            border-radius: 16px;
            box-shadow: 0 10px 40px rgba(0,0,0,0.2);
            text-align: center;
        }
        .checkmark {
            width: 80px;
            height: 80px;
            border-radius: 50%;
            background: #4CAF50;
            margin: 0 auto 20px;
            display: flex;
            align-items: center;
            justify-content: center;
        }
        .checkmark svg {
            width: 40px;
            height: 40px;
            fill: white;
        }
        h1 { color: #333; margin-bottom: 10px; }
        p { color: #666; }
    </style>
</head>
<body>
    <div class="card">
        <div class="checkmark">
            <svg viewBox="0 0 24 24"><path d="M9 16.17L4.83 12l-1.42 1.41L9 19 21 7l-1.41-1.41z"/></svg>
        </div>
        <h1>Authentication Successful!</h1>
        <p>You can close this window and return to the terminal.</p>
    </div>
</body>
</html>`)

	resultCh <- authResult{code: code}
}

func buildAuthURL(clientID, redirectURI, state, challenge string) string {
	params := url.Values{
		"client_id":             {clientID},
		"redirect_uri":          {redirectURI},
		"response_type":         {"code"},
		"scope":                 {strings.Join(scopes, " ")},
		"state":                 {state},
		"code_challenge":        {challenge},
		"code_challenge_method": {"S256"},
		"access_type":           {"offline"},
		"prompt":                {"consent"},
	}
	return authURL + "?" + params.Encode()
}

func exchangeCode(ctx context.Context, clientID, clientSecret, redirectURI, code, verifier string) (*oauth2.Token, error) {
	cfg := &oauth2.Config{
		ClientID:     clientID,
		ClientSecret: clientSecret,
		Endpoint:     google.Endpoint,
		RedirectURL:  redirectURI,
		Scopes:       scopes,
	}

	return cfg.Exchange(ctx, code, oauth2.SetAuthURLParam("code_verifier", verifier))
}

func saveToken(profileName string, token *oauth2.Token) (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}

	dir := filepath.Join(home, ".gplay", "tokens")
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return "", err
	}

	tokenPath := filepath.Join(dir, fmt.Sprintf("%s.json", profileName))
	data, err := json.MarshalIndent(token, "", "  ")
	if err != nil {
		return "", err
	}

	if err := os.WriteFile(tokenPath, data, 0o600); err != nil {
		return "", err
	}

	return tokenPath, nil
}

func generatePKCE() (verifier, challenge string, err error) {
	// Generate 32 random bytes for verifier
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", "", err
	}
	verifier = base64.RawURLEncoding.EncodeToString(b)

	// SHA256 hash for challenge
	h := sha256.Sum256([]byte(verifier))
	challenge = base64.RawURLEncoding.EncodeToString(h[:])

	return verifier, challenge, nil
}

func generateState() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

func openBrowser(url string) error {
	var cmd *exec.Cmd

	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", url)
	case "linux":
		cmd = exec.Command("xdg-open", url)
	case "windows":
		cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", url)
	default:
		return fmt.Errorf("unsupported platform: %s", runtime.GOOS)
	}

	return cmd.Start()
}
