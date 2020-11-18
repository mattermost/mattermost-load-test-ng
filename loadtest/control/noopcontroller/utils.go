package noopcontroller

import (
	"math"
	"math/rand"
	"time"
)

func pickIdleTimeMs(minIdleTimeMs, avgIdleTimeMs int, rate float64) time.Duration {
	// Randomly selecting a value in the interval
	// [minIdleTimeMs, avgIdleTimeMs*2 - minIdleTimeMs).
	// This will give us an expected value equal to avgIdleTimeMs.
	// TODO: consider if it makes more sense to select this value using
	// a truncated normal distribution.
	idleMs := rand.Intn(avgIdleTimeMs*2-minIdleTimeMs*2) + minIdleTimeMs
	idleTimeMs := time.Duration(math.Round(float64(idleMs) * rate))

	return idleTimeMs * time.Millisecond
}
