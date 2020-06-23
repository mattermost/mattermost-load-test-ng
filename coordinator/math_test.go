package coordinator

import (
	"math"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestSlope(t *testing.T) {
	var samples []point
	require.True(t, math.IsNaN(slope(samples)))

	samples = []point{
		{time.Unix(0, 0), 0},
	}
	require.True(t, math.IsNaN(slope(samples)))

	samples = []point{
		{time.Unix(0, 0), 0},
		{time.Unix(1, 0), 4},
	}
	require.Equal(t, float64(4), slope(samples))

	samples = []point{
		{time.Unix(0, 0), 0},
		{time.Unix(1, 0), 4},
		{time.Unix(2, 0), 8},
	}
	require.Equal(t, float64(4), slope(samples))

	samples = []point{
		{time.Unix(0, 0), 10},
		{time.Unix(1, 0), 8},
		{time.Unix(2, 0), 6},
		{time.Unix(3, 0), 4},
		{time.Unix(4, 0), 2},
	}
	require.Equal(t, float64(-2), slope(samples))

	samples = []point{
		{time.Unix(0, 0), 8},
		{time.Unix(2, 0), 4},
		{time.Unix(8, 0), 8},
		{time.Unix(10, 0), 4},
		{time.Unix(20, 0), 8},
	}
	s := slope(samples)
	require.Equal(t, float64(0.065), math.Round(s*1000)/1000)

	samples = []point{
		{time.Unix(0, 0), 0},
		{time.Unix(1, 0), 4},
		{time.Unix(2, 0), 8},
		{time.Unix(3, 0), 12},
		{time.Unix(4, 0), 16},
		{time.Unix(5, 0), 12},
		{time.Unix(6, 0), 8},
		{time.Unix(7, 0), 12},
		{time.Unix(8, 0), 16},
		{time.Unix(9, 0), 12},
	}
	s = slope(samples)
	require.Equal(t, float64(1.1879), math.Round(s*10000)/10000)
}

var slopeSink float64

func BenchmarkSlope(b *testing.B) {
	samples := make([]point, 10000)
	for i := 0; i < len(samples); i++ {
		samples[i].x = time.Unix(int64(i), 0)
		if i > 0 {
			samples[i].y = samples[i-1].y + 4
		}
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		slopeSink = slope(samples)
	}
}
