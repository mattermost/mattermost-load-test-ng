package coordinator

import (
	"time"
)

// hasPassed returns true if the provided duration has passed since the
// provided time. Returns false otherwise.
func hasPassed(t time.Time, d time.Duration) bool {
	return time.Now().After(t.Add(d))
}

// min finds the minimum between the provided int values.
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
