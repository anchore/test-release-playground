package art

import (
	"fmt"
	"math"
	"math/rand/v2"
	"sort"
	"strings"
)

// city is a known peering / pop / endpoint location.
type city struct {
	name    string // human display
	country string // ISO-ish two letter
	airport string // iata-ish code used to mint backbone hostnames
	lat     float64
	lon     float64
}

// peeringCities is the pool we draw intermediate hops from. They're real
// peering hotspots — picking by great-circle distance to the path produces
// plausible-looking routes without needing any real geo data.
var peeringCities = []city{
	{"Seattle", "US", "sea", 47.6, -122.3},
	{"San Jose", "US", "sjc", 37.3, -121.9},
	{"Los Angeles", "US", "lax", 34.0, -118.2},
	{"Denver", "US", "den", 39.7, -104.9},
	{"Dallas", "US", "dfw", 32.8, -96.8},
	{"Chicago", "US", "ord", 41.9, -87.7},
	{"Atlanta", "US", "atl", 33.7, -84.4},
	{"Miami", "US", "mia", 25.8, -80.2},
	{"Ashburn", "US", "iad", 39.0, -77.5},
	{"New York", "US", "jfk", 40.7, -74.0},
	{"Toronto", "CA", "yyz", 43.7, -79.4},
	{"Reykjavik", "IS", "kef", 64.1, -21.9},
	{"London", "GB", "lhr", 51.5, -0.1},
	{"Amsterdam", "NL", "ams", 52.4, 4.9},
	{"Frankfurt", "DE", "fra", 50.1, 8.7},
	{"Paris", "FR", "cdg", 48.9, 2.4},
	{"Madrid", "ES", "mad", 40.4, -3.7},
	{"Stockholm", "SE", "arn", 59.3, 18.1},
	{"Warsaw", "PL", "waw", 52.2, 21.0},
	{"Milan", "IT", "mxp", 45.5, 9.2},
	{"Istanbul", "TR", "ist", 41.0, 28.9},
	{"Dubai", "AE", "dxb", 25.3, 55.4},
	{"Mumbai", "IN", "bom", 19.1, 72.9},
	{"Singapore", "SG", "sin", 1.3, 103.8},
	{"Hong Kong", "HK", "hkg", 22.3, 114.2},
	{"Seoul", "KR", "icn", 37.5, 126.5},
	{"Tokyo", "JP", "nrt", 35.7, 139.7},
	{"Sydney", "AU", "syd", -33.9, 151.2},
	{"Auckland", "NZ", "akl", -36.8, 174.8},
	{"Johannesburg", "ZA", "jnb", -26.2, 28.0},
	{"Cape Town", "ZA", "cpt", -33.9, 18.4},
	{"Cairo", "EG", "cai", 30.0, 31.2},
	{"São Paulo", "BR", "gru", -23.5, -46.6},
	{"Buenos Aires", "AR", "eze", -34.6, -58.4},
	{"Lima", "PE", "lim", -12.1, -77.0},
	{"Mexico City", "MX", "mex", 19.4, -99.1},
}

// destinations is the menu of plausible targets for the simulated tour.
// One is picked at random when --destination isn't supplied.
var destinations = []struct {
	host string
	city city
}{
	{"github.com", city{"Ashburn", "US", "iad", 39.0, -77.5}},
	{"www.cloudflare.com", city{"San Francisco", "US", "sfo", 37.8, -122.4}},
	{"www.bbc.co.uk", city{"London", "GB", "lhr", 51.5, -0.1}},
	{"www.nhk.or.jp", city{"Tokyo", "JP", "nrt", 35.7, 139.7}},
	{"abc.net.au", city{"Sydney", "AU", "syd", -33.9, 151.2}},
	{"www.bund.de", city{"Berlin", "DE", "ber", 52.5, 13.4}},
	{"www.gob.mx", city{"Mexico City", "MX", "mex", 19.4, -99.1}},
	{"www.gov.za", city{"Pretoria", "ZA", "prt", -25.7, 28.2}},
	{"news.ycombinator.com", city{"San Francisco", "US", "sfo", 37.8, -122.4}},
	{"www.amazon.in", city{"Mumbai", "IN", "bom", 19.1, 72.9}},
	{"www.gov.br", city{"Brasília", "BR", "bsb", -15.8, -47.9}},
	{"www.reddit.com", city{"San Francisco", "US", "sfo", 37.8, -122.4}},
}

// hop is one step of the simulated route.
type hop struct {
	idx      int
	hostname string
	addr     string
	loc      city
	rttMs    float64
	private  bool // private/internal hop with no public geo (rendered specially)
}

// route is the full ordered sequence including origin and destination.
type route struct {
	origin      city
	destination city
	targetHost  string
	hops        []hop
}

