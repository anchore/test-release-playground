// Package art renders a simulated traceroute as a colored ASCII world map
// with a sidebar of hop details. Everything is deterministic given a seed
// so output is reproducible from the command line.
package art

import (
	"fmt"
	"io"
	"math/rand/v2"
	"strings"
	"time"
)

// Options configures a render.
type Options struct {
	NoColor     bool
	Seed        int64
	Destination string // user-supplied destination hint; empty = random
	Width       int    // map width in cells; 0 → default
}

const (
	defaultWidth = 104
	mapHeight    = 32
)

// glyphs used in the rendered map. landGlyphs is indexed by sub-sample
// density (0..4) so coastlines get a lighter shade than continental
// interiors, giving an anti-aliased look at the same grid resolution.
var landGlyphs = [5]string{" ", "░", "░", "▒", "▓"}

const (
	glyphWater  = " "
	glyphPath   = "•"
	glyphHop    = "●"
	glyphOrigin = "◉"
	glyphTarget = "◆"
)

// landColor is the 256-colour index used to paint continent cells. Bright
// enough to read against any terminal background but muted enough to let
// the latency-coloured route pop on top of it.
const landColor = 244

// Render runs a single simulated traceroute and writes the full scene to w.
func Render(w io.Writer, opts Options) error {
	seed := opts.Seed
	if seed == 0 {
		seed = time.Now().UnixNano()
	}
	rng := rand.New(rand.NewPCG(uint64(seed), uint64(seed)^0x9e3779b97f4a7c15))

	width := opts.Width
	if width <= 0 {
		width = defaultWidth
	}
	// the hop table and the legend strip each need ~96 columns of inner
	// space to render without wrapping. Anything narrower turns the
	// picture into a wreck, so clamp upward and accept the user-supplied
	// width only when it can hold the whole frame.
	if width < 100 {
		width = 100
	}

	pal := newPalette(!opts.NoColor)
	host, dest := pickDestination(rng, opts.Destination)
	r := generateRoute(rng, host, dest)

	canvas := newCanvas(width, mapHeight, pal)
	canvas.drawLand()
	canvas.drawRoute(r)

	var b strings.Builder
	writeFrame(&b, canvas, r, pal, seed)
	_, err := io.WriteString(w, b.String())
	return err
}

// cell carries one rendered grid square: a glyph and its 256-colour index.
// color == -1 means "no colour wrapping" (will use the palette default).
type cell struct {
	glyph string
	color int
}

type canvas struct {
	w, h int
	grid [][]cell
	pal  palette
}

func newCanvas(width, height int, pal palette) *canvas {
	g := make([][]cell, height)
	for row := range height {
		g[row] = make([]cell, width)
		for col := range width {
			g[row][col] = cell{glyph: glyphWater, color: -1}
		}
	}
	return &canvas{w: width, h: height, grid: g, pal: pal}
}

func (c *canvas) drawLand() {
	mask := densityMask(c.w, c.h)
	for row := range c.h {
		for col := range c.w {
			density := mask[row][col]
			if density == 0 {
				continue
			}
			c.grid[row][col] = cell{glyph: landGlyphs[density], color: landColor}
		}
	}
}

// drawRoute paints the path between consecutive hops, then stamps the hops
// themselves on top so markers are never obscured by their own lines.
func (c *canvas) drawRoute(r route) {
	// derive a (col,row) per public hop, plus a virtual point for the origin.
	type plotted struct {
		col, row int
		color    int
		private  bool
	}
	points := []plotted{{}} // placeholder filled below
	originCol, originRow := project(r.origin.lat, r.origin.lon, c.w, c.h)
	points[0] = plotted{col: originCol, row: originRow, color: latencyColor(0), private: false}

	for _, h := range r.hops {
		if h.private {
			continue
		}
		col, row := project(h.loc.lat, h.loc.lon, c.w, c.h)
		points = append(points, plotted{col: col, row: row, color: latencyColor(h.rttMs)})
	}

	// draw path segments between consecutive public points.
	for i := 0; i+1 < len(points); i++ {
		a := points[i]
		b := points[i+1]
		segColor := b.color // colour the segment by the destination hop's latency
		c.drawLine(a.col, a.row, b.col, b.row, glyphPath, segColor)
	}

	// stamp the hop markers (last one wins, so endpoints overwrite path glyphs).
	for i, p := range points {
		glyph := glyphHop
		switch {
		case i == 0:
			glyph = glyphOrigin
		case i == len(points)-1:
			glyph = glyphTarget
		}
		c.put(p.col, p.row, glyph, p.color)
	}
}

