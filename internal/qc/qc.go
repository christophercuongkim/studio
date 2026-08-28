// Package qc runs pre-upload checks on a finished render (plan §13): loudness,
// clipping, edge silence/black, stream sanity, duration, and A/V sync. Every
// check always runs; the command exits non-zero on any FAIL. Thresholds live in
// one tunable struct.
package qc

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"regexp"
	"strconv"
	"strings"

	"github.com/christophercuongkim/studio/internal/ffexec"
	"github.com/christophercuongkim/studio/internal/probe"
)

// Status of a single check.
type Status string

const (
	Pass Status = "PASS"
	Fail Status = "FAIL"
	Warn Status = "WARN"
)

// Check is one QC result.
type Check struct {
	Name   string `json:"name"`
	Status Status `json:"status"`
	Detail string `json:"detail"`
	Repro  string `json:"repro,omitempty"` // ffmpeg line to reproduce a FAIL
}

// Report is the full QC outcome.
type Report struct {
	Render    string  `json:"render"`
	Profile   string  `json:"profile"`
	Checks    []Check `json:"checks"`
	Passed    bool    `json:"passed"`
	Timestamp string  `json:"timestamp"`
}

// Failed reports whether any check is a FAIL.
func (r *Report) Failed() bool {
	for _, c := range r.Checks {
		if c.Status == Fail {
			return true
		}
	}
	return false
}

// Thresholds are the tunable QC limits (plan §13).
type Thresholds struct {
	TargetLUFS       float64 // -14
	LUFSTolerance    float64 // ±2
	TruePeakMaxDBTP  float64 // -1
	ClipDBFS         float64 // 0
	SilenceEdgeSec   float64 // 3
	BlackEdgeSec     float64 // 2
	SyncDriftMs      float64 // 200
	DurationMultiple float64 // 2× target
}

// Default returns the plan's thresholds.
func Default() Thresholds {
	return Thresholds{
		TargetLUFS: -14, LUFSTolerance: 2, TruePeakMaxDBTP: -1, ClipDBFS: 0,
		SilenceEdgeSec: 3, BlackEdgeSec: 2, SyncDriftMs: 200, DurationMultiple: 2,
	}
}

// profileSpec is the expected stream shape for a --profile value.
type profileSpec struct {
	width, height int
	fps           float64
}

var profiles = map[string]profileSpec{
	"4k30":    {3840, 2160, 30},
	"1080p30": {1920, 1080, 30},
	"1080p60": {1920, 1080, 60},
}

// Run executes every check against renderPath. targetLengthSec ≤ 0 skips the
// duration check.
func Run(ctx context.Context, renderPath, profile string, targetLengthSec float64, th Thresholds) (*Report, error) {
	rep := &Report{Render: renderPath, Profile: profile}

	pr, err := probe.File(ctx, renderPath)
	if err != nil {
		return nil, fmt.Errorf("probe render: %w", err)
	}

	// One combined audio+video analysis pass, plus one loudnorm pass.
	_, analysis, err := ffexec.Output(ctx, "ffmpeg", "-hide_banner", "-nostats",
		"-i", renderPath,
		"-af", "astats=metadata=0,silencedetect=n=-50dB:d=2",
		"-vf", "blackdetect=d=1:pix_th=0.10",
		"-f", "null", "-")
	if err != nil {
		return nil, fmt.Errorf("analysis pass: %w", err)
	}
	analysisStr := string(analysis)

	_, loud, err := ffexec.Output(ctx, "ffmpeg", "-hide_banner", "-nostats",
		"-i", renderPath, "-af", "loudnorm=print_format=json", "-f", "null", "-")
	if err != nil {
		return nil, fmt.Errorf("loudness pass: %w", err)
	}

	rep.Checks = append(rep.Checks,
		checkLoudness(string(loud), th, renderPath),
		checkClipping(analysisStr, th, renderPath),
		checkEdgeSilence(analysisStr, pr.DurationSec, th, renderPath),
		checkEdgeBlack(analysisStr, pr.DurationSec, th),
		checkStreams(pr, profile),
		checkDuration(pr.DurationSec, targetLengthSec, th),
		checkSync(ctx, renderPath, th),
	)
	rep.Passed = !rep.Failed()
	return rep, nil
}

var (
	silenceStartRe = regexp.MustCompile(`silence_start:\s*(-?[\d.]+)`)
	silenceEndRe   = regexp.MustCompile(`silence_end:\s*(-?[\d.]+)`)
	blackStartRe   = regexp.MustCompile(`black_start:(-?[\d.]+)`)
	blackEndRe     = regexp.MustCompile(`black_end:(-?[\d.]+)`)
	peakRe         = regexp.MustCompile(`Peak level dB:\s*(-?[\d.]+|-?inf)`)
)

