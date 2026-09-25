package playclient

import (
	"context"
	"fmt"
	"io"
	"math/rand/v2"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/tamtom/play-console-cli/internal/config"
)

const (
	defaultMaxRetries = 3
	defaultRetryDelay = time.Second
	maximumRetryDelay = 30 * time.Second
)

type retryTransport struct {
	base       http.RoundTripper
	maxRetries int
	baseDelay  time.Duration
}

// ApplyRetryPolicy installs the repository's bounded, read-only retry policy
// on a Google API HTTP client. Sibling official API clients use this function
// so GPLAY_MAX_RETRIES and GPLAY_RETRY_DELAY behave consistently.
func ApplyRetryPolicy(client *http.Client, cfg *config.Config) error {
	maxRetries, retryDelay, err := retrySettings(cfg)
	if err != nil {
		return fmt.Errorf("resolve retry settings: %w", err)
	}
	client.Transport = &retryTransport{
		base:       client.Transport,
		maxRetries: maxRetries,
		baseDelay:  retryDelay,
	}
	return nil
}

func (t *retryTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	base := t.base
	if base == nil {
		base = http.DefaultTransport
	}
	if t.maxRetries <= 0 || !isRetryableMethod(req.Method) || (req.Body != nil && req.GetBody == nil) {
		return base.RoundTrip(req)
	}

	for attempt := 0; ; attempt++ {
		request := req
		if attempt > 0 && req.GetBody != nil {
			body, err := req.GetBody()
			if err != nil {
				return nil, fmt.Errorf("recreate request body for retry: %w", err)
			}
			request = req.Clone(req.Context())
			request.Body = body
		}

		resp, err := base.RoundTrip(request)
		if attempt >= t.maxRetries || !isTransientResponse(resp, err) {
			return resp, err
		}
		if resp != nil && resp.Body != nil {
			_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, 64<<10))
			_ = resp.Body.Close()
		}

		delay := t.retryDelay(attempt, resp)
		if err := sleepWithContext(req.Context(), delay); err != nil {
			return nil, err
		}
	}
}

func isRetryableMethod(method string) bool {
	switch method {
	case http.MethodGet, http.MethodHead, http.MethodOptions:
		return true
	default:
		return false
	}
}

func isTransientResponse(resp *http.Response, err error) bool {
	if err != nil {
		return true
	}
	if resp == nil {
		return false
	}
	switch resp.StatusCode {
	case http.StatusRequestTimeout,
		http.StatusTooManyRequests,
		http.StatusInternalServerError,
		http.StatusBadGateway,
		http.StatusServiceUnavailable,
		http.StatusGatewayTimeout:
		return true
	default:
		return false
	}
}

func (t *retryTransport) retryDelay(attempt int, resp *http.Response) time.Duration {
	if resp != nil {
		if delay, ok := parseRetryAfter(resp.Header.Get("Retry-After"), time.Now()); ok {
			return minDuration(delay, maximumRetryDelay)
		}
	}

	delay := t.baseDelay
	if delay <= 0 {
		delay = defaultRetryDelay
	}
	for i := 0; i < attempt && delay < maximumRetryDelay; i++ {
		delay = minDuration(delay*2, maximumRetryDelay)
	}
	// Apply 80-120% jitter so concurrent clients do not retry in lockstep.
	spread := delay * 4 / 10
	if spread <= 0 {
		return delay
	}
	return minDuration(delay*8/10+time.Duration(rand.Int64N(int64(spread))), maximumRetryDelay)
}

func parseRetryAfter(value string, now time.Time) (time.Duration, bool) {
	value = strings.TrimSpace(value)
	if value == "" {
		return 0, false
	}
	if seconds, err := strconv.Atoi(value); err == nil && seconds >= 0 {
		return time.Duration(seconds) * time.Second, true
	}
	when, err := http.ParseTime(value)
	if err != nil || !when.After(now) {
		return 0, false
	}
	return when.Sub(now), true
}

func sleepWithContext(ctx context.Context, delay time.Duration) error {
	timer := time.NewTimer(delay)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}

func retrySettings(cfg *config.Config) (int, time.Duration, error) {
	maxRetries := defaultMaxRetries
	retryDelay := defaultRetryDelay
	if cfg != nil {
		if cfg.MaxRetriesConfigured() {
			maxRetries = cfg.MaxRetries
		}
		if raw := strings.TrimSpace(cfg.RetryDelay); raw != "" {
			parsed, err := time.ParseDuration(raw)
			if err != nil || parsed <= 0 {
				return 0, 0, fmt.Errorf("invalid retry_delay %q", raw)
			}
			retryDelay = parsed
		}
	}

	if raw := strings.TrimSpace(os.Getenv("GPLAY_MAX_RETRIES")); raw != "" {
		parsed, err := strconv.Atoi(raw)
		if err != nil || parsed < 0 || parsed > 30 {
			return 0, 0, fmt.Errorf("GPLAY_MAX_RETRIES must be between 0 and 30, got %q", raw)
		}
		maxRetries = parsed
	}
	if raw := strings.TrimSpace(os.Getenv("GPLAY_RETRY_DELAY")); raw != "" {
		parsed, err := time.ParseDuration(raw)
		if err != nil || parsed <= 0 {
			return 0, 0, fmt.Errorf("GPLAY_RETRY_DELAY must be a positive duration, got %q", raw)
		}
		retryDelay = parsed
	}
	return maxRetries, retryDelay, nil
}

func minDuration(a, b time.Duration) time.Duration {
	if a < b {
		return a
	}
	return b
}
