package coordinator

import (
	"time"
)

type point struct {
	x time.Time
	y int
}

// slope calculates the slope of the best fit line using simple linear
// regression (least squares method).
func slope(points []point) float64 {
	n := float64(len(points))
	var sumXY float64
	var sumX float64
	var sumY float64
	var sumXX float64

	for _, p := range points {
		x := float64(p.x.Unix() - points[0].x.Unix())
		y := float64(p.y)
		sumXY += x * y
		sumX += x
		sumY += y
		sumXX += x * x
	}

	return ((n * sumXY) - (sumX * sumY)) / ((n * sumXX) - (sumX * sumX))
}

func avg(points []point) float64 {
	var total int
	for _, p := range points {
		total += p.y
	}
	return float64(total) / float64(len(points))
}
