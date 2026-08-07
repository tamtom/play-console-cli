// Package webdriver drives a real Chrome instance over the Chrome DevTools
// Protocol.
//
// Play Console's create-app request body is undocumented; a hand-built payload
// is rejected by the server. Driving the console's own UI is the reliable path:
// whatever the frontend sends is by definition correct, and it keeps working
// when Google changes the wire format.
package webdriver

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"golang.org/x/net/websocket"
)

// devToolsPortFile is written by Chrome inside the user-data directory once
// the debugging endpoint is listening. Its first line is the port.
const devToolsPortFile = "DevToolsActivePort"

// Browser is a connected Chrome instance.
type Browser struct {
	conn *websocket.Conn
	mu   sync.Mutex
	next int
}

// target is one debuggable page as reported by /json/list.
type target struct {
	Type                 string `json:"type"`
	URL                  string `json:"url"`
	WebSocketDebuggerURL string `json:"webSocketDebuggerUrl"`
}

// cdpRequest is a DevTools Protocol command.
type cdpRequest struct {
	ID     int            `json:"id"`
	Method string         `json:"method"`
	Params map[string]any `json:"params,omitempty"`
}

// cdpResponse is a command result or an event. Events carry no ID.
type cdpResponse struct {
	ID     int             `json:"id"`
	Result json.RawMessage `json:"result"`
	Error  *struct {
		Message string `json:"message"`
	} `json:"error"`
}

// DebugPortArg is the Chrome flag that exposes the DevTools endpoint. Port 0
// lets Chrome choose a free port, which it records in DevToolsActivePort.
const DebugPortArg = "--remote-debugging-port=0"

// dialOrigin is the Origin we present when opening the DevTools websocket.
// Chrome 111+ rejects the upgrade unless the origin is explicitly allowed,
// hence AllowOriginArg; it is scoped to loopback rather than "*".
const dialOrigin = "http://127.0.0.1"

// AllowOriginArg permits exactly the origin dial() sends.
const AllowOriginArg = "--remote-allow-origins=" + dialOrigin

// activePort reads the port Chrome chose for its debugging endpoint.
func activePort(userDataDir string) (int, error) {
	raw, err := os.ReadFile(filepath.Join(userDataDir, devToolsPortFile)) // #nosec G304 -- path derived from gplay's own profile dir
	if err != nil {
		return 0, err
	}
	line, _, _ := strings.Cut(strings.TrimSpace(string(raw)), "\n")
	port, err := strconv.Atoi(strings.TrimSpace(line))
	if err != nil {
		return 0, fmt.Errorf("unreadable %s: %w", devToolsPortFile, err)
	}
	return port, nil
}

// Connect attaches to a Chrome already running against userDataDir, waiting up
// to timeout for its debugging endpoint to accept connections.
func Connect(ctx context.Context, userDataDir string, timeout time.Duration) (*Browser, error) {
	deadline := time.Now().Add(timeout)
	var lastErr error
	for {
		if ctx.Err() != nil {
			return nil, ctx.Err()
		}
		port, err := activePort(userDataDir)
		if err == nil {
			var b *Browser
			if b, lastErr = dial(ctx, port); lastErr == nil {
				return b, nil
			}
		} else {
			lastErr = err
		}
		if time.Now().After(deadline) {
			return nil, fmt.Errorf("no Chrome debugging endpoint for %s: %w", userDataDir, lastErr)
		}
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(200 * time.Millisecond):
		}
	}
}

// dial connects to the first page target on the given debugging port.
func dial(ctx context.Context, port int) (*Browser, error) {
	endpoint := fmt.Sprintf("http://127.0.0.1:%d/json/list", port)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var targets []target
	if err := json.NewDecoder(resp.Body).Decode(&targets); err != nil {
		return nil, fmt.Errorf("decoding target list: %w", err)
	}
	for _, t := range targets {
		if t.Type != "page" || t.WebSocketDebuggerURL == "" {
			continue
		}
		conn, err := websocket.Dial(t.WebSocketDebuggerURL, "", dialOrigin)
		if err != nil {
			return nil, fmt.Errorf("connecting to Chrome target: %w", err)
		}
		return &Browser{conn: conn}, nil
	}
	return nil, fmt.Errorf("no debuggable page target on port %d", port)
}

