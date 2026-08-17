package common

import (
	"context"
	"fmt"
	"math"
	"strings"
	"time"
)

const BackoffJitter = float64(0.15)
const MaxBackoffJitter = 500 * time.Millisecond
const BackoffMaxRetriesEpsilon = 1e-9

// TODO: enforce a timeout or log a warning if it's exceeded?
// Some contexts don't have a deadline but instead can just be cancelled after a while
func WithRetries(
	ctx context.Context, logger Logger, fn func() error,
) WrappedError {
	maxObservedRunTime := time.Duration(0)
	retriedFraction := float64(0) // When >= 1, max retries is reached
	retriesByCategory := map[string]int{}
	errs := []error{}
	getPreviousErrorsDebugValue := func() DebugValue {
		message := "no previous errors"
		if len(errs) > 0 {
			var builder strings.Builder
			builder.WriteString("from oldest to newest:")
			for _, prevErr := range errs {
				builder.WriteString("\n")
				builder.WriteString(prevErr.Error())
			}
			message = builder.String()
		}
		return DebugValue{
			Name:    "previous retry errors (WithRetries)",
			Message: message,
			Value:   errs,
		}
	}
	wrapError := func(wrappedErr WrappedError) WrappedError {
		wrappedErr = wrappedErr.CloneAsWrappedError()
		wrappedErr.AddDebugValuesMut(
			getPreviousErrorsDebugValue(),
			DebugValue{
				Name: "retries reset by WithRetries from...",
				Message: fmt.Sprintf(
					"max retries: %v, base backoff: %v, backoff multiplier: %v",
					wrappedErr.MaxRetries(), wrappedErr.RetryBackoffBase(), wrappedErr.RetryBackoffMultiplier(),
				),
			},
		)
		wrappedErr.DisableRetriesMut()
		return wrappedErr
	}

	for {
		startTime := time.Now()
		stdErr := fn()
		if stdErr == nil {
			return nil
		}
		wrappedErr := AutoWrapError(stdErr)
		if wrappedErr.MaxRetries() > 0 {
			retriedFraction += 1 / float64(wrappedErr.MaxRetries()+1)
		}
		if retriedFraction >= 1-BackoffMaxRetriesEpsilon ||
			(wrappedErr.MaxRetries() < 1 && wrappedErr.MaxRetries() != -1) {
			return wrapError(wrappedErr)
		}
		errs = append(errs, stdErr)

		retries := retriesByCategory[wrappedErr.GeneralCategory()]
		backoff := CalculateBackoff(retries, wrappedErr.RetryBackoffBase(), wrappedErr.RetryBackoffMultiplier())
		// Errors with a multiplier of 1 shouldn't increase the backoff for other errors with the same category
		if wrappedErr.RetryBackoffMultiplier() > 1 {
			retriesByCategory[wrappedErr.GeneralCategory()] = retries + 1
		}

		runTime := time.Since(startTime)
		maxObservedRunTime = max(maxObservedRunTime, runTime)
		deadline, hasDeadline := ctx.Deadline()
		if hasDeadline && time.Until(deadline) < maxObservedRunTime+backoff {
			return wrapError(wrappedErr)
		}

		logger.Debug(
			"[WithRetries] retrying temporary error",
			"waitMs", backoff.Milliseconds(),
			"error", wrappedErr,
		)

		select {
		case <-time.After(backoff):
		case <-ctx.Done():
			return WrapErrorWithCategories(
				context.Canceled,
				ErrTypeTimeout, "with retries", ErrTypeCommon,
			).CommonError().AddDebugValue(getPreviousErrorsDebugValue())
		}
	}
}

func CalculateBackoff(retries int, base time.Duration, multiplier float64) time.Duration {
	withoutJitter := float64(base) * math.Pow(multiplier, float64(retries))
	jitter := RandPositiveNegativeRange(min(withoutJitter*BackoffJitter, float64(MaxBackoffJitter)))
	return time.Duration(math.Round(
		withoutJitter + jitter,
	))
}
