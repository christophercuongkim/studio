package apply

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/christophercuongkim/studio/internal/manifest"
)

var testCreated = time.Date(2026, 8, 27, 14, 23, 1, 0, time.UTC)

// buildProj writes a project with the given clips' files on disk and a saved
// manifest, returning the project dir.
func buildProj(t *testing.T, clips []manifest.Clip) string {
	t.Helper()
	dir := t.TempDir()
	for _, c := range clips {
		writeFile(t, dir, c.Files.Original, "orig-"+c.ID)
		if c.Files.Proxy != "" {
			writeFile(t, dir, c.Files.Proxy, "proxy-"+c.ID)
		}
		for _, s := range c.Files.Sidecars {
			writeFile(t, dir, s, "side-"+c.ID)
		}
	}
	man := &manifest.Manifest{
		SchemaVersion: manifest.SchemaVersion,
		Shoot:         manifest.Shoot{Root: dir, Title: "trip", CreatedAt: testCreated, CamCode: "DJI"},
		Clips:         clips,
	}
	if err := manifest.Save(dir, man); err != nil {
		t.Fatal(err)
	}
	return dir
}

func writeFile(t *testing.T, dir, rel, content string) {
	t.Helper()
	p := filepath.Join(dir, rel)
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func keptClip(id string, seq int, desc string, withProxy, withSidecar bool) manifest.Clip {
	c := manifest.Clip{
		ID: id, Seq: seq, Stem: "RAW_" + id,
		Files:     manifest.Files{Original: "originals/RAW_" + id + ".MP4"},
		Media:     manifest.Media{CreatedAt: testCreated, VCodec: "hevc"},
		ProxyInfo: manifest.ProxyInfo{Source: manifest.ProxyGenerated},
		Review:    manifest.Review{Status: manifest.StatusKept, Desc: desc, Rating: 5},
	}
	if withProxy {
		c.Files.Proxy = "proxy/RAW_" + id + ".mp4"
	}
	if withSidecar {
		c.Files.Sidecars = []string{"originals/RAW_" + id + ".SRT"}
	}
	if !withProxy {
		c.ProxyInfo = manifest.ProxyInfo{Source: manifest.ProxyNone, Note: "n/a"}
	}
	return c
}

// snapshot records every regular file under dir (relative path → content),
// excluding the manifest and undo journals, for byte-identical comparison.
func snapshot(t *testing.T, dir string) map[string]string {
	t.Helper()
	out := map[string]string{}
	filepath.WalkDir(dir, func(p string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		rel, _ := filepath.Rel(dir, p)
		if rel == manifest.FileName || strings.HasPrefix(rel, "undo/") {
			return nil
		}
		b, _ := os.ReadFile(p)
		out[rel] = string(b)
		return nil
	})
	return out
}

func TestBuildDryRun(t *testing.T) {
	dir := buildProj(t, []manifest.Clip{keptClip("c-001", 1, "alpha", true, true)})
	man, _ := manifest.Load(dir)
	plan, err := Build(man, dir)
	if err != nil {
		t.Fatal(err)
	}
	table := plan.DryRunTable()
	if !strings.Contains(table, "originals/RAW_c-001.MP4") ||
		!strings.Contains(table, "20260827_DJI001_alpha.MP4") {
		t.Errorf("dry-run table missing rename:\n%s", table)
	}
	// Dry-run must not touch disk: original still present.
	if _, err := os.Stat(filepath.Join(dir, "originals/RAW_c-001.MP4")); err != nil {
		t.Error("dry-run should not have renamed anything")
	}
}

func TestApplyThenUndoRestoresByteIdentical(t *testing.T) {
	dir := buildProj(t, []manifest.Clip{
		keptClip("c-001", 1, "alpha", true, true),
		keptClip("c-002", 2, "beta", true, false),
		func() manifest.Clip {
			c := keptClip("c-003", 3, "skip", true, false)
			c.Review.Status = manifest.StatusRejected
			c.Review.Desc = "" // rejected with no desc → left untouched
			return c
		}(),
	})
	before := snapshot(t, dir)

	man, _ := manifest.Load(dir)
	plan, err := Build(man, dir)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := Execute(man, dir, plan, testCreated); err != nil {
		t.Fatal(err)
	}

	// Kept clips renamed; rejected clip untouched.
	if _, err := os.Stat(filepath.Join(dir, "originals/20260827_DJI001_alpha.MP4")); err != nil {
		t.Error("c-001 original not renamed")
	}
	if _, err := os.Stat(filepath.Join(dir, "proxy/20260827_DJI001_alpha.mp4")); err != nil {
		t.Error("c-001 proxy not renamed")
	}
	if _, err := os.Stat(filepath.Join(dir, "originals/RAW_c-003.MP4")); err != nil {
		t.Error("rejected c-003 should be untouched")
	}
	reMan, _ := manifest.Load(dir)
	if c := reMan.ClipByID("c-001"); !c.Applied.Done || c.Applied.FinalStem != "20260827_DJI001_alpha" {
		t.Errorf("c-001 applied state wrong: %+v", c.Applied)
	}

	// Undo restores the exact original tree.
	logPath, err := LatestLog(dir)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := Undo(dir, logPath); err != nil {
		t.Fatal(err)
	}
	after := snapshot(t, dir)
	if len(after) != len(before) {
		t.Fatalf("file count changed: before %d, after %d", len(before), len(after))
	}
	for rel, content := range before {
		if after[rel] != content {
			t.Errorf("file %s not restored (%q vs %q)", rel, after[rel], content)
		}
	}
	undoMan, _ := manifest.Load(dir)
	if c := undoMan.ClipByID("c-001"); c.Applied.Done {
		t.Error("undo did not reset applied state")
	}
}

func TestDuplicateFinalStemRejected(t *testing.T) {
	// Two kept clips with the same seq + desc collide on final name.
	dir := buildProj(t, []manifest.Clip{
		keptClip("c-001", 7, "dup", false, false),
		keptClip("c-002", 7, "dup", false, false),
	})
	man, _ := manifest.Load(dir)
	_, err := Build(man, dir)
	if err == nil || !strings.Contains(err.Error(), "both produce final name") {
		t.Fatalf("expected duplicate-name error, got %v", err)
	}
}

func TestEmptyDescRejected(t *testing.T) {
	dir := buildProj(t, []manifest.Clip{keptClip("c-001", 1, "", false, false)})
	man, _ := manifest.Load(dir)
	_, err := Build(man, dir)
	if err == nil || !strings.Contains(err.Error(), "empty desc") {
		t.Fatalf("expected empty-desc error, got %v", err)
	}
}

func TestTargetExistsRejected(t *testing.T) {
	dir := buildProj(t, []manifest.Clip{keptClip("c-001", 1, "alpha", false, false)})
	// Pre-create the target path so the plan must refuse.
	writeFile(t, dir, "originals/20260827_DJI001_alpha.MP4", "in the way")
	man, _ := manifest.Load(dir)
	_, err := Build(man, dir)
	if err == nil || !strings.Contains(err.Error(), "target already exists") {
		t.Fatalf("expected target-exists error, got %v", err)
	}
}

func TestMidRunFailureIsRecoverable(t *testing.T) {
	dir := buildProj(t, []manifest.Clip{keptClip("c-001", 1, "alpha", true, false)})
	before := snapshot(t, dir)

	man, _ := manifest.Load(dir)
	plan, err := Build(man, dir)
	if err != nil {
		t.Fatal(err)
	}

	// Make proxy/ read-only so the original rename (in originals/) succeeds but
	// the proxy rename fails mid-plan.
	proxyDir := filepath.Join(dir, "proxy")
	if err := os.Chmod(proxyDir, 0o555); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.Chmod(proxyDir, 0o755) })

	logPath, execErr := Execute(man, dir, plan, testCreated)
	if execErr == nil {
		t.Fatal("expected mid-run rename failure")
	}
	if logPath == "" {
		t.Fatal("expected a journal path even on failure")
	}
	// Manifest must NOT have been saved as applied. Use the schema-only loader:
	// the validating one would fail because c-001's original was renamed away,
	// leaving the on-disk manifest path stale until undo runs.
	reMan, err := manifest.LoadFile(filepath.Join(dir, manifest.FileName))
	if err != nil {
		t.Fatal(err)
	}
	if reMan.ClipByID("c-001").Applied.Done {
		t.Error("manifest should not record applied on partial failure")
	}

	// Restore perms and undo: the tree must come back byte-identical.
	os.Chmod(proxyDir, 0o755)
	if _, err := Undo(dir, logPath); err != nil {
		t.Fatal(err)
	}
	after := snapshot(t, dir)
	if !equalMaps(before, after) {
		t.Errorf("undo after partial failure did not restore tree:\nbefore=%v\nafter=%v", keys(before), keys(after))
	}
}

