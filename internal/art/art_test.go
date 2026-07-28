package art

import (
	"bytes"
	"math"
	"math/rand/v2"
	"strings"
	"testing"
)

func TestProjectRoundTrip(t *testing.T) {
	const w, h = 104, 32

	tests := []struct {
		name     string
		lat, lon float64
	}{
		{"equator greenwich", 0, 0},
		{"london", 51.5, -0.1},
		{"tokyo", 35.7, 139.7},
		{"sydney", -33.9, 151.2},
		{"sao paulo", -23.5, -46.6},
		{"clip north", 80, 10}, // above latTop — clipped to 75
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			col, row := project(tt.lat, tt.lon, w, h)
			if col < 0 || col >= w || row < 0 || row >= h {
				t.Fatalf("projection out of bounds: col=%d row=%d", col, row)
			}
			gotLon := unprojectLonFloat(float64(col), w)
			if math.Abs(gotLon-tt.lon) > 4 {
				// 4 degree tolerance because the grid is coarse
				t.Errorf("lon round-trip drift too large: want %.2f got %.2f", tt.lon, gotLon)
			}
		})
	}
}

func TestDensityMaskHasBothLandAndWater(t *testing.T) {
	mask := densityMask(104, 32)
	var land, water int
	for _, row := range mask {
		for _, d := range row {
			if d > 0 {
				land++
			} else {
				water++
			}
		}
	}
	if land == 0 || water == 0 {
		t.Fatalf("expected both land and water: land=%d water=%d", land, water)
	}
	// sanity: land should occupy somewhere between 15% and 60% of the grid
	total := land + water
	frac := float64(land) / float64(total)
	if frac < 0.15 || frac > 0.60 {
		t.Errorf("land fraction looks off: %.2f", frac)
	}
}

func TestLatencyColorMonotonic(t *testing.T) {
	// not strictly monotonic in colour index, but a low rtt should never
	// produce the "red" buckets and a high one should never produce green.
	if c := latencyColor(1); c != 46 {
		t.Errorf("1ms should be brightest green, got %d", c)
	}
	if c := latencyColor(900); c != 196 {
		t.Errorf("900ms should be deep red, got %d", c)
	}
}

func TestGenerateRouteMonotonicRTT(t *testing.T) {
	rng := rand.New(rand.NewPCG(42, 42^0x9e3779b97f4a7c15))
	host, dest := pickDestination(rng, "tokyo")
	r := generateRoute(rng, host, dest)

	if len(r.hops) < 4 {
		t.Fatalf("expected at least 4 hops, got %d", len(r.hops))
	}
	if r.hops[len(r.hops)-1].hostname != host {
		t.Errorf("last hop hostname = %q, want %q", r.hops[len(r.hops)-1].hostname, host)
	}
	if r.destination.name != dest.name {
		t.Errorf("destination = %q, want %q", r.destination.name, dest.name)
	}

	for i := 1; i < len(r.hops); i++ {
		// allow small jitter dips, but no hop should regress by more than 5ms.
		if r.hops[i].rttMs < r.hops[i-1].rttMs-5 {
			t.Errorf("rtt regressed too much at hop %d: %.1f -> %.1f",
				i, r.hops[i-1].rttMs, r.hops[i].rttMs)
		}
	}
}

func TestRenderDeterministic(t *testing.T) {
	var a, b bytes.Buffer
	opts := Options{Seed: 1234, NoColor: true, Destination: "github.com"}
	if err := Render(&a, opts); err != nil {
		t.Fatal(err)
	}
	if err := Render(&b, opts); err != nil {
		t.Fatal(err)
	}
	if a.String() != b.String() {
		t.Fatal("render output is non-deterministic with the same seed")
	}
	if !strings.Contains(a.String(), "TRACEART") {
		t.Fatal("expected TRACEART title in output")
	}
	if !strings.Contains(a.String(), "github.com") {
		t.Fatal("expected destination host in output")
	}
}

func TestRenderColorEscapes(t *testing.T) {
	var buf bytes.Buffer
	if err := Render(&buf, Options{Seed: 7, NoColor: false}); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(buf.String(), "\x1b[") {
		t.Fatal("expected ANSI escapes when colour is enabled")
	}

	buf.Reset()
	if err := Render(&buf, Options{Seed: 7, NoColor: true}); err != nil {
		t.Fatal(err)
	}
	if strings.Contains(buf.String(), "\x1b[") {
		t.Fatal("did not expect ANSI escapes with --no-color")
	}
}

func TestPickDestinationFallback(t *testing.T) {
	rng := rand.New(rand.NewPCG(1, 1))
	host, _ := pickDestination(rng, "wholly-unknown-host.example")
	if host != "wholly-unknown-host.example" {
		t.Errorf("expected user host preserved on fallback, got %q", host)
	}
}

func TestVisibleLenIgnoresEscapes(t *testing.T) {
	if got := visibleLen("\x1b[31mhello\x1b[0m"); got != 5 {
		t.Errorf("visibleLen ignoring ansi = %d, want 5", got)
	}
	if got := visibleLen("plain"); got != 5 {
		t.Errorf("visibleLen plain = %d, want 5", got)
	}
}