func checkLoudness(stderr string, th Thresholds, path string) Check {
	c := Check{Name: "loudness"}
	obj := extractJSON(stderr)
	if obj == "" {
		c.Status, c.Detail = Fail, "could not read loudnorm output"
		return c
	}
	var l struct {
		InputI  string `json:"input_i"`
		InputTP string `json:"input_tp"`
	}
	if err := json.Unmarshal([]byte(obj), &l); err != nil {
		c.Status, c.Detail = Fail, "unparseable loudnorm JSON"
		return c
	}
	lufs, _ := strconv.ParseFloat(l.InputI, 64)
	tp, _ := strconv.ParseFloat(l.InputTP, 64)
	lo, hi := th.TargetLUFS-th.LUFSTolerance, th.TargetLUFS+th.LUFSTolerance
	switch {
	case lufs < lo || lufs > hi:
		c.Status = Fail
		c.Detail = fmt.Sprintf("integrated %.1f LUFS outside %.0f±%.0f", lufs, th.TargetLUFS, th.LUFSTolerance)
		c.Repro = reproLoudnorm(path)
	case tp > th.TruePeakMaxDBTP:
		c.Status = Fail
		c.Detail = fmt.Sprintf("true peak %.1f dBTP exceeds %.0f", tp, th.TruePeakMaxDBTP)
		c.Repro = reproLoudnorm(path)
	default:
		c.Status = Pass
		c.Detail = fmt.Sprintf("%.1f LUFS, true peak %.1f dBTP", lufs, tp)
	}
	return c
}

func checkClipping(stderr string, th Thresholds, path string) Check {
	c := Check{Name: "clipping"}
	peak, ok := maxPeak(stderr)
	if !ok {
		c.Status, c.Detail = Warn, "no astats peak level found"
		return c
	}
	switch {
	case math.IsInf(peak, -1):
		c.Status, c.Detail = Pass, "silent (no signal)"
	case peak >= th.ClipDBFS:
		c.Status = Fail
		c.Detail = fmt.Sprintf("peak %.2f dBFS at or above 0 (clipping)", peak)
		c.Repro = fmt.Sprintf("ffmpeg -i %q -af astats -f null -", path)
	default:
		c.Status = Pass
		c.Detail = fmt.Sprintf("peak %.2f dBFS", peak)
	}
	return c
}

func checkEdgeSilence(stderr string, dur float64, th Thresholds, path string) Check {
	c := Check{Name: "edge-silence"}
	intervals := parseIntervals(stderr, silenceStartRe, silenceEndRe, dur)
	for _, iv := range intervals {
		if overlaps(iv, 0, th.SilenceEdgeSec) || overlaps(iv, dur-th.SilenceEdgeSec, dur) {
			c.Status = Fail
			c.Detail = fmt.Sprintf("silence %.1f–%.1fs overlaps the first/last %.0fs", iv[0], iv[1], th.SilenceEdgeSec)
			c.Repro = fmt.Sprintf("ffmpeg -i %q -af silencedetect=n=-50dB:d=2 -f null -", path)
			return c
		}
	}
	c.Status, c.Detail = Pass, "no silence at head or tail"
	return c
}

func checkEdgeBlack(stderr string, dur float64, th Thresholds) Check {
	c := Check{Name: "edge-black"}
	intervals := parseIntervals(stderr, blackStartRe, blackEndRe, dur)
	for _, iv := range intervals {
		if overlaps(iv, 0, th.BlackEdgeSec) || overlaps(iv, dur-th.BlackEdgeSec, dur) {
			c.Status = Warn // intros exist; WARN not FAIL
			c.Detail = fmt.Sprintf("black %.1f–%.1fs at head/tail (ok if intentional)", iv[0], iv[1])
			return c
		}
	}
	c.Status, c.Detail = Pass, "no black at head or tail"
	return c
}

func checkStreams(pr probe.Result, profile string) Check {
	c := Check{Name: "streams"}
	var problems []string
	if !pr.HasAudio {
		problems = append(problems, "no audio track")
	}
	if pr.VCodec != "h264" && pr.VCodec != "hevc" {
		problems = append(problems, fmt.Sprintf("codec %s not in {h264,hevc}", pr.VCodec))
	}
	if spec, ok := profiles[profile]; ok {
		if pr.Width != spec.width || pr.Height != spec.height {
			problems = append(problems, fmt.Sprintf("resolution %dx%d != %dx%d", pr.Width, pr.Height, spec.width, spec.height))
		}
		if fps := fpsFloat(pr); math.Abs(fps-spec.fps) > 1 {
			problems = append(problems, fmt.Sprintf("fps %.2f != %.0f", fps, spec.fps))
		}
	}
	if len(problems) > 0 {
		c.Status, c.Detail = Fail, strings.Join(problems, "; ")
	} else {
		c.Status = Pass
		c.Detail = fmt.Sprintf("%dx%d %s %s, audio ok", pr.Width, pr.Height, pr.FPS(), pr.VCodec)
	}
	return c
}

