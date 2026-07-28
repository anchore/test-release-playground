package art

// continents are expressed as outline polygons in (lat, lon) degrees. Each
// polygon is rasterised with 2x2 supersampling so that cells which only
// partially overlap a coastline render with a lighter glyph — giving an
// anti-aliased look at the same grid resolution.
//
// Vertices don't need to follow any winding convention; the point-in-
// polygon test is a standard ray-cast.

type point struct{ lat, lon float64 }

var continents = [][]point{
	// north america + central america
	{
		{70, -160}, {73, -130}, {78, -100}, {73, -78}, {66, -62},
		{55, -60}, {47, -53}, {45, -65}, {41, -70}, {37, -76},
		{32, -80}, {26, -80}, {25, -82}, {30, -85}, {30, -89},
		{29, -94}, {26, -97}, {22, -98}, {19, -96}, {18, -90},
		{17, -88}, {9, -79}, {15, -95}, {19, -104}, {24, -110},
		{32, -117}, {38, -123}, {44, -124}, {48, -124}, {54, -130},
		{59, -140}, {60, -149}, {58, -160}, {64, -167}, {68, -166},
		{70, -160},
	},

	// greenland
	{
		{78, -22}, {82, -32}, {80, -45}, {76, -55}, {68, -53},
		{60, -45}, {62, -41}, {68, -32}, {75, -22}, {78, -22},
	},

	// south america
	{
		{12, -72}, {12, -60}, {5, -52}, {-3, -43}, {-12, -37},
		{-22, -40}, {-26, -48}, {-33, -52}, {-38, -58}, {-44, -65},
		{-50, -69}, {-55, -67}, {-54, -72}, {-45, -74}, {-32, -71},
		{-18, -71}, {-5, -81}, {2, -79}, {8, -77}, {12, -72},
	},

	// africa
	{
		{36, -6}, {37, 10}, {33, 11}, {31, 20}, {32, 30},
		{28, 35}, {15, 40}, {12, 43}, {-1, 42}, {-12, 40},
		{-18, 37}, {-26, 33}, {-35, 22}, {-34, 18}, {-22, 14},
		{-8, 13}, {4, 9}, {6, -8}, {12, -17}, {20, -17},
		{27, -13}, {32, -9}, {36, -6},
	},

	// eurasia (europe + asia, one huge polygon)
	{
		{36, -9}, {44, -9}, {48, -5}, {50, -2}, {53, 5},
		{55, 9}, {58, 10}, {63, 5}, {68, 13}, {71, 26},
		{67, 41}, {68, 55}, {72, 80}, {73, 110}, {72, 140},
		{67, 175}, {60, 170}, {55, 160}, {49, 156}, {46, 142},
		{42, 132}, {40, 130}, {35, 126}, {32, 121}, {25, 118},
		{21, 110}, {12, 109}, {10, 104}, {3, 103}, {1, 104},
		{8, 99}, {16, 98}, {21, 92}, {22, 90}, {12, 80},
		{8, 77}, {23, 68}, {25, 57}, {17, 55}, {12, 44},
		{15, 42}, {30, 33}, {37, 28}, {40, 27}, {41, 20},
		{40, 18}, {44, 12}, {44, 8}, {43, -2}, {38, -9}, {36, -9},
	},

	// british isles
	{
		{59, -3}, {58, -2}, {55, -1}, {53, 1}, {51, 1},
		{50, -4}, {52, -5}, {55, -6}, {55, -10}, {58, -7}, {59, -3},
	},

	// iceland
	{
		{66, -14}, {66, -20}, {64, -23}, {63, -19}, {64, -14}, {66, -14},
	},

	// japan
	{
		{45, 141}, {42, 144}, {37, 141}, {35, 140}, {34, 135},
		{33, 131}, {31, 130}, {34, 132}, {37, 138}, {40, 140}, {45, 141},
	},

	// philippines (sparse)
	{
		{18, 121}, {14, 125}, {10, 126}, {6, 122}, {10, 120},
		{16, 120}, {18, 121},
	},

	// new guinea
	{
		{-1, 131}, {-3, 141}, {-9, 150}, {-10, 140}, {-7, 135},
		{-3, 134}, {-1, 131},
	},

	// borneo
	{
		{7, 109}, {2, 118}, {-3, 116}, {-4, 113}, {-1, 110},
		{4, 109}, {7, 109},
	},

	// sumatra
	{
		{6, 96}, {2, 103}, {-5, 105}, {-6, 102}, {-1, 98},
		{4, 95}, {6, 96},
	},

	// java
	{
		{-6, 106}, {-8, 114}, {-9, 115}, {-8, 108}, {-6, 106},
	},

	// australia
	{
		{-11, 142}, {-17, 146}, {-25, 153}, {-37, 150}, {-38, 141},
		{-34, 137}, {-32, 132}, {-32, 116}, {-22, 114}, {-15, 123},
		{-12, 130}, {-11, 135}, {-11, 142},
	},

	// new zealand (combined as one rough silhouette)
	{
		{-34, 172}, {-37, 177}, {-42, 175}, {-46, 170}, {-46, 167},
		{-41, 172}, {-34, 172},
	},

	// madagascar
	{
		{-12, 49}, {-15, 50}, {-22, 48}, {-25, 45}, {-15, 45}, {-12, 49},
	},
}

func inPolygon(lat, lon float64, poly []point) bool {
	inside := false
	n := len(poly)
	for i, j := 0, n-1; i < n; j, i = i, i+1 {
		if (poly[i].lat > lat) != (poly[j].lat > lat) &&
			lon < (poly[j].lon-poly[i].lon)*(lat-poly[i].lat)/(poly[j].lat-poly[i].lat)+poly[i].lon {
			inside = !inside
		}
	}
	return inside
}

func isLand(lat, lon float64) bool {
	for _, poly := range continents {
		if inPolygon(lat, lon, poly) {
			return true
		}
	}
	return false
}

// densityMask returns, for each cell, how many of its 4 sub-samples fell
// on land. The value is in [0, 4] and lets the renderer pick a glyph that
// suggests anti-aliasing along coastlines.
func densityMask(width, height int) [][]int {
	mask := make([][]int, height)
	// sample offsets within a single cell: a 2x2 subgrid.
	// each row/col index already maps to a (lat, lon); we step by half a
	// cell in both directions to find the 4 sub-sample centres.
	for row := range height {
		mask[row] = make([]int, width)
		for col := range width {
			count := 0
			for _, dr := range []float64{-0.25, 0.25} {
				for _, dc := range []float64{-0.25, 0.25} {
					lat := unprojectLatFloat(float64(row)+dr, height)
					lon := unprojectLonFloat(float64(col)+dc, width)
					if isLand(lat, lon) {
						count++
					}
				}
			}
			mask[row][col] = count
		}
	}
	return mask
}
