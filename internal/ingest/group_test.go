package ingest

import (
	"strings"
	"testing"

	"github.com/christophercuongkim/studio/internal/probe"
)

// mk mirrors what scan() produces: base keeps its case, ext is lowercased.
func mk(name string, k kind, size int64) foundFile {
	base, ext := name, ""
	if i := lastDot(name); i >= 0 {
		base, ext = name[:i], strings.ToLower(name[i:])
	}
	return foundFile{path: "/dump/" + name, base: base, ext: ext, kind: k, size: size}
}

func lastDot(s string) int {
	for i := len(s) - 1; i >= 0; i-- {
		if s[i] == '.' {
			return i
		}
	}
	return -1
}

func TestGroupFilesDJIAndGoPro(t *testing.T) {
	files := []foundFile{
		// DJI: original + LRF proxy + SRT sidecar, exact shared stem.
		mk("DJI_0001.MP4", kindMedia, 1000),
		mk("DJI_0001.LRF", kindProxy, 50),
		mk("DJI_0001.SRT", kindSidecar, 5),
		// GoPro: GX original pairs with GL LRV proxy (2nd char differs).
		mk("GX010042.MP4", kindMedia, 2000),
		mk("GL010042.LRV", kindProxy, 80),
		// Orphan sidecar: no original in its bucket.
		mk("random.SRT", kindSidecar, 3),
	}
	groups, unmatched := groupFiles(files)

	if len(groups) != 2 {
		t.Fatalf("got %d groups, want 2: %+v", len(groups), groups)
	}
	byStem := map[string]group{}
	for _, g := range groups {
		byStem[g.stem] = g
	}

	dji, ok := byStem["DJI_0001"]
	if !ok {
		t.Fatal("missing DJI_0001 group")
	}
	if dji.proxyCand == nil || dji.proxyCand.ext != ".lrf" {
		t.Errorf("DJI proxy candidate not the LRF: %+v", dji.proxyCand)
	}
	if len(dji.sidecars) != 1 || dji.sidecars[0].ext != ".srt" {
		t.Errorf("DJI sidecars wrong: %+v", dji.sidecars)
	}

	gopro, ok := byStem["GX010042"]
	if !ok {
		t.Fatal("GoPro GL did not group with GX")
	}
	if gopro.proxyCand == nil || gopro.proxyCand.base != "GL010042" {
		t.Errorf("GoPro proxy candidate wrong: %+v", gopro.proxyCand)
	}

	if len(unmatched) != 1 || unmatched[0].base != "random" {
		t.Errorf("unmatched wrong: %+v", unmatched)
	}
}

func TestGroupFilesLargestIsOriginal(t *testing.T) {
	files := []foundFile{
		mk("clip.MP4", kindMedia, 100),
		mk("clip.MOV", kindMedia, 900), // larger → original
	}
	groups, _ := groupFiles(files)
	if len(groups) != 1 {
		t.Fatalf("want 1 group, got %d", len(groups))
	}
	if groups[0].original.ext != ".mov" {
		t.Errorf("original = %s, want the larger .mov", groups[0].original.ext)
	}
	if len(groups[0].extras) != 1 || groups[0].extras[0].ext != ".mp4" {
		t.Errorf("extras = %+v, want the smaller .mp4", groups[0].extras)
	}
}

func TestProxyAdoptable(t *testing.T) {
	orig := probe.Result{VCodec: "hevc", HasAudio: true, DurationSec: 42.0}
	tests := []struct {
		name string
		cand probe.Result
		want bool
	}{
		{"good", probe.Result{VCodec: "h264", HasAudio: true, DurationSec: 42.4}, true},
		{"not h264", probe.Result{VCodec: "hevc", HasAudio: true, DurationSec: 42.0}, false},
		{"audio mismatch", probe.Result{VCodec: "h264", HasAudio: false, DurationSec: 42.0}, false},
		{"too short", probe.Result{VCodec: "h264", HasAudio: true, DurationSec: 40.0}, false},
		{"edge within tolerance", probe.Result{VCodec: "h264", HasAudio: true, DurationSec: 43.5}, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, reason := proxyAdoptable(tt.cand, orig)
			if got != tt.want {
				t.Errorf("proxyAdoptable = %v (%s), want %v", got, reason, tt.want)
			}
		})
	}
}

func TestDetectCollisions(t *testing.T) {
	pairs := []movePair{
		{src: "/dump/a/clip.mp4", dst: "/proj/originals/clip.mp4"},
		{src: "/dump/b/clip.mp4", dst: "/proj/originals/clip.mp4"}, // clashes with above
		{src: "/dump/unique.mp4", dst: "/proj/originals/unique.mp4"},
	}
	clashes := detectCollisions(pairs)
	if len(clashes) != 1 {
		t.Fatalf("want 1 collision, got %d: %+v", len(clashes), clashes)
	}
	if clashes[0].dst != "/proj/originals/clip.mp4" {
		t.Errorf("collision dst = %s", clashes[0].dst)
	}
}
