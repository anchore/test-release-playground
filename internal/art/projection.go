package art

import "math"

// mercator projection clipped at +75°N / -60°S — enough to show all populated
// continents without the polar singularity. The grid is laid out so:
//
//	col 0..width-1   spans longitude -180..+180
//	row 0..height-1  spans the projected latitude band (top = +75, bottom = -60)
const (
	latTop    = 75.0
	latBottom = -60.0
)

// projectionBounds returns the mercator y values for the top/bottom clips.
func projectionBounds() (float64, float64) {
	return mercatorY(latTop), mercatorY(latBottom)
}

func mercatorY(latDeg float64) float64 {
	// guard against polar extremes — shouldn't trigger since we clip, but be safe.
	if latDeg >= 89.5 {
		latDeg = 89.5
	}
	if latDeg <= -89.5 {
		latDeg = -89.5
	}
	r := latDeg * math.Pi / 180.0
	return math.Log(math.Tan(math.Pi/4 + r/2))
}

// project converts (lat, lon) in degrees to integer (col, row) inside a
// width x height grid using mercator clipped to [latBottom, latTop].
func project(lat, lon float64, width, height int) (int, int) {
	if lat > latTop {
		lat = latTop
	}
	if lat < latBottom {
		lat = latBottom
	}
	for lon > 180 {
		lon -= 360
	}
	for lon < -180 {
		lon += 360
	}
	topY, botY := projectionBounds()
	x := (lon + 180.0) / 360.0 * float64(width-1)
	y := (topY - mercatorY(lat)) / (topY - botY) * float64(height-1)
	return int(math.Round(x)), int(math.Round(y))
}

// unprojectLatFloat inverse-projects a (fractional) row index back to its
// latitude in degrees. Fractional rows are used during 2x2 supersampling
// of continent polygons.
func unprojectLatFloat(row float64, height int) float64 {
	topY, botY := projectionBounds()
	y := topY - row/float64(height-1)*(topY-botY)
	return (2*math.Atan(math.Exp(y)) - math.Pi/2) * 180.0 / math.Pi
}

// unprojectLonFloat is the fractional-column counterpart.
func unprojectLonFloat(col float64, width int) float64 {
	return col/float64(width-1)*360.0 - 180.0
}