func (c *canvas) put(col, row int, glyph string, color int) {
	if col < 0 || col >= c.w || row < 0 || row >= c.h {
		return
	}
	c.grid[row][col] = cell{glyph: glyph, color: color}
}

// drawLine is a Bresenham line walk that won't overwrite existing hop glyphs.
func (c *canvas) drawLine(x0, y0, x1, y1 int, glyph string, color int) {
	dx := abs(x1 - x0)
	dy := -abs(y1 - y0)
	sx, sy := step(x0, x1), step(y0, y1)
	err := dx + dy
	x, y := x0, y0
	for {
		if (x != x0 || y != y0) && (x != x1 || y != y1) {
			// skip if a hop marker is already here (preserve markers).
			if c.cellGlyph(x, y) != glyphHop && c.cellGlyph(x, y) != glyphOrigin && c.cellGlyph(x, y) != glyphTarget {
				c.put(x, y, glyph, color)
			}
		}
		if x == x1 && y == y1 {
			return
		}
		e2 := 2 * err
		if e2 >= dy {
			err += dy
			x += sx
		}
		if e2 <= dx {
			err += dx
			y += sy
		}
	}
}

func (c *canvas) cellGlyph(col, row int) string {
	if col < 0 || col >= c.w || row < 0 || row >= c.h {
		return ""
	}
	return c.grid[row][col].glyph
}

func abs(n int) int {
	if n < 0 {
		return -n
	}
	return n
}

func step(a, b int) int {
	switch {
	case a < b:
		return 1
	case a > b:
		return -1
	default:
		return 0
	}
}

// table column widths. Lines:
//
//	" M  ##  RTT      LOCATION                    ADDRESS             HOSTNAME"
//	 1  1 2  2 2     7  2  25                    2  18               2  rest
const (
	colIdxW  = 2
	colRTTW  = 7
	colLocW  = 25
	colAddrW = 18
)

// writeFrame paints the title bar, the map, a divider, the hop table, and
// the latency legend — all inside one box.
func writeFrame(b *strings.Builder, c *canvas, r route, pal palette, seed int64) {
	inner := c.w
	border := strings.Repeat("─", inner+2)

	title := fmt.Sprintf(" TRACEART :: %s → %s ", labelCity(r.origin), r.targetHost)
	dashes := max(0, inner+2-visibleLen(title))
	titleLine := title + strings.Repeat("─", dashes)
	fmt.Fprintf(b, "┌%s┐\n", pal.bold(titleLine))

	// map body
	for row := range c.h {
		b.WriteString("│ ")
		for col := range c.w {
			cell := c.grid[row][col]
			if cell.color < 0 || !pal.enabled {
				b.WriteString(cell.glyph)
			} else {
				fmt.Fprintf(b, "%s%s\x1b[0m", pal.fg256(cell.color), cell.glyph)
			}
		}
		b.WriteString(" │\n")
	}

	// divider between map and table
	fmt.Fprintf(b, "├%s┤\n", border)

	// host column eats whatever is left after the fixed columns + separators.
	// fixed prefix is " M  " (4 chars), then each fixed column plus its
	// 2-char gap, then the host column.
	const fixedLead = 4
	hostW := max(12, inner+2-(fixedLead+colIdxW+2+colRTTW+2+colLocW+2+colAddrW+2))

	writeTableRow(b, inner, formatHeader(pal, hostW))
	for _, h := range r.hops {
		writeTableRow(b, inner, formatHopRow(h, pal, hostW))
	}

	// legend + status footer
	fmt.Fprintf(b, "├%s┤\n", border)
	writeTableRow(b, inner, " "+legendLine(pal))
	writeTableRow(b, inner, pal.dim(fmt.Sprintf(" simulated route — seed %d — %d hops, %.0fms total",
		seed, len(r.hops), r.hops[len(r.hops)-1].rttMs)))

	fmt.Fprintf(b, "└%s┘\n", border)
}

