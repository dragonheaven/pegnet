// Copyright (c) of parts are held by the various contributors (see the CLA)
// Licensed under the MIT License. See LICENSE file in the project root for full license information.

package polling

import (
	"math"
	"time"

	"github.com/cenkalti/backoff"
)

// Default values for PollingExponentialBackOff.
const (
	DefaultInitialInterval     = 800 * time.Millisecond
	DefaultRandomizationFactor = 0.5
	DefaultMultiplier          = 1.5
	DefaultMaxInterval         = 3 * time.Second
	DefaultMaxElapsedTime      = 10 * time.Second // max 10 seconds
)

// PollingExponentialBackOff creates an instance of ExponentialBackOff
func PollingExponentialBackOff() *backoff.ExponentialBackOff {
	b := &backoff.ExponentialBackOff{
		InitialInterval:     DefaultInitialInterval,
		RandomizationFactor: DefaultRandomizationFactor,
		Multiplier:          DefaultMultiplier,
		MaxInterval:         DefaultMaxInterval,
		MaxElapsedTime:      DefaultMaxElapsedTime,
		Clock:               backoff.SystemClock,
	}
	b.Reset()
	return b
}

func TruncateTo4(v float64) float64 {
	return float64(int64(v*1e4)) / 1e4
}

func TruncateTo8(v float64) float64 {
	return float64(int64(v*1e8)) / 1e8
}

func RoundTo8(v float64) float64 {
	return math.Round(v*1e8) / 1e8
}
