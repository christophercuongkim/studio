package thumbs

import (
	"context"
	"image"
	"image/color"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func TestSharpnessDistinguishesBlurFromDetail(t *testing.T) {
	// Uniform gray: Laplacian ~0 everywhere → low sharpness.
	flat := image.NewRGBA(image.Rect(0, 0, 32, 32))
	for y := range 32 {
		for x := range 32 {
			flat.Set(x, y, color.RGBA{128, 128, 128, 255})
		}
	}
	// Per-pixel checkerboard: maximal high-frequency detail → high sharpness.
	checker := image.NewRGBA(image.Rect(0, 0, 32, 32))
	for y := range 32 {
		for x := range 32 {
			v := uint8(0)
			if (x+y)%2 == 0 {
				v = 255
			}
			checker.Set(x, y, color.RGBA{v, v, v, 255})
		}
	}
	flatS, checkerS := Sharpness(flat), Sharpness(checker)
	if !(checkerS > flatS*100) {
		t.Errorf("checker sharpness %.2f should greatly exceed flat %.2f", checkerS, flatS)
	}
}

func TestSelectDropsBlurryAndSpreads(t *testing.T) {
	// 8 frames; even-indexed are sharp, odd are blurry; timestamps 0..70.
	var frames []Frame
	for i := range 8 {
		sharp := 100.0
		if i%2 == 1 {
			sharp = 1.0 // blurry half
		}
		frames = append(frames, Frame{Path: "x", TimestampSec: float64(i * 10), Sharpness: sharp})
	}
	got := Select(frames, 3)
	if len(got) != 3 {
		t.Fatalf("got %d, want 3", len(got))
	}
	// Output sorted by timestamp.
	for i := 1; i < len(got); i++ {
		if got[i].TimestampSec < got[i-1].TimestampSec {
			t.Errorf("not sorted by timestamp: %v", got)
		}
	}
	// All chosen must be from the sharp half.
	for _, f := range got {
		if f.Sharpness < 50 {
			t.Errorf("selected a blurry frame: %+v", f)
		}
	}
	// Spread: first and last sharp frames (ts 0 and 60) should be picked.
	if got[0].TimestampSec != 0 || got[len(got)-1].TimestampSec != 60 {
		t.Errorf("expected the extremes 0 and 60, got %v", []float64{got[0].TimestampSec, got[len(got)-1].TimestampSec})
	}
}

func TestSelectSmallPoolReturnsAll(t *testing.T) {
	frames := []Frame{
		{TimestampSec: 5, Sharpness: 1},
		{TimestampSec: 1, Sharpness: 9},
	}
	got := Select(frames, 10)
	if len(got) != 2 || got[0].TimestampSec != 1 {
		t.Errorf("small pool should return all, sorted by ts: %v", got)
	}
}

// TestRunFromRender exercises extraction, scoring, selection, candidate writing,
// and contact-sheet generation end-to-end against a real render.
func TestRunFromRender(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping thumbs integration in -short mode")
	}
	if _, err := exec.LookPath("ffmpeg"); err != nil {
		t.Skip("ffmpeg not on PATH")
	}
	dir := t.TempDir()
	render := filepath.Join(dir, "render.mp4")
	// A moving pattern so frames differ and have real detail.
	gen := exec.Command("ffmpeg", "-y", "-f", "lavfi",
		"-i", "testsrc2=s=640x360:r=30:d=20", "-c:v", "libx264", "-pix_fmt", "yuv420p", render)
	if out, err := gen.CombinedOutput(); err != nil {
		t.Fatalf("gen: %v\n%s", err, out)
	}

	res, err := Run(context.Background(), Options{
		ProjectDir: dir, RenderPath: render, Source: FromRender, Count: 5,
	}, nil)
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if len(res.Candidates) == 0 || len(res.Candidates) > 5 {
		t.Errorf("got %d candidates, want 1..5", len(res.Candidates))
	}
	for _, c := range res.Candidates {
		if _, err := os.Stat(c); err != nil {
			t.Errorf("candidate missing: %s", c)
		}
	}
	if _, err := os.Stat(res.ContactSheet); err != nil {
		t.Errorf("contact sheet missing: %v", err)
	}
	// raw/ scratch dir must be cleaned up.
	if _, err := os.Stat(filepath.Join(dir, "thumbs", "raw")); !os.IsNotExist(err) {
		t.Errorf("raw/ should be deleted")
	}
}
