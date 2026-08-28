package qc

import (
	"context"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/christophercuongkim/studio/internal/probe"
)

var th = Default()

func TestCheckLoudness(t *testing.T) {
	good := `{"input_i":"-14.0","input_tp":"-2.0","input_lra":"5.0"}`
	if c := checkLoudness(good, th, "f"); c.Status != Pass {
		t.Errorf("good loudness = %s (%s)", c.Status, c.Detail)
	}
	tooLoud := `{"input_i":"-3.0","input_tp":"-2.0"}`
	if c := checkLoudness(tooLoud, th, "f"); c.Status != Fail || c.Repro == "" {
		t.Errorf("too loud should FAIL with repro: %+v", c)
	}
	peaking := `{"input_i":"-14.0","input_tp":"-0.3"}`
	if c := checkLoudness(peaking, th, "f"); c.Status != Fail {
		t.Errorf("true-peak over -1 should FAIL: %+v", c)
	}
	if c := checkLoudness("no json here", th, "f"); c.Status != Fail {
		t.Errorf("missing JSON should FAIL")
	}
}

func TestCheckClipping(t *testing.T) {
	if c := checkClipping("Peak level dB: -3.01", th, "f"); c.Status != Pass {
		t.Errorf("headroom should PASS: %+v", c)
	}
	if c := checkClipping("Peak level dB: 0.00", th, "f"); c.Status != Fail || c.Repro == "" {
		t.Errorf("0 dBFS should FAIL with repro: %+v", c)
	}
	// Multiple channels: the max wins.
	if c := checkClipping("Peak level dB: -6.0\nPeak level dB: 0.5", th, "f"); c.Status != Fail {
		t.Errorf("clipping channel should FAIL: %+v", c)
	}
	if c := checkClipping("Peak level dB: -inf", th, "f"); c.Status != Pass {
		t.Errorf("-inf (silence) should not FAIL clipping: %+v", c)
	}
}

func TestCheckEdgeSilence(t *testing.T) {
	// Leading silence 0–3.5s of a 60s render → FAIL.
	head := "silence_start: 0\nsilence_end: 3.5 | silence_duration: 3.5"
	if c := checkEdgeSilence(head, 60, th, "f"); c.Status != Fail {
		t.Errorf("head silence should FAIL: %+v", c)
	}
	// Trailing silence near the end → FAIL.
	tail := "silence_start: 58\nsilence_end: 60"
	if c := checkEdgeSilence(tail, 60, th, "f"); c.Status != Fail {
		t.Errorf("tail silence should FAIL: %+v", c)
	}
	// Silence in the middle → PASS.
	mid := "silence_start: 20\nsilence_end: 25"
	if c := checkEdgeSilence(mid, 60, th, "f"); c.Status != Pass {
		t.Errorf("mid silence should PASS: %+v", c)
	}
}

func TestCheckEdgeBlack(t *testing.T) {
	head := "black_start:0 black_end:1.5"
	if c := checkEdgeBlack(head, 60, th); c.Status != Warn {
		t.Errorf("head black should WARN: %+v", c)
	}
	none := "some other output"
	if c := checkEdgeBlack(none, 60, th); c.Status != Pass {
		t.Errorf("no black should PASS: %+v", c)
	}
}

func TestCheckStreams(t *testing.T) {
	good := probe.Result{Width: 1920, Height: 1080, VCodec: "h264", HasAudio: true, FPSNum: 30, FPSDen: 1}
	if c := checkStreams(good, "1080p30"); c.Status != Pass {
		t.Errorf("good streams should PASS: %+v", c)
	}
	wrongRes := probe.Result{Width: 1280, Height: 720, VCodec: "h264", HasAudio: true, FPSNum: 30, FPSDen: 1}
	if c := checkStreams(wrongRes, "1080p30"); c.Status != Fail {
		t.Errorf("wrong resolution should FAIL: %+v", c)
	}
	noAudio := probe.Result{Width: 1920, Height: 1080, VCodec: "h264", HasAudio: false, FPSNum: 30, FPSDen: 1}
	if c := checkStreams(noAudio, "1080p30"); c.Status != Fail {
		t.Errorf("no audio should FAIL: %+v", c)
	}
	badCodec := probe.Result{Width: 1920, Height: 1080, VCodec: "vp9", HasAudio: true, FPSNum: 30, FPSDen: 1}
	if c := checkStreams(badCodec, ""); c.Status != Fail {
		t.Errorf("bad codec should FAIL: %+v", c)
	}
}

func TestCheckDuration(t *testing.T) {
	if c := checkDuration(300, 100, th); c.Status != Warn {
		t.Errorf(">2x target should WARN: %+v", c)
	}
	if c := checkDuration(120, 100, th); c.Status != Pass {
		t.Errorf("within 2x should PASS: %+v", c)
	}
	if c := checkDuration(9999, 0, th); c.Status != Pass {
		t.Errorf("no target should PASS: %+v", c)
	}
}

func TestExtractJSONAndMaxPeak(t *testing.T) {
	s := "junk\n{ \"input_i\":\"-14\" }\nmore junk"
	if got := extractJSON(s); got != `{ "input_i":"-14" }` {
		t.Errorf("extractJSON = %q", got)
	}
	if p, ok := maxPeak("Peak level dB: -6\nPeak level dB: -2"); !ok || p != -2 {
		t.Errorf("maxPeak = %v,%v", p, ok)
	}
}

// TestRunReal runs the real ffmpeg passes against a clipping full-scale sine, so
// the loudness and clipping checks trip and the report has all checks.
func TestRunReal(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping ffmpeg qc integration in -short mode")
	}
	for _, b := range []string{"ffmpeg", "ffprobe"} {
		if _, err := exec.LookPath(b); err != nil {
			t.Skipf("%s not on PATH", b)
		}
	}
	dir := t.TempDir()
	render := filepath.Join(dir, "r.mp4")
	// Full-scale 1kHz sine (0 dBFS → clips, loud) over black 1080p30.
	gen := exec.Command("ffmpeg", "-y",
		"-f", "lavfi", "-i", "color=c=black:s=1920x1080:r=30:d=2",
		"-f", "lavfi", "-i", "sine=frequency=1000:duration=2",
		"-c:v", "libx264", "-pix_fmt", "yuv420p", "-c:a", "aac", "-shortest", render)
	if out, err := gen.CombinedOutput(); err != nil {
		t.Fatalf("gen: %v\n%s", err, out)
	}

	rep, err := Run(context.Background(), render, "1080p30", 0, Default())
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if len(rep.Checks) != 7 {
		t.Errorf("got %d checks, want 7", len(rep.Checks))
	}
	byName := map[string]Check{}
	for _, c := range rep.Checks {
		byName[c.Name] = c
	}
	// A bare 1kHz sine is nowhere near -14 LUFS, so loudness must FAIL — proving
	// the loudnorm pass and JSON parse work end-to-end.
	if byName["loudness"].Status != Fail {
		t.Errorf("bare sine should FAIL loudness: %+v", byName["loudness"])
	}
	if !rep.Failed() {
		t.Errorf("expected the report to fail overall; checks: %+v", rep.Checks)
	}
	// 1080p30 h264 + aac must satisfy the stream check.
	if byName["streams"].Status != Pass {
		t.Errorf("1080p30 h264+aac streams should PASS: %+v", byName["streams"])
	}
}
