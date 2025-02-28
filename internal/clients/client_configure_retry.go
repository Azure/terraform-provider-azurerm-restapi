package clients

import (
	"context"
	"regexp"
	"time"

	"github.com/Azure/terraform-provider-azapi/internal/retry"
	"github.com/cenkalti/backoff/v4"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

// configureCustomRetry configures the retry configuration based on the supplied retry configuration.
// Using a dedicated function to allow for easier testing.
// Data callback funcs have been removed because we no longer use them.
func configureCustomRetry(ctx context.Context, rtry retry.RetryValue, useReadAfterCreateValues bool) (*backoff.ExponentialBackOff, []regexp.Regexp, []int) {
	// Configure default retry configuration.
	// The default is to retry on 429 codes, so using the context deadline as max elapsed time is sane.
	// Add 1 second to the max elapsed time to allow the context deadline to be reached,
	// which is the error message that callers expect.
	maxElapsed := 5 * time.Minute
	if ctxDeadline, ok := ctx.Deadline(); ok {
		maxElapsed = time.Until(ctxDeadline) + time.Second
	}
	backOff := backoff.NewExponentialBackOff(
		backoff.WithInitialInterval(retry.DefaultIntervalSeconds*time.Second),
		backoff.WithMaxInterval(retry.DefaultMaxIntervalSeconds*time.Second),
		backoff.WithMaxElapsedTime(maxElapsed),
	)
	errRegExps := []regexp.Regexp{}
	statusCodes := rtry.GetDefaultRetryableStatusCodes()
	// Add default read after create values for the default retry configuration.
	if useReadAfterCreateValues {
		statusCodes = rtry.GetDefaultRetryableReadAfterCreateStatusCodes()
	}

	// If a custom retry configuration is supplied, use it.
	if !rtry.IsNull() && !rtry.IsUnknown() {
		tflog.Debug(ctx, "using custom retry configuration")

		backOff = backoff.NewExponentialBackOff(
			backoff.WithInitialInterval(rtry.GetIntervalSecondsAsDuration()),
			backoff.WithMaxInterval(rtry.GetMaxIntervalSecondsAsDuration()),
			backoff.WithMultiplier(rtry.GetMultiplier()),
			backoff.WithRandomizationFactor(rtry.GetRandomizationFactor()),
			backoff.WithMaxElapsedTime(maxElapsed),
		)
		errRegExps = rtry.GetErrorMessagesRegex()
	}

	return backOff, errRegExps, statusCodes
}
