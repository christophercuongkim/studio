package search

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/christophercuongkim/studio/internal/manifest"
)

func clip(id, desc string, rating int, dur float64, status string, created time.Time) manifest.Clip {
	return manifest.Clip{
		ID: id, Seq: 1, Stem: "RAW_" + id,
		Files:     manifest.Files{Original: "originals/RAW_" + id + ".MP4", Proxy: "proxy/RAW_" + id + ".mp4"},
		Media:     manifest.Media{DurationSec: dur, CreatedAt: created, VCodec: "hevc"},
		ProxyInfo: manifest.ProxyInfo{Source: manifest.ProxyGenerated},
		Review:    manifest.Review{Status: status, Rating: rating, Desc: desc},
	}
}

func writeManifest(t *testing.T, dir, title, cam string, clips []manifest.Clip) {
	t.Helper()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	man := &manifest.Manifest{
		SchemaVersion: manifest.SchemaVersion,
		Shoot:         manifest.Shoot{Root: dir, Title: title, CamCode: cam, CreatedAt: time.Now()},
		Clips:         clips,
	}
	if err := manifest.Save(dir, man); err != nil {
		t.Fatal(err)
	}
}

// tree builds a searchRoot with three shoots (one corrupt) and a hidden dir.
func tree(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	aug1 := time.Date(2026, 8, 1, 12, 0, 0, 0, time.UTC)
	aug2 := time.Date(2026, 8, 2, 12, 0, 0, 0, time.UTC)
	jul15 := time.Date(2026, 7, 15, 12, 0, 0, 0, time.UTC)

	writeManifest(t, filepath.Join(root, "shootA"), "lake trip", "DJI", []manifest.Clip{
		clip("c-001", "sunset dock", 5, 42, manifest.StatusKept, aug1),
		clip("c-002", "pier walk", 3, 120, manifest.StatusPending, aug2),
	})
	writeManifest(t, filepath.Join(root, "shootB"), "city night", "A", []manifest.Clip{
		clip("c-001", "neon sign", 4, 10, manifest.StatusKept, jul15),
	})
	// Corrupt manifest must warn, not fail.
	os.MkdirAll(filepath.Join(root, "shootC"), 0o755)
	os.WriteFile(filepath.Join(root, "shootC", "manifest.json"), []byte("{not valid"), 0o644)
	// Hidden dir must be skipped entirely.
	writeManifest(t, filepath.Join(root, ".trash"), "ignored", "X", []manifest.Clip{
		clip("c-001", "should not appear", 5, 1, manifest.StatusKept, aug1),
	})
	return root
}

func TestCollectWarnsOnCorruptAndSkipsHidden(t *testing.T) {
	root := tree(t)
	records, warnings := Collect([]string{root})
	if len(records) != 3 {
		t.Errorf("got %d records, want 3", len(records))
	}
	if len(warnings) != 1 {
		t.Errorf("got %d warnings, want 1 (corrupt): %v", len(warnings), warnings)
	}
	for _, r := range records {
		if r.Desc == "should not appear" {
			t.Error("hidden .trash shoot was not skipped")
		}
		if !filepath.IsAbs(r.OriginalPath) {
			t.Errorf("original path not absolute: %s", r.OriginalPath)
		}
	}
}

func TestFiltersAndSort(t *testing.T) {
	records, _ := Collect([]string{tree(t)})

	// Text AND across desc.
	if got := Apply(records, withTerms("sunset", "dock")); len(got) != 1 || got[0].Desc != "sunset dock" {
		t.Errorf("AND term match = %+v", names(got))
	}
	if got := Apply(records, withTerms("walk")); len(got) != 1 {
		t.Errorf("single term = %v", names(got))
	}

	// rating 4+ → sunset(5), neon(4), sorted rating desc.
	f := NewFilters()
	f.MinRating = 4
	got := Apply(records, f)
	if len(got) != 2 || got[0].Rating != 5 || got[1].Rating != 4 {
		t.Errorf("rating 4+ sort = %v", names(got))
	}

	// status kept.
	f = NewFilters()
	f.Status = manifest.StatusKept
	if got := Apply(records, f); len(got) != 2 {
		t.Errorf("status kept = %v", names(got))
	}

	// cam DJI (case-insensitive).
	f = NewFilters()
	f.Cam = "dji"
	if got := Apply(records, f); len(got) != 2 {
		t.Errorf("cam dji = %v", names(got))
	}

	// since 2026-08 → the two August clips.
	f = NewFilters()
	f.Since, _ = ParsePeriodStart("2026-08")
	if got := Apply(records, f); len(got) != 2 {
		t.Errorf("since 2026-08 = %v", names(got))
	}
	// until 2026-07 → only the July clip.
	f = NewFilters()
	f.UntilEnd, _ = ParsePeriodEnd("2026-07")
	if got := Apply(records, f); len(got) != 1 || got[0].Desc != "neon sign" {
		t.Errorf("until 2026-07 = %v", names(got))
	}

	// dur 5s..30s → neon(10).
	f = NewFilters()
	f.MinDur, f.MaxDur, _ = ParseDurRange("5s..30s")
	if got := Apply(records, f); len(got) != 1 || got[0].Desc != "neon sign" {
		t.Errorf("dur 5s..30s = %v", names(got))
	}

	// shoot substring.
	f = NewFilters()
	f.Shoot = "lake"
	if got := Apply(records, f); len(got) != 2 {
		t.Errorf("shoot lake = %v", names(got))
	}
}

func TestParseHelpers(t *testing.T) {
	if n, _ := ParseRating("4+"); n != 4 {
		t.Errorf("ParseRating(4+) = %d", n)
	}
	if _, err := ParseRating("9"); err == nil {
		t.Error("ParseRating(9) should error")
	}
	min, max, err := ParseDurRange("30s..2m")
	if err != nil || min != 30 || max != 120 {
		t.Errorf("ParseDurRange = %v,%v,%v", min, max, err)
	}
	if _, _, err := ParseDurRange("nope"); err == nil {
		t.Error("ParseDurRange(nope) should error")
	}
}

func withTerms(terms ...string) Filters {
	f := NewFilters()
	f.Terms = terms
	return f
}

func names(rs []Record) []string {
	var out []string
	for _, r := range rs {
		out = append(out, r.Desc)
	}
	return out
}
