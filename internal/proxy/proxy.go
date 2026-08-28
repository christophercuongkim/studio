// Package proxy generates edit/scrub proxies with ffmpeg (plan §6 step 6). It
// is separate from ingest so a future `studio proxies` rebuild command (§19)
// can reuse the exact same encoder settings.
package proxy

import (
	"context"

	"github.com/christophercuongkim/studio/internal/ffexec"
)

// Generate transcodes src into a 480p H.264 proxy at dst. The settings match
// the plan's fixed ffmpeg line: bicubic scale to 480 lines (even width),
// libx264 high/veryfast/crf 26, 96k AAC, faststart for instant scrubbing.
//
// It returns ffmpeg's full stderr (which ingest tees to its run log) alongside
// the error. On failure the error carries the stderr tail (via ffexec), which
// the caller records as the clip's proxyInfo.note.
func Generate(ctx context.Context, src, dst string) (stderr []byte, err error) {
	_, stderr, err = ffexec.Output(ctx, "ffmpeg",
		"-y",
		"-i", src,
		"-vf", "scale=-2:480:flags=bicubic",
		"-c:v", "libx264", "-profile:v", "high",
		"-preset", "veryfast", "-crf", "26",
		"-g", "48", "-pix_fmt", "yuv420p",
		"-c:a", "aac", "-b:a", "96k",
		"-movflags", "+faststart",
		dst,
	)
	return stderr, err
}
