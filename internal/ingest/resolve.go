package ingest

import (
	"fmt"
	"math"

	"github.com/christophercuongkim/studio/internal/probe"
)

// proxyDurationTolerance is the allowed gap between an original and a candidate
// camera proxy before the proxy is considered a mismatch (plan §6 step 6).
const proxyDurationTolerance = 1.5 // seconds

// proxyAdoptable decides whether a camera-provided low-res file can stand in as
// the edit proxy: it must be H.264, agree with the original on audio presence,
// and be within ~1.5s of the original's duration. The reason is returned for
// the ingest log when the candidate is rejected.
func proxyAdoptable(cand, orig probe.Result) (ok bool, reason string) {
	if cand.VCodec != "h264" {
		return false, fmt.Sprintf("codec %q is not h264", cand.VCodec)
	}
	if cand.HasAudio != orig.HasAudio {
		return false, fmt.Sprintf("audio mismatch (candidate hasAudio=%v, original hasAudio=%v)", cand.HasAudio, orig.HasAudio)
	}
	if d := math.Abs(cand.DurationSec - orig.DurationSec); d > proxyDurationTolerance {
		return false, fmt.Sprintf("duration differs by %.2fs (> %.1fs)", d, proxyDurationTolerance)
	}
	return true, ""
}
