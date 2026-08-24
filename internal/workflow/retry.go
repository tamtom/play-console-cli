package workflow

import (
	"context"
	"errors"
	"fmt"
	"time"
)

const (
	minRetryAttempts  = 2
	maxRetryAttempts  = 100
	maxPolicyDuration = 24 * time.Hour
)

type retryPolicy struct {
	maxAttempts int
	delay       time.Duration
	timeout     time.Duration
	configured  bool
}

func retryPolicyForStep(step Step) (retryPolicy, error) {
	policy := retryPolicy{maxAttempts: 1}
	if step.Retry == nil {
		if step.Timeout == nil {
			return policy, nil
		}
	} else {
		if step.Retry.MaxAttempts < minRetryAttempts || step.Retry.MaxAttempts > maxRetryAttempts {
			return policy, fmt.Errorf("retry.max_attempts must be between %d and %d", minRetryAttempts, maxRetryAttempts)
		}
		delay, err := parseRetryDelay(step.Retry.Delay)
		if err != nil {
			return policy, err
		}
		policy.maxAttempts = step.Retry.MaxAttempts
		policy.delay = delay
		policy.configured = true
	}
	if step.Timeout != nil {
		timeout, err := parsePolicyDuration(*step.Timeout)
		if err != nil {
			return policy, fmt.Errorf("timeout %w", err)
		}
		policy.timeout = timeout
		policy.configured = true
	}
	return policy, nil
}

func parseRetryDelay(value string) (time.Duration, error) {
	delay, err := parsePolicyDuration(value)
	if err != nil {
		return 0, fmt.Errorf("retry.delay %w", err)
	}
	return delay, nil
}

func parsePolicyDuration(value string) (time.Duration, error) {
	delay, err := time.ParseDuration(value)
	if err != nil || delay <= 0 || delay > maxPolicyDuration {
		return 0, fmt.Errorf("must be a positive duration no greater than %s", maxPolicyDuration)
	}
	return delay, nil
}

func classifyAttemptFailure(parent context.Context, attemptErr, runErr error, timeout time.Duration) (string, string) {
	if parent.Err() != nil {
		return "canceled", "step canceled: " + parent.Err().Error()
	}
	if errors.Is(attemptErr, context.DeadlineExceeded) {
		return "timeout", "step timed out after " + timeout.String()
	}
	return "command_failed", runErr.Error()
}

func attemptStatus(failureReason string) string {
	switch failureReason {
	case "timeout", "canceled":
		return failureReason
	default:
		return "error"
	}
}

func nextAttemptInvocation(attempts []AttemptResult) int {
	invocation := 1
	for _, attempt := range attempts {
		if attempt.Invocation >= invocation {
			invocation = attempt.Invocation + 1
		}
	}
	return invocation
}

func commandExitCode(err error) int {
	if err == nil {
		return 0
	}
	type exitCoder interface {
		ExitCode() int
	}
	if exitErr, ok := err.(exitCoder); ok {
		return exitErr.ExitCode()
	}
	return 1
}

func waitForRetry(ctx context.Context, delay time.Duration) error {
	timer := time.NewTimer(delay)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}