// great-circle distance in km using the haversine formula.
func haversineKm(a, b city) float64 {
	const earthKm = 6371.0
	rad := math.Pi / 180.0
	dLat := (b.lat - a.lat) * rad
	dLon := (b.lon - a.lon) * rad
	la1 := a.lat * rad
	la2 := b.lat * rad
	h := math.Sin(dLat/2)*math.Sin(dLat/2) +
		math.Cos(la1)*math.Cos(la2)*math.Sin(dLon/2)*math.Sin(dLon/2)
	return 2 * earthKm * math.Asin(math.Sqrt(h))
}

// generateRoute simulates an entire traceroute end-to-end. The result is
// deterministic for a given rng so --seed reproduces the same tour.
func generateRoute(rng *rand.Rand, targetHost string, dest city) route {
	origin := pickOrigin(rng, dest)

	// pick intermediate peering cities biased toward the great-circle path.
	// aim for a long, interesting tour — 8-12 backbone hops on top of the
	// private+destination hops gives a full traceroute-y feel.
	intermediates := selectAlongPath(origin, dest, rng, 8+rng.IntN(5))

	r := route{
		origin:      origin,
		destination: dest,
		targetHost:  targetHost,
	}

	// total round-trip rises monotonically with distance — give the final
	// hop a believable round-trip ceiling (roughly speed-of-light in fibre
	// plus assorted middlebox overhead).
	distKm := haversineKm(origin, dest)
	finalRTT := 4.0 + distKm/110.0 + 8.0*rng.Float64() // ~9.1ms per 1000km plus jitter

	// build the hop list. First three are local / ISP private hops, then
	// public backbone hops on intermediate cities, then destination edge.
	addPrivate := func(name, addr string, rtt float64) {
		r.hops = append(r.hops, hop{
			idx:      len(r.hops) + 1,
			hostname: name,
			addr:     addr,
			loc:      origin,
			rttMs:    rtt,
			private:  true,
		})
	}
	addPublic := func(loc city, rtt float64) {
		r.hops = append(r.hops, hop{
			idx:      len(r.hops) + 1,
			hostname: backboneHostname(loc, rng),
			addr:     fakeAddr(rng),
			loc:      loc,
			rttMs:    rtt,
		})
	}

	addPrivate("_gateway", "192.168.1.1", 0.4+rng.Float64()*0.6)
	addPrivate("cpe."+pickWord(rng, ispWords)+".net", "10.0.0.1", 1.5+rng.Float64()*2.0)
	addPrivate(pickWord(rng, ispWords)+"-gw.isp.net", randPubIPv4(rng), 4.0+rng.Float64()*4.0)

	// interpolate RTTs across the public backbone hops up to ~80% of final,
	// then the last public hop lands at finalRTT.
	publicHops := append([]city{}, intermediates...)
	publicHops = append(publicHops, dest)
	startRTT := r.hops[len(r.hops)-1].rttMs + 3.0
	for i, loc := range publicHops {
		// quadratic ease toward the destination latency so things bunch up
		// as the packet crosses the long-haul leg.
		t := float64(i+1) / float64(len(publicHops))
		eased := t * t
		rtt := startRTT + (finalRTT-startRTT)*eased
		rtt += (rng.Float64() - 0.4) * 6.0 // small jitter, slightly biased down
		if rtt < startRTT {
			rtt = startRTT
		}
		addPublic(loc, rtt)
	}

	// rewrite the very last hop's hostname to look like the destination's edge.
	last := &r.hops[len(r.hops)-1]
	last.hostname = targetHost

	return r
}

func pickOrigin(rng *rand.Rand, dest city) city {
	// home is always one of a handful of plausible "starting" cities so the
	// route always begins somewhere recognisable. To keep the tour
	// visually interesting, throw out candidates that sit too close to
	// the destination — at least 4000 km of journey to draw.
	starts := []city{
		{"Brooklyn", "US", "jfk", 40.65, -73.95},
		{"Portland", "US", "pdx", 45.5, -122.7},
		{"Austin", "US", "aus", 30.3, -97.7},
		{"Boulder", "US", "den", 40.0, -105.3},
		{"Berlin", "DE", "ber", 52.5, 13.4},
		{"Bristol", "GB", "brs", 51.5, -2.6},
		{"Melbourne", "AU", "mel", -37.8, 145.0},
		{"Bengaluru", "IN", "blr", 12.97, 77.6},
		{"Cape Town", "ZA", "cpt", -33.9, 18.4},
		{"Buenos Aires", "AR", "eze", -34.6, -58.4},
		{"Vancouver", "CA", "yvr", 49.3, -123.1},
		{"Helsinki", "FI", "hel", 60.2, 24.9},
	}
	far := starts[:0]
	for _, s := range starts {
		if haversineKm(s, dest) >= 4000 {
			far = append(far, s)
		}
	}
	if len(far) == 0 {
		far = starts
	}
	return far[rng.IntN(len(far))]
}

// scoredCity carries a peering city annotated with its position along a
// candidate path.
type scoredCity struct {
	c        city
	progress float64 // 0..1 along the path
	offset   float64 // how far off-path in km (triangle excess)
}