// Close releases the DevTools connection. It does not quit Chrome: the window
// is the user's, and their sign-in lives in that profile.
func (b *Browser) Close() error {
	if b == nil || b.conn == nil {
		return nil
	}
	return b.conn.Close()
}

// send issues one command and returns its result, skipping protocol events.
func (b *Browser) send(ctx context.Context, method string, params map[string]any) (json.RawMessage, error) {
	b.mu.Lock()
	defer b.mu.Unlock()

	b.next++
	id := b.next
	if deadline, ok := ctx.Deadline(); ok {
		_ = b.conn.SetDeadline(deadline)
	} else {
		_ = b.conn.SetDeadline(time.Now().Add(60 * time.Second))
	}
	if err := websocket.JSON.Send(b.conn, cdpRequest{ID: id, Method: method, Params: params}); err != nil {
		return nil, fmt.Errorf("%s: %w", method, err)
	}
	for {
		var resp cdpResponse
		if err := websocket.JSON.Receive(b.conn, &resp); err != nil {
			return nil, fmt.Errorf("%s: %w", method, err)
		}
		if resp.ID != id {
			continue // an event, or a reply to an earlier command
		}
		if resp.Error != nil {
			return nil, fmt.Errorf("%s: %s", method, resp.Error.Message)
		}
		return resp.Result, nil
	}
}

// Navigate loads a URL in the attached page.
func (b *Browser) Navigate(ctx context.Context, url string) error {
	_, err := b.send(ctx, "Page.navigate", map[string]any{"url": url})
	return err
}

// MouseClick dispatches a trusted mouse click at the given viewport
// coordinates. Some console controls ignore synthetic element.click() calls
// (proven against the live console's bulk-price save); trusted CDP input
// reaches the same handlers as a real user click. A hover precedes the press
// because some controls arm their handlers on pointerover.
func (b *Browser) MouseClick(ctx context.Context, x, y float64) error {
	if _, err := b.send(ctx, "Input.dispatchMouseEvent", map[string]any{
		"type": "mouseMoved", "x": x, "y": y, "button": "none",
	}); err != nil {
		return err
	}
	if _, err := b.send(ctx, "Input.dispatchMouseEvent", map[string]any{
		"type": "mousePressed", "x": x, "y": y, "button": "left", "clickCount": 1,
	}); err != nil {
		return err
	}
	_, err := b.send(ctx, "Input.dispatchMouseEvent", map[string]any{
		"type": "mouseReleased", "x": x, "y": y, "button": "left", "clickCount": 1,
	})
	return err
}

// InsertText types text as a real keyboard would, into the focused element.
// The console is a compiled Dart app whose text fields do not always pick up
// values set via element.value plus synthetic events; trusted text insertion
// is indistinguishable from typing.
func (b *Browser) InsertText(ctx context.Context, text string) error {
	_, err := b.send(ctx, "Input.insertText", map[string]any{"text": text})
	return err
}

// SetFileInputFiles sets files on the first file input matching selector, via
// the DOM domain (works for hidden inputs, which is how the console's upload
// dialogs are built). Files must be absolute paths on this machine.
func (b *Browser) SetFileInputFiles(ctx context.Context, selector string, files []string) error {
	doc, err := b.send(ctx, "DOM.getDocument", map[string]any{"depth": -1})
	if err != nil {
		return fmt.Errorf("DOM.getDocument: %w", err)
	}
	var docResult struct {
		Root struct {
			NodeID int `json:"nodeId"`
		} `json:"root"`
	}
	if err := json.Unmarshal(doc, &docResult); err != nil {
		return fmt.Errorf("DOM.getDocument: %w", err)
	}
	query, err := b.send(ctx, "DOM.querySelector", map[string]any{
		"nodeId":   docResult.Root.NodeID,
		"selector": selector,
	})
	if err != nil {
		return fmt.Errorf("DOM.querySelector %q: %w", selector, err)
	}
	var queryResult struct {
		NodeID int `json:"nodeId"`
	}
	if err := json.Unmarshal(query, &queryResult); err != nil {
		return fmt.Errorf("DOM.querySelector %q: %w", selector, err)
	}
	if queryResult.NodeID == 0 {
		return fmt.Errorf("no file input matches %q", selector)
	}
	if _, err := b.send(ctx, "DOM.setFileInputFiles", map[string]any{
		"files":  files,
		"nodeId": queryResult.NodeID,
	}); err != nil {
		return fmt.Errorf("DOM.setFileInputFiles: %w", err)
	}
	return nil
}

