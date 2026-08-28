package thumbs

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/christophercuongkim/studio/internal/ffexec"
)

// frameSize is the extraction resolution (plan §14). Frames are letterboxed to
// preserve aspect.
const frameW, frameH = 1280, 720

// scaleFilter fits the source into frameW×frameH without distortion.
var scaleFilter = fmt.Sprintf(
	"scale=%d:%d:force_original_aspect_ratio=decrease,pad=%d:%d:(ow-iw)/2:(oh-ih)/2",
	frameW, frameH, frameW, frameH)

// ExtractRender samples renderPath every N seconds (N = duration/60, min 2s)
// into rawDir, returning the frames with their timestamps.
func ExtractRender(ctx context.Context, renderPath, rawDir string, durationSec float64) ([]Frame, error) {
	n := durationSec / 60
	if n < 2 {
		n = 2
	}
	if err := os.MkdirAll(rawDir, 0o755); err != nil {
		return nil, err
	}
	pattern := filepath.Join(rawDir, "r-%04d.png")
	_, err := ffexec.Run(ctx, "ffmpeg", "-y", "-i", renderPath,
		"-vf", fmt.Sprintf("fps=1/%f,%s", n, scaleFilter),
		pattern)
	if err != nil {
		return nil, err
	}
	files, err := listPNGs(rawDir, "r-")
	if err != nil {
		return nil, err
	}
	frames := make([]Frame, len(files))
	for i, f := range files {
		frames[i] = Frame{Path: f, TimestampSec: float64(i) * n}
	}
	return frames, nil
}

// clipSource is one rating-5 original to sample.
type clipSource struct {
	Path        string
	DurationSec float64
	ID          string
}

// ExtractClips samples 5 frames from each source clip into rawDir.
func ExtractClips(ctx context.Context, sources []clipSource, rawDir string) ([]Frame, error) {
	if err := os.MkdirAll(rawDir, 0o755); err != nil {
		return nil, err
	}
	const perClip = 5
	var frames []Frame
	for _, src := range sources {
		dur := src.DurationSec
		if dur <= 0 {
			dur = float64(perClip) // fall back to ~1fps
		}
		pattern := filepath.Join(rawDir, "c-"+src.ID+"-%03d.png")
		_, err := ffexec.Run(ctx, "ffmpeg", "-y", "-i", src.Path,
			"-vf", fmt.Sprintf("fps=%d/%f,%s", perClip, dur, scaleFilter),
			pattern)
		if err != nil {
			return nil, err
		}
		files, err := listPNGs(rawDir, "c-"+src.ID+"-")
		if err != nil {
			return nil, err
		}
		step := dur / perClip
		for i, f := range files {
			frames = append(frames, Frame{Path: f, TimestampSec: float64(i) * step})
		}
	}
	return frames, nil
}

// listPNGs returns the sorted .png files in dir whose base name starts with
// prefix.
func listPNGs(dir, prefix string) ([]string, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	var out []string
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		name := e.Name()
		if strings.HasPrefix(name, prefix) && strings.HasSuffix(name, ".png") {
			out = append(out, filepath.Join(dir, name))
		}
	}
	sort.Strings(out)
	return out, nil
}