func equalMaps(a, b map[string]string) bool {
	if len(a) != len(b) {
		return false
	}
	for k, v := range a {
		if b[k] != v {
			return false
		}
	}
	return true
}

func keys(m map[string]string) []string {
	var ks []string
	for k := range m {
		ks = append(ks, k)
	}
	sort.Strings(ks)
	return ks
}

func TestApplyRoutesRejectedWithDescToNotUsing(t *testing.T) {
	dir := buildProj(t, []manifest.Clip{
		keptClip("c-001", 1, "keeper", true, false),
		func() manifest.Clip { // rejected but named → not_using/
			c := keptClip("c-002", 2, "blurry-take", true, true)
			c.Review.Status = manifest.StatusRejected
			return c
		}(),
		func() manifest.Clip { // rejected, no desc → untouched
			c := keptClip("c-003", 3, "x", false, false)
			c.Review.Status = manifest.StatusRejected
			c.Review.Desc = ""
			return c
		}(),
	})
	before := snapshot(t, dir)

	man, _ := manifest.Load(dir)
	plan, err := Build(man, dir)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := Execute(man, dir, plan, testCreated); err != nil {
		t.Fatal(err)
	}

	// Keeper renamed in place.
	if _, err := os.Stat(filepath.Join(dir, "originals/20260827_DJI001_keeper.MP4")); err != nil {
		t.Error("kept clip not renamed in place")
	}
	// Rejected-but-named clip: renamed AND moved into not_using/ (original, proxy, sidecar).
	for _, p := range []string{
		"not_using/20260827_DJI002_blurry-take.MP4",
		"not_using/20260827_DJI002_blurry-take.mp4",
		"not_using/20260827_DJI002_blurry-take.SRT",
	} {
		if _, err := os.Stat(filepath.Join(dir, p)); err != nil {
			t.Errorf("expected %s in not_using/: %v", p, err)
		}
	}
	// Rejected with no desc: left where it was.
	if _, err := os.Stat(filepath.Join(dir, "originals/RAW_c-003.MP4")); err != nil {
		t.Error("rejected, undescribed clip should be untouched")
	}

	// Manifest records the not_using clip as applied with its final stem.
	reMan, _ := manifest.Load(dir)
	if c := reMan.ClipByID("c-002"); !c.Applied.Done || c.Files.Original != "not_using/20260827_DJI002_blurry-take.MP4" {
		t.Errorf("c-002 applied/path wrong: %+v", c.Applied)
	}

	// Undo restores the byte-identical tree.
	logPath, _ := LatestLog(dir)
	if _, err := Undo(dir, logPath); err != nil {
		t.Fatal(err)
	}
	if !equalMaps(before, snapshot(t, dir)) {
		t.Error("undo did not restore the tree after not_using routing")
	}
}