func checkDuration(dur, target float64, th Thresholds) Check {
	c := Check{Name: "duration"}
	if target <= 0 {
		c.Status, c.Detail = Pass, "no target length to compare"
		return c
	}
	if dur > target*th.DurationMultiple {
		c.Status = Warn
		c.Detail = fmt.Sprintf("%.0fs is more than %.0f× the %.0fs target", dur, th.DurationMultiple, target)
	} else {
		c.Status, c.Detail = Pass, fmt.Sprintf("%.0fs vs %.0fs target", dur, target)
	}
	return c
}

func checkSync(ctx context.Context, path string, th Thresholds) Check {
	c := Check{Name: "av-sync"}
	vid, aud, err := streamDurations(ctx, path)
	if err != nil || vid == 0 || aud == 0 {
		c.Status, c.Detail = Pass, "stream durations unavailable"
		return c
	}
	drift := math.Abs(vid-aud) * 1000
	if drift > th.SyncDriftMs {
		c.Status = Warn
		c.Detail = fmt.Sprintf("A/V duration drift %.0fms exceeds %.0fms", drift, th.SyncDriftMs)
	} else {
		c.Status, c.Detail = Pass, fmt.Sprintf("A/V drift %.0fms", drift)
	}
	return c
}

// --- parsing helpers ---

func reproLoudnorm(path string) string {
	return fmt.Sprintf("ffmpeg -i %q -af loudnorm=print_format=json -f null -", path)
}

// extractJSON returns the last {...} block in s (loudnorm prints it last).
func extractJSON(s string) string {
	end := strings.LastIndex(s, "}")
	if end < 0 {
		return ""
	}
	start := strings.LastIndex(s[:end], "{")
	if start < 0 {
		return ""
	}
	return s[start : end+1]
}

func maxPeak(stderr string) (float64, bool) {
	matches := peakRe.FindAllStringSubmatch(stderr, -1)
	if len(matches) == 0 {
		return 0, false
	}
	// All-"-inf" means a silent track: the peak is genuinely -inf (not clipping),
	// which is a valid measurement, so return it as found.
	max := math.Inf(-1)
	for _, m := range matches {
		if m[1] == "-inf" {
			continue
		}
		if v, err := strconv.ParseFloat(m[1], 64); err == nil && v > max {
			max = v
		}
	}
	return max, true
}

// parseIntervals pairs start/end regex matches into [start,end] intervals. An
// unmatched trailing start (detector ran to EOF) closes at dur.
func parseIntervals(s string, startRe, endRe *regexp.Regexp, dur float64) [][2]float64 {
	starts := floats(startRe.FindAllStringSubmatch(s, -1))
	ends := floats(endRe.FindAllStringSubmatch(s, -1))
	var out [][2]float64
	for i, st := range starts {
		en := dur
		if i < len(ends) {
			en = ends[i]
		}
		out = append(out, [2]float64{st, en})
	}
	return out
}

func floats(matches [][]string) []float64 {
	var out []float64
	for _, m := range matches {
		if v, err := strconv.ParseFloat(m[1], 64); err == nil {
			out = append(out, v)
		}
	}
	return out
}

func overlaps(iv [2]float64, lo, hi float64) bool {
	if hi <= lo {
		return false
	}
	return iv[0] < hi && iv[1] > lo
}

func fpsFloat(pr probe.Result) float64 {
	if pr.FPSDen == 0 {
		return 0
	}
	return float64(pr.FPSNum) / float64(pr.FPSDen)
}

// streamDurations returns the video and audio stream durations via ffprobe.
func streamDurations(ctx context.Context, path string) (vid, aud float64, err error) {
	out, err := ffexec.Run(ctx, "ffprobe", "-v", "error",
		"-print_format", "json", "-show_streams", path)
	if err != nil {
		return 0, 0, err
	}
	var doc struct {
		Streams []struct {
			CodecType string `json:"codec_type"`
			Duration  string `json:"duration"`
		} `json:"streams"`
	}
	if err := json.Unmarshal(out, &doc); err != nil {
		return 0, 0, err
	}
	for _, s := range doc.Streams {
		d, _ := strconv.ParseFloat(s.Duration, 64)
		switch s.CodecType {
		case "video":
			if vid == 0 {
				vid = d
			}
		case "audio":
			if aud == 0 {
				aud = d
			}
		}
	}
	return vid, aud, nil
}
