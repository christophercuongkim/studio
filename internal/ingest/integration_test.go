package ingest

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/christophercuongkim/studio/internal/manifest"
)

// TestRunEndToEnd exercises the full ingest pipeline against a synthesized dump
// matching the M2 acceptance scenario: a DJI clip (original + adoptable LRF +
// SRT), a GoPro clip (GX original + GL LRV), a clip with no camera proxy (→
// generated), an orphan SRT (→ unmatched), and a corrupt MP4 (→ probe failure,
// left in the dump). It then checks --append idempotency.
func TestRunEndToEnd(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping ingest integration test in -short mode")
	}
	requireBinaries(t, "ffmpeg", "ffprobe")

	dump := t.TempDir()
	proj := t.TempDir()
	ctx := context.Background()

	// DJI: hevc original with audio, adoptable h264 LRF with audio, SRT sidecar.
	genVideo(t, filepath.Join(dump, "DJI_0001.MP4"), "libx265", true, "mp4")
	genVideo(t, filepath.Join(dump, "DJI_0001.LRF"), "libx264", true, "mp4")
	os.WriteFile(filepath.Join(dump, "DJI_0001.SRT"), []byte("1\n00:00:00,000 --> 00:00:01,000\nhi\n"), 0o644)

	// GoPro: GX original + GL LRV proxy (both h264 with audio → adoptable).
	genVideo(t, filepath.Join(dump, "GX010042.MP4"), "libx264", true, "mp4")
	genVideo(t, filepath.Join(dump, "GL010042.LRV"), "libx264", true, "mp4")

	// A clip with no camera proxy → studio must generate one.
	genVideo(t, filepath.Join(dump, "solo.MOV"), "libx264", true, "mov")

	// Orphan sidecar and corrupt media.
	os.WriteFile(filepath.Join(dump, "orphan.SRT"), []byte("stray"), 0o644)
	os.WriteFile(filepath.Join(dump, "corrupt.MP4"), []byte("not a real mp4 file at all"), 0o644)

	sum, err := Run(ctx, Options{DumpDir: dump, ProjectDir: proj, CamCode: "DJI", Jobs: 4, Copy: true})
	if err != nil {
		t.Fatalf("Run: %v", err)
	}

	if sum.NewClips != 3 {
		t.Errorf("NewClips = %d, want 3", sum.NewClips)
	}
	if sum.AdoptedProxies != 2 {
		t.Errorf("AdoptedProxies = %d, want 2 (DJI LRF + GoPro LRV)", sum.AdoptedProxies)
	}
	if sum.GeneratedProxies != 1 {
		t.Errorf("GeneratedProxies = %d, want 1 (solo)", sum.GeneratedProxies)
	}
	if len(sum.ProbeFailures) != 1 {
		t.Errorf("ProbeFailures = %v, want 1 (corrupt)", sum.ProbeFailures)
	}
	if len(sum.UnmatchedSidecars) != 1 {
		t.Errorf("UnmatchedSidecars = %v, want 1 (orphan)", sum.UnmatchedSidecars)
	}

	// Corrupt file must still be in the dump (never moved).
	if _, err := os.Stat(filepath.Join(dump, "corrupt.MP4")); err != nil {
		t.Errorf("corrupt.MP4 should remain in dump: %v", err)
	}
	// Orphan went to _unmatched, not the manifest.
	if _, err := os.Stat(filepath.Join(proj, "originals", "_unmatched", "orphan.SRT")); err != nil {
		t.Errorf("orphan.SRT should be under originals/_unmatched: %v", err)
	}

	// Manifest is valid and internally consistent.
	man, err := manifest.Load(proj)
	if err != nil {
		t.Fatalf("Load manifest: %v", err)
	}
	if len(man.Clips) != 3 {
		t.Fatalf("manifest has %d clips, want 3", len(man.Clips))
	}
	var djiAdopted bool
	for _, c := range man.Clips {
		if c.Files.Original == "" || c.Media.XXH64 == "" {
			t.Errorf("clip %s missing original/checksum", c.ID)
		}
		if c.Review.Status != manifest.StatusPending {
			t.Errorf("clip %s status = %s, want pending", c.ID, c.Review.Status)
		}
		if c.Stem == "DJI_0001" && c.ProxyInfo.Source == manifest.ProxyCamera {
			djiAdopted = true
		}
	}
	if !djiAdopted {
		t.Error("DJI LRF was not adopted as a camera proxy")
	}

	// Re-run without --append is refused.
	if _, err := Run(ctx, Options{DumpDir: dump, ProjectDir: proj}); err == nil {
		t.Error("second run without --append should be refused")
	}

	// --append with the same dump adds nothing (checksum dedupe); corrupt still
	// fails but is not fatal.
	sum2, err := Run(ctx, Options{DumpDir: dump, ProjectDir: proj, Append: true, Jobs: 4, Copy: true})
	if err != nil {
		t.Fatalf("append Run: %v", err)
	}
	if sum2.NewClips != 0 {
		t.Errorf("append added %d clips, want 0", sum2.NewClips)
	}
	if sum2.SkippedExisting != 3 {
		t.Errorf("append SkippedExisting = %d, want 3", sum2.SkippedExisting)
	}

	// Adding a genuinely new clip and appending picks up exactly one. It must
	// have distinct content — ffmpeg is deterministic, so a clip generated with
	// identical settings would hash identically and be deduped.
	genVideoDistinct(t, filepath.Join(dump, "fresh.MP4"))
	sum3, err := Run(ctx, Options{DumpDir: dump, ProjectDir: proj, Append: true, Jobs: 4, Copy: true})
	if err != nil {
		t.Fatalf("append fresh Run: %v", err)
	}
	if sum3.NewClips != 1 {
		t.Errorf("append fresh added %d clips, want 1", sum3.NewClips)
	}
}

