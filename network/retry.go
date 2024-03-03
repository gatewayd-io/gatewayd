package network

import (
	"errors"
	"math"
	"time"

	"github.com/rs/zerolog"
)

const (
	BackoffMultiplierCap = 10
	BackoffDurationCap   = time.Minute
)

type RetryCallback func() (any, error)

type IRetry interface {
	Retry(_ RetryCallback) (any, error)
}

type Retry struct {
	logger             zerolog.Logger
	Retries            int
	Backoff            time.Duration
	BackoffMultiplier  float64
	DisableBackoffCaps bool
}

var _ IRetry = (*Retry)(nil)

// Retry runs the callback function and retries it if it fails.
// It'll wait for the duration of the backoff between retries.
func (r *Retry) Retry(callback RetryCallback) (any, error) {
	var (
		object any
		err    error
		retry  int
	)

	if callback == nil {
		return nil, errors.New("callback is nil")
	}

	// If the number of retries is 0, just run the callback once (first attempt).
	if r == nil {
		return callback()
	}

	// The first attempt counts as a retry.
	for ; retry <= r.Retries; retry++ {
		// Wait for the backoff duration before retrying. The backoff duration is
		// calculated by multiplying the backoff duration by the backoff multiplier
		// raised to the power of the number of retries. For example, if the backoff
		// duration is 1 second and the backoff multiplier is 2, the backoff duration
		// will be 1 second, 2 seconds, 4 seconds, 8 seconds, etc. The backoff duration
		// is capped at 1 minute and the backoff multiplier is capped at 10, so the
		// backoff duration will be 1 minute after 6 retries. The backoff multiplier
		// is capped at 10 to prevent the backoff duration from growing too quickly,
		// unless the backoff caps are disabled.
		// Example: 1 second * 2 ^ 1 = 2 seconds
		//  		1 second * 2 ^ 2 = 4 seconds
		//  		1 second * 2 ^ 3 = 8 seconds
		//  		1 second * 2 ^ 4 = 16 seconds
		//  		1 second * 2 ^ 5 = 32 seconds
		//  		1 second * 2 ^ 6 = 1 minute
		// 			1 second * 2 ^ 7 = 1 minute (capped)
		// 			1 second * 2 ^ 8 = 1 minute (capped)
		// 			1 second * 2 ^ 9 = 1 minute (capped)
		// 			1 second * 2 ^ 10 = 1 minute (capped)
		backoffDuration := r.Backoff * time.Duration(
			math.Pow(r.BackoffMultiplier, float64(retry)),
		)

		if !r.DisableBackoffCaps && backoffDuration > BackoffDurationCap {
			backoffDuration = BackoffDurationCap
		}

		if retry > 0 {
			r.logger.Debug().Fields(
				map[string]interface{}{
					"retry": retry,
					"delay": backoffDuration.String(),
				},
			).Msg("Trying to run callback again")
		} else {
			r.logger.Trace().Msg("First attempt to run callback")
		}

		// Try and retry the callback.
		object, err = callback()
		if err == nil {
			return object, nil
		}

		time.Sleep(backoffDuration)
	}

	r.logger.Error().Err(err).Msgf("Failed to run callback after %d retries", retry)

	return nil, err
}

func NewRetry(
	retries int,
	backoff time.Duration,
	backoffMultiplier float64,
	disableBackoffCaps bool,
	logger zerolog.Logger,
) *Retry {
	retry := Retry{
		Retries:            retries,
		Backoff:            backoff,
		BackoffMultiplier:  backoffMultiplier,
		DisableBackoffCaps: disableBackoffCaps,
		logger:             logger,
	}

	// If the number of retries is less than 0, set it to 0 to disable retries.
	if retry.Retries < 0 {
		retry.Retries = 0
	}

	if !retry.DisableBackoffCaps && retry.BackoffMultiplier > BackoffMultiplierCap {
		retry.BackoffMultiplier = BackoffMultiplierCap
	}

	return &retry
}