func scoreCandidates(origin, dest city, totalDist float64) []scoredCity {
	var out []scoredCity
	for _, c := range peeringCities {
		if sameCity(c, origin) || sameCity(c, dest) {
			continue
		}
		dOrigin := haversineKm(origin, c)
		dDest := haversineKm(c, dest)
		progress := dOrigin / (dOrigin + dDest)
		if progress < 0.05 || progress > 0.95 {
			continue
		}
		out = append(out, scoredCity{c: c, progress: progress, offset: dOrigin + dDest - totalDist})
	}
	return out
}

func filterByOffset(candidates []scoredCity, totalDist, frac float64) []scoredCity {
	maxOffset := totalDist * frac
	out := make([]scoredCity, 0, len(candidates))
	for _, s := range candidates {
		if s.offset <= maxOffset {
			out = append(out, s)
		}
	}
	return out
}

// selectAlongPath chooses up to `n` distinct peering cities ordered by how
// far along the origin→dest great-circle they sit. Bias is toward cities
// close to the path so we don't generate teleport-y routes.
func selectAlongPath(origin, dest city, rng *rand.Rand, n int) []city {
	totalDist := haversineKm(origin, dest)
	if totalDist < 1 {
		totalDist = 1
	}
	scored := scoreCandidates(origin, dest, totalDist)

	// start tight (12% of path length) and loosen progressively if we
	// don't have enough candidates for the requested hop count — a long,
	// slightly scenic detour beats a stubbed two-hop route.
	candidates := filterByOffset(scored, totalDist, 0.12)
	for relax := 0.18; len(candidates) < n && relax <= 0.40; relax += 0.07 {
		candidates = filterByOffset(scored, totalDist, relax)
	}
	sort.Slice(candidates, func(i, j int) bool { return candidates[i].progress < candidates[j].progress })
	if len(candidates) == 0 {
		return nil
	}
	n = min(n, len(candidates))
	// stratified pick: divide progress space into n buckets and choose one
	// city from each, so the resulting hop chain moves monotonically toward
	// the destination instead of zigzagging.
	out := make([]city, 0, n)
	for i := range n {
		lo := i * len(candidates) / n
		hi := (i + 1) * len(candidates) / n
		if hi <= lo {
			hi = lo + 1
		}
		if hi > len(candidates) {
			hi = len(candidates)
		}
		pick := candidates[lo+rng.IntN(hi-lo)]
		out = append(out, pick.c)
	}
	return out
}

func sameCity(a, b city) bool { return a.name == b.name && a.country == b.country }

var ispWords = []string{"comcast", "verizon", "spectrum", "vodafone", "ntt", "telia", "level3", "telefonica"}
var backboneVendors = []string{"cogentco.com", "ntt.net", "telia.net", "level3.net", "he.net", "zayo.net", "gtt.net"}

func pickWord(rng *rand.Rand, words []string) string {
	return words[rng.IntN(len(words))]
}

func backboneHostname(c city, rng *rand.Rand) string {
	vendor := backboneVendors[rng.IntN(len(backboneVendors))]
	return fmt.Sprintf("ae-%d.r%02d.%s%02d.%s",
		1+rng.IntN(48),
		rng.IntN(40),
		c.airport,
		1+rng.IntN(9),
		vendor,
	)
}

func fakeAddr(rng *rand.Rand) string {
	// pull from a few realistic /8s assigned to common transit networks so
	// the addresses don't look obviously fake.
	prefixes := []string{"4", "38", "62", "129", "154", "173", "184", "208"}
	return fmt.Sprintf("%s.%d.%d.%d",
		prefixes[rng.IntN(len(prefixes))],
		rng.IntN(256), rng.IntN(256), 1+rng.IntN(254))
}

func randPubIPv4(rng *rand.Rand) string {
	return fmt.Sprintf("96.%d.%d.%d", rng.IntN(256), rng.IntN(256), 1+rng.IntN(254))
}

// pickDestination chooses a destination, either by user input matching one
// of the known hosts (substring, case-insensitive) or at random.
func pickDestination(rng *rand.Rand, requested string) (string, city) {
	if requested != "" {
		needle := strings.ToLower(requested)
		// match against host, city name, country, and airport code so
		// `--destination tokyo` or `--destination jp` both work.
		for _, d := range destinations {
			if strings.Contains(strings.ToLower(d.host), needle) ||
				strings.Contains(strings.ToLower(d.city.name), needle) ||
				strings.EqualFold(d.city.country, needle) ||
				strings.EqualFold(d.city.airport, needle) {
				return d.host, d.city
			}
		}
		// also try the broader peering-city pool so any well-known city works.
		for _, c := range peeringCities {
			if strings.Contains(strings.ToLower(c.name), needle) ||
				strings.EqualFold(c.country, needle) ||
				strings.EqualFold(c.airport, needle) {
				return requested, c
			}
		}
		// nothing matched — fall back to a random target so the user still
		// sees their hostname in the tour rather than getting an error.
		c := peeringCities[rng.IntN(len(peeringCities))]
		return requested, c
	}
	d := destinations[rng.IntN(len(destinations))]
	return d.host, d.city
}