func TestRunRefusesCollision(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping in -short mode")
	}
	requireBinaries(t, "ffmpeg", "ffprobe")

	dump := t.TempDir()
	proj := t.TempDir()
	// Two identically-named originals in different subdirectories flatten to the
	// same destination — must abort before moving anything.
	os.MkdirAll(filepath.Join(dump, "card1"), 0o755)
	os.MkdirAll(filepath.Join(dump, "card2"), 0o755)
	genVideo(t, filepath.Join(dump, "card1", "CLIP.MP4"), "libx264", true, "mp4")
	genVideo(t, filepath.Join(dump, "card2", "CLIP.MP4"), "libx264", true, "mp4")

	if _, err := Run(context.Background(), Options{DumpDir: dump, ProjectDir: proj, Jobs: 2}); err == nil {
		t.Fatal("expected collision error")
	}
	// Nothing should have moved.
	if _, err := os.Stat(filepath.Join(proj, "originals")); err == nil {
		t.Error("originals/ created despite collision abort")
	}
}

// --- test helpers ---

func requireBinaries(t *testing.T, names ...string) {
	t.Helper()
	for _, n := range names {
		if _, err := exec.LookPath(n); err != nil {
			t.Skipf("%s not on PATH", n)
		}
	}
}

// genVideo writes a ~1s clip at path using the given video codec, optionally
// with an audio track. format forces the muxer for extensions ffmpeg can't
// infer (.LRF/.LRV).
func genVideo(t *testing.T, path, vcodec string, withAudio bool, format string) {
	t.Helper()
	args := []string{"-y", "-f", "lavfi", "-i", "testsrc2=size=320x180:rate=30:duration=1"}
	if withAudio {
		args = append(args, "-f", "lavfi", "-i", "sine=frequency=440:duration=1")
	}
	args = append(args, "-c:v", vcodec, "-pix_fmt", "yuv420p")
	if vcodec == "libx265" {
		args = append(args, "-tag:v", "hvc1")
	}
	if withAudio {
		args = append(args, "-c:a", "aac", "-shortest")
	}
	args = append(args, "-f", format, path)

	cmd := exec.Command("ffmpeg", args...)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("ffmpeg gen %s failed: %v\n%s", path, err, out)
	}
}

// genVideoDistinct writes a clip whose bytes differ from genVideo's output
// (different duration and tone), so its checksum is unique.
func genVideoDistinct(t *testing.T, path string) {
	t.Helper()
	cmd := exec.Command("ffmpeg", "-y",
		"-f", "lavfi", "-i", "testsrc2=size=320x180:rate=30:duration=2",
		"-f", "lavfi", "-i", "sine=frequency=880:duration=2",
		"-c:v", "libx264", "-pix_fmt", "yuv420p", "-c:a", "aac", "-shortest",
		"-f", "mp4", path,
	)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("ffmpeg gen distinct %s failed: %v\n%s", path, err, out)
	}
}