func writeTableRow(b *strings.Builder, inner int, content string) {
	visible := visibleLen(content)
	pad := max(0, inner+2-visible)
	fmt.Fprintf(b, "│%s%s│\n", content, strings.Repeat(" ", pad))
}

func formatHeader(pal palette, hostW int) string {
	// 3-char prefix ("   ") lines up with " M " in the row format below.
	cols := "    " + padRight("#", colIdxW) +
		"  " + padLeft("RTT", colRTTW) +
		"  " + padRight("LOCATION", colLocW) +
		"  " + padRight("ADDRESS", colAddrW) +
		"  " + padRight("HOSTNAME", hostW)
	return pal.dim(cols)
}

func formatHopRow(h hop, pal palette, hostW int) string {
	var (
		loc         string
		markerGlyph string
		markerColor int
	)
	if h.private {
		loc = "— private —"
		markerGlyph = "·"
		markerColor = 240
	} else {
		loc = labelCity(h.loc)
		markerGlyph = "●"
		markerColor = latencyColor(h.rttMs)
	}

	marker := markerGlyph
	if pal.enabled {
		marker = pal.fg256(markerColor) + markerGlyph + "\x1b[0m"
	}

	// %5.1fms packs e.g. " 14.7ms" and "126.7ms" into exactly 7 columns.
	rtt := fmt.Sprintf("%5.1fms", h.rttMs)
	addr := fmt.Sprintf("(%s)", h.addr)

	cols := " " + padRight(fmt.Sprintf("%02d", h.idx), colIdxW) +
		"  " + rtt +
		"  " + padRight(truncate(loc, colLocW), colLocW) +
		"  " + padRight(truncate(addr, colAddrW), colAddrW) +
		"  " + padRight(truncate(h.hostname, hostW), hostW)

	if h.private {
		cols = pal.dim(cols)
	}
	return " " + marker + " " + cols
}

func legendLine(pal palette) string {
	var parts []string
	for _, b := range legendBuckets {
		bullet := "●"
		if pal.enabled {
			bullet = pal.fg256(b.color) + "●" + "\x1b[0m"
		}
		parts = append(parts, bullet+" "+b.label)
	}
	return pal.dim("latency  ") + strings.Join(parts, "  ")
}

func labelCity(c city) string {
	if c.country == "" {
		return c.name
	}
	return fmt.Sprintf("%s, %s", c.name, c.country)
}

func truncate(s string, n int) string {
	runes := []rune(s)
	if len(runes) <= n {
		return s
	}
	if n <= 1 {
		return string(runes[:n])
	}
	return string(runes[:n-1]) + "…"
}

// padRight pads s on the right with spaces so its visible width is exactly
// n. Multibyte runes (São Paulo, Brasília) count as one column each, which
// fmt's %-*s gets wrong.
func padRight(s string, n int) string {
	diff := n - visibleLen(s)
	if diff <= 0 {
		return s
	}
	return s + strings.Repeat(" ", diff)
}

// padLeft is the right-aligned counterpart, used for numeric columns where
// the data is right-anchored so the digits line up.
func padLeft(s string, n int) string {
	diff := n - visibleLen(s)
	if diff <= 0 {
		return s
	}
	return strings.Repeat(" ", diff) + s
}

// visibleLen returns the printable column width of s, ignoring SGR escapes.
// We assume every printable rune occupies one cell — the glyphs we use are
// all single-cell on any terminal worth its salt.
func visibleLen(s string) int {
	n := 0
	inEsc := false
	for _, r := range s {
		switch {
		case r == 0x1b:
			inEsc = true
		case inEsc && r == 'm':
			inEsc = false
		case inEsc:
			// still inside the escape
		default:
			n++
		}
	}
	return n
}