// PressEnter dispatches a trusted Enter key press, into the focused element.
func (b *Browser) PressEnter(ctx context.Context) error {
	for _, typ := range []string{"keyDown", "keyUp"} {
		if _, err := b.send(ctx, "Input.dispatchKeyEvent", map[string]any{
			"type": typ, "key": "Enter", "code": "Enter",
			"windowsVirtualKeyCode": 13, "nativeVirtualKeyCode": 13,
		}); err != nil {
			return err
		}
	}
	return nil
}

// evalResult is the shape of Runtime.evaluate's reply.
type evalResult struct {
	Result struct {
		Value json.RawMessage `json:"value"`
	} `json:"result"`
	ExceptionDetails *struct {
		Text      string `json:"text"`
		Exception *struct {
			Description string `json:"description"`
		} `json:"exception"`
	} `json:"exceptionDetails"`
}

// Eval runs a JavaScript expression in the page and decodes its value into out
// (which may be nil). Promises are awaited.
func (b *Browser) Eval(ctx context.Context, expression string, out any) error {
	raw, err := b.send(ctx, "Runtime.evaluate", map[string]any{
		"expression":    expression,
		"returnByValue": true,
		"awaitPromise":  true,
	})
	if err != nil {
		return err
	}
	var res evalResult
	if err := json.Unmarshal(raw, &res); err != nil {
		return fmt.Errorf("decoding evaluate result: %w", err)
	}
	if res.ExceptionDetails != nil {
		msg := res.ExceptionDetails.Text
		if res.ExceptionDetails.Exception != nil && res.ExceptionDetails.Exception.Description != "" {
			msg = res.ExceptionDetails.Exception.Description
		}
		return fmt.Errorf("page script failed: %s", msg)
	}
	if out == nil || len(res.Result.Value) == 0 {
		return nil
	}
	return json.Unmarshal(res.Result.Value, out)
}

// EvalUntil polls a JavaScript boolean expression until it is true, so callers
// can wait for the console's async UI instead of sleeping blindly.
func (b *Browser) EvalUntil(ctx context.Context, expression string, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	for {
		var ok bool
		err := b.Eval(ctx, expression, &ok)
		if err == nil && ok {
			return nil
		}
		if time.Now().After(deadline) {
			if err != nil {
				return fmt.Errorf("waiting for %q: %w", expression, err)
			}
			return fmt.Errorf("timed out waiting for %q", expression)
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(250 * time.Millisecond):
		}
	}
}

// LaunchArgs returns the Chrome flags needed for a driveable instance.
func LaunchArgs(userDataDir string) []string {
	return append(InteractiveArgs(userDataDir), DebugPortArg, AllowOriginArg)
}

// InteractiveArgs returns the Chrome flags for a window the user drives
// themselves (sign-in). No DevTools port: while that window holds a live
// Google session, an open debug endpoint would let any local process that
// reads DevToolsActivePort drive the session.
func InteractiveArgs(userDataDir string) []string {
	return []string{
		"--user-data-dir=" + userDataDir,
		"--no-first-run",
		"--no-default-browser-check",
	}
}

// Running reports whether a Chrome is already serving DevTools for this
// profile. A second launch against the same user-data-dir just forwards to the
// existing process, so callers must not blindly spawn another.
func Running(userDataDir string) bool {
	port, err := activePort(userDataDir)
	if err != nil {
		return false
	}
	conn, err := net.DialTimeout("tcp", fmt.Sprintf("127.0.0.1:%d", port), time.Second)
	if err != nil {
		return false
	}
	_ = conn.Close()
	return true
}

// BringToFront activates the attached page's tab.
func BringToFront(ctx context.Context, b *Browser) error {
	_, err := b.send(ctx, "Page.bringToFront", nil)
	return err
}

// Reload refreshes the page, bypassing the cache.
func Reload(ctx context.Context, b *Browser) error {
	_, err := b.send(ctx, "Page.reload", map[string]any{"ignoreCache": true})
	return err
}
