package art

import "fmt"

// palette is the bundle of ansi escapes used everywhere. When colour is
// disabled every helper collapses to the identity transform.
type palette struct {
	enabled bool
}

func newPalette(enabled bool) palette { return palette{enabled: enabled} }

func (p palette) wrap(code, s string) string {
	if !p.enabled {
		return s
	}
	return code + s + "\x1b[0m"
}

// fg256 returns an ansi escape for a 256-colour foreground.
func (p palette) fg256(idx int) string {
	if !p.enabled {
		return ""
	}
	return fmt.Sprintf("\x1b[38;5;%dm", idx)
}

func (p palette) dim(s string) string  { return p.wrap("\x1b[2m", s) }
func (p palette) bold(s string) string { return p.wrap("\x1b[1m", s) }

// latencyColor maps a round-trip-time in milliseconds onto a 256-colour
// index. The gradient runs green → lime → yellow → orange → red so a
// reader can read overall route health from the markers alone.
func latencyColor(rtt float64) int {
	switch {
	case rtt < 10:
		return 46 // bright green
	case rtt < 30:
		return 82 // lime
	case rtt < 60:
		return 118 // yellow-green
	case rtt < 100:
		return 226 // yellow
	case rtt < 150:
		return 214 // orange
	case rtt < 220:
		return 208 // dark orange
	case rtt < 320:
		return 202 // red-orange
	default:
		return 196 // red
	}
}

// legendBuckets describes the gradient stops used in the legend strip; keep
// this in sync with latencyColor.
var legendBuckets = []struct {
	label string
	color int
}{
	{"<10ms", 46},
	{"<30ms", 82},
	{"<60ms", 118},
	{"<100ms", 226},
	{"<150ms", 214},
	{"<220ms", 208},
	{"<320ms", 202},
	{"320+", 196},
}
