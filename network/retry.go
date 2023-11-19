package network

import (
	"math"
	"net"
	"time"

	"github.com/rs/zerolog"
)

const (
	BackoffMultiplierCap = 10
	BackoffDurationCap   = time.Minute
)

type IRetry interface {
	DialTimeout(network, address string, timeout time.Duration) (net.Conn, error)
}

type Retry struct {
	logger             zerolog.Logger
	Retries            int
	Backoff            time.Duration
	BackoffMultiplier  float64
	DisableBackoffCaps bool
}

var _ IRetry = (*Retry)(nil)

// DialTimeout dials a connection with a timeout, retrying if it fails.
// It'll wait for the duration of the backoff between retries.
func (r *Retry) DialTimeout(network, address string, timeout time.Duration) (net.Conn, error) {
	var (
		conn  net.Conn
		err   error
		retry int
	)

	if r == nil {
		// Just dial the connection once.
		if timeout == 0 {
			return net.Dial(network, address) //nolint: wrapcheck
		}

		return net.DialTimeout(network, address, timeout) //nolint: wrapcheck
	}

	for ; retry < r.Retries; retry++ {
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
			math.Pow(r.BackoffMultiplier, float64(retry+1)),
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
			).Msg("Trying to connect")
		} else {
			r.logger.Trace().Msg("Trying to connect for the first time")
		}

		// Dial the connection with a timeout if one is provided, otherwise dial the
		// connection without a timeout. Dialing without a timeout will block
		// indefinitely.
		if timeout > 0 {
			conn, err = net.DialTimeout(network, address, timeout)
		} else {
			conn, err = net.Dial(network, address)
		}
		// If the connection was successful, return it.
		if err == nil {
			return conn, nil
		}

		time.Sleep(backoffDuration)
	}

	r.logger.Error().Err(err).Msgf("Failed to connect after %d retries", retry)

	return nil, err //nolint: wrapcheck
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

	if retry.Retries == 0 {
		retry.Retries = 1
	}

	if !retry.DisableBackoffCaps && retry.BackoffMultiplier > BackoffMultiplierCap {
		retry.BackoffMultiplier = BackoffMultiplierCap
	}

	return &retry
}
