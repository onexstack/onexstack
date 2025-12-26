package retry

import (
	"context"
	"fmt"
	"time"

	"k8s.io/apimachinery/pkg/util/wait"
)

const (
	defaultBackoffSteps    = 10
	defaultBackoffFactor   = 1.25
	defaultBackoffDuration = 5 * time.Second
	defaultBackoffJitter   = 1.0
)

// Retry retries a given function with exponential backoff.
// It returns error if all retry attempts are exhausted or the condition never becomes true.
func Retry(fn wait.ConditionFunc, initialBackoffSec int) error {
	backoffDuration := defaultBackoffDuration
	if initialBackoffSec > 0 {
		backoffDuration = time.Duration(initialBackoffSec) * time.Second
	}

	backoffConfig := wait.Backoff{
		Steps:    defaultBackoffSteps,
		Factor:   defaultBackoffFactor,
		Duration: backoffDuration,
		Jitter:   defaultBackoffJitter,
	}

	return wait.ExponentialBackoff(backoffConfig, fn)
}

// Poll tries a condition func until it returns true, an error, or the timeout is reached.
// It waits for the interval duration before the first check.
func Poll(interval, timeout time.Duration, condition wait.ConditionFunc) error {
	return wait.Poll(interval, timeout, condition)
}

// PollImmediate tries a condition func until it returns true, an error, or the timeout is reached.
// It checks the condition immediately before waiting.
func PollImmediate(interval, timeout time.Duration, condition wait.ConditionFunc) error {
	return wait.PollImmediate(interval, timeout, condition)
}

// RunImmediatelyThenPeriod executes the function immediately once (synchronously) and returns error if it fails.
// If successful, it starts an asynchronous loop that executes the function periodically.
// The periodic execution continues until the context is cancelled.
//
// Key behaviors:
//   - First execution is synchronous and blocking
//   - Returns error immediately if first execution fails
//   - Subsequent executions are asynchronous and errors are silently ignored
//   - The period timer starts AFTER the first execution completes
func RunImmediatelyThenPeriod(ctx context.Context, f func(context.Context) error, period time.Duration) error {
	// Synchronously execute once and validate
	if err := f(ctx); err != nil {
		return fmt.Errorf("initial execution failed: %w", err)
	}

	// Start asynchronous periodic execution
	go func() {
		// Wait for one period before the next execution to maintain interval consistency
		timer := time.NewTimer(period)
		defer timer.Stop()

		select {
		case <-timer.C:
			// Timer expired, proceed to periodic loop
		case <-ctx.Done():
			// Context cancelled before first periodic run
			return
		}

		// Enter periodic execution loop
		wait.UntilWithContext(ctx, func(ctx context.Context) {
			// Errors in periodic execution are intentionally ignored
			// Consider adding logging here if needed:
			// if err := f(ctx); err != nil {
			//     klog.Errorf("Periodic execution failed: %v", err)
			// }
			_ = f(ctx)
		}, period)
	}()

	return nil
}
