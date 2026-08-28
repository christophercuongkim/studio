// Package probe wraps ffprobe: it runs one JSON probe per media file and
// distills the streams/format output into the handful of fields studio cares
// about (plan §6 step 3, §4 media schema).
package probe

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/christophercuongkim/studio/internal/ffexec"
)

// Result is the distilled probe of a single media file.
type Result struct {
	DurationSec float64
	Width       int
	Height      int
	FPSNum      int
	FPSDen      int
	VCodec      string    // e.g. "hevc", "h264"
	HasAudio    bool      // at least one audio stream present
	CreatedAt   time.Time // zero if the file has no creation_time tag
	SizeBytes   int64
}

// FPS renders the frame rate as a trimmed decimal string ("29.97", "30",
// "23.976") for the manifest and the review UI. Returns "" if unknown.
func (r Result) FPS() string {
	if r.FPSNum == 0 || r.FPSDen == 0 {
		return ""
	}
	v := float64(r.FPSNum) / float64(r.FPSDen)
	// Three decimals is enough to distinguish 29.97/23.976/59.94; trim zeros so
	// integer rates render as "30" not "30.000".
	s := strconv.FormatFloat(v, 'f', 3, 64)
	s = strings.TrimRight(s, "0")
	s = strings.TrimRight(s, ".")
	return s
}

// ffprobe's JSON shape, only the fields we read.
type rawProbe struct {
	Streams []struct {
		CodecType    string `json:"codec_type"`
		CodecName    string `json:"codec_name"`
		Width        int    `json:"width"`
		Height       int    `json:"height"`
		RFrameRate   string `json:"r_frame_rate"`
		AvgFrameRate string `json:"avg_frame_rate"`
	} `json:"streams"`
	Format struct {
		Duration string `json:"duration"`
		Size     string `json:"size"`
		Tags     struct {
			CreationTime string `json:"creation_time"`
		} `json:"tags"`
	} `json:"format"`
}

// File probes a media file on disk.
func File(ctx context.Context, path string) (Result, error) {
	out, err := ffexec.Run(ctx, "ffprobe",
		"-v", "error",
		"-print_format", "json",
		"-show_format",
		"-show_streams",
		path,
	)
	if err != nil {
		return Result{}, err
	}
	return Parse(out)
}

// Parse distills raw ffprobe JSON into a Result. It is separate from File so
// the parser is unit-testable against fixtures without invoking ffprobe.
func Parse(data []byte) (Result, error) {
	var raw rawProbe
	if err := json.Unmarshal(data, &raw); err != nil {
		return Result{}, fmt.Errorf("parse ffprobe json: %w", err)
	}

	var r Result
	var haveVideo bool
	for _, s := range raw.Streams {
		switch s.CodecType {
		case "video":
			if haveVideo {
				continue // first video stream wins
			}
			haveVideo = true
			r.VCodec = s.CodecName
			r.Width = s.Width
			r.Height = s.Height
			// Prefer r_frame_rate; fall back to avg_frame_rate (VFR sources).
			if n, d, ok := parseRational(s.RFrameRate); ok {
				r.FPSNum, r.FPSDen = n, d
			} else if n, d, ok := parseRational(s.AvgFrameRate); ok {
				r.FPSNum, r.FPSDen = n, d
			}
		case "audio":
			r.HasAudio = true
		}
	}
	if !haveVideo {
		return Result{}, fmt.Errorf("no video stream found")
	}

	if raw.Format.Duration != "" {
		if d, err := strconv.ParseFloat(raw.Format.Duration, 64); err == nil {
			r.DurationSec = d
		}
	}
	if raw.Format.Size != "" {
		if n, err := strconv.ParseInt(raw.Format.Size, 10, 64); err == nil {
			r.SizeBytes = n
		}
	}
	if ct := raw.Format.Tags.CreationTime; ct != "" {
		if t, err := time.Parse(time.RFC3339, ct); err == nil {
			r.CreatedAt = t.UTC()
		}
	}
	return r, nil
}

// parseRational parses ffprobe's "num/den" frame-rate strings. It rejects the
// "0/0" that ffprobe emits for streams without a meaningful rate.
func parseRational(s string) (num, den int, ok bool) {
	numStr, denStr, found := strings.Cut(s, "/")
	if !found {
		return 0, 0, false
	}
	n, err1 := strconv.Atoi(numStr)
	d, err2 := strconv.Atoi(denStr)
	if err1 != nil || err2 != nil || n == 0 || d == 0 {
		return 0, 0, false
	}
	return n, d, true
}
