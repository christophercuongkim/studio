package probe

import (
	"context"
	"os/exec"
	"path/filepath"
	"testing"
)

// TestFileAgainstRealFfprobe exercises the exec path end-to-end: it synthesizes
// a tiny clip with ffmpeg and probes it with the real ffprobe. Skipped in
// -short mode or when the binaries are absent, so unit runs stay hermetic.
func TestFileAgainstRealFfprobe(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping ffprobe integration test in -short mode")
	}
	for _, bin := range []string{"ffmpeg", "ffprobe"} {
		if _, err := exec.LookPath(bin); err != nil {
			t.Skipf("%s not on PATH", bin)
		}
	}

	dir := t.TempDir()
	clip := filepath.Join(dir, "clip.mp4")
	ctx := context.Background()

	gen := exec.CommandContext(ctx, "ffmpeg", "-y",
		"-f", "lavfi", "-i", "testsrc2=size=1280x720:rate=30000/1001:duration=1",
		"-f", "lavfi", "-i", "sine=frequency=440:duration=1",
		"-c:v", "libx264", "-pix_fmt", "yuv420p", "-c:a", "aac",
		"-metadata", "creation_time=2026-08-27T14:23:01.000000Z",
		clip,
	)
	if out, err := gen.CombinedOutput(); err != nil {
		t.Fatalf("ffmpeg gen failed: %v\n%s", err, out)
	}

	r, err := File(ctx, clip)
	if err != nil {
		t.Fatalf("File: %v", err)
	}
	if r.Width != 1280 || r.Height != 720 {
		t.Errorf("dims = %dx%d, want 1280x720", r.Width, r.Height)
	}
	if r.VCodec != "h264" {
		t.Errorf("vcodec = %q, want h264", r.VCodec)
	}
	if !r.HasAudio {
		t.Error("expected audio stream")
	}
	if r.FPS() != "29.97" {
		t.Errorf("fps = %q, want 29.97", r.FPS())
	}
	if r.CreatedAt.IsZero() {
		t.Error("expected creation_time to parse")
	}
	if r.SizeBytes == 0 {
		t.Error("expected non-zero size")
	}
}

// TestFileMissingBinary confirms a missing binary surfaces ErrNotFound rather
// than a cryptic exec error.
func TestFileMissingBinary(t *testing.T) {
	// Point PATH at an empty dir so ffprobe cannot be found.
	t.Setenv("PATH", t.TempDir())
	_, err := File(context.Background(), "whatever.mp4")
	if err == nil {
		t.Fatal("expected error with empty PATH")
	}
}
