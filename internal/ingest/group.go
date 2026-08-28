package ingest

import (
	"regexp"
	"sort"
	"strings"
)

// group is a clip-in-waiting: one original plus its optional camera proxy
// candidate and metadata sidecars, all sharing a canonical key.
type group struct {
	stem      string      // the original's base name; the manifest clip stem
	original  foundFile   // the largest media-kind file
	proxyCand *foundFile  // .lrf/.lrv candidate, or nil
	sidecars  []foundFile // .srt/.thm/.xml/.wav
	extras    []foundFile // extra media files beyond the original (rare; kept as sidecars)
}

// pairingRule maps a filename to the canonical group key of its clip. Cameras
// name the proxy differently from the original (GoPro GL↔GX), so grouping can't
// rely on an exact stem match alone (plan §6 step 2).
type pairingRule struct {
	name string
	// key returns the canonical grouping key for base, and whether the rule
	// applies. The first applicable rule wins; identity is the fallback.
	key func(base string) (string, bool)
}

// goProRe matches GoPro's chaptered naming: G{X,H,L,M}{filenum}{clipnum}.
// GX/GH are originals (HEVC/AVC), GL/GM are the paired .LRV low-res proxies.
var goProRe = regexp.MustCompile(`^G([XHLM])(\d{6,})$`)

// pairingRules are tried in order; identity is applied last by canonicalKey.
var pairingRules = []pairingRule{
	{
		name: "gopro",
		key: func(base string) (string, bool) {
			m := goProRe.FindStringSubmatch(base)
			if m == nil {
				return "", false
			}
			// Normalize the proxy variant to its original: L→X, M→H.
			second := m[1]
			switch second {
			case "L":
				second = "X"
			case "M":
				second = "H"
			}
			return "g" + second + m[2], true
		},
	},
}

// canonicalKey returns the grouping key for a base name, applying the first
// matching pairing rule and falling back to the lowercased base (identity,
// which covers DJI where original/proxy/sidecar share an exact stem).
func canonicalKey(base string) string {
	for _, r := range pairingRules {
		if k, ok := r.key(base); ok {
			return k
		}
	}
	return strings.ToLower(base)
}

// groupFiles buckets scanned files into groups and separates out unmatched
// files (sidecars/proxies with no original in their bucket, e.g. an orphan SRT).
// Groups are returned sorted by original filename for deterministic seq order.
func groupFiles(files []foundFile) (groups []group, unmatched []foundFile) {
	buckets := map[string][]foundFile{}
	var order []string
	for _, f := range files {
		k := canonicalKey(f.base)
		if _, seen := buckets[k]; !seen {
			order = append(order, k)
		}
		buckets[k] = append(buckets[k], f)
	}

	for _, k := range order {
		bucket := buckets[k]
		var media []foundFile
		var g group
		for _, f := range bucket {
			switch f.kind {
			case kindMedia:
				media = append(media, f)
			case kindProxy:
				fc := f
				// Keep the largest proxy candidate if several somehow appear.
				if g.proxyCand == nil || fc.size > g.proxyCand.size {
					g.proxyCand = &fc
				}
			case kindSidecar:
				g.sidecars = append(g.sidecars, f)
			}
		}

		if len(media) == 0 {
			// No original: every file here is unmatched (orphan sidecar/proxy).
			if g.proxyCand != nil {
				unmatched = append(unmatched, *g.proxyCand)
			}
			unmatched = append(unmatched, g.sidecars...)
			continue
		}

		// Largest media file is the original; any others are kept as extras.
		sort.Slice(media, func(i, j int) bool { return media[i].size > media[j].size })
		g.original = media[0]
		g.extras = media[1:]
		g.stem = g.original.base
		groups = append(groups, g)
	}

	// Deterministic order: by original filename.
	sort.Slice(groups, func(i, j int) bool { return groups[i].stem < groups[j].stem })
	return groups, unmatched
}
