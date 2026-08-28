// Package chapters turns Kdenlive timeline guides into a YouTube chapter list
// (plan §11): frame positions become timestamps, YouTube's rules are enforced,
// and the block can be written into video.yaml's description between sentinels.
package chapters

import (
	"fmt"
	"sort"
	"strings"

	"github.com/christophercuongkim/studio/internal/kdenlive"
)

// YouTube chapter rules (plan §11).
const (
	minChapters   = 3
	minChapterSec = 10.0
)

// Sentinels delimit the managed chapters block in video.yaml:description.
const (
	StartSentinel = "<!-- chapters:start -->"
	EndSentinel   = "<!-- chapters:end -->"
)

// Chapter is one YouTube chapter.
type Chapter struct {
	StartSec float64
	Title    string
}

// Result is the computed chapter list plus any non-fatal warnings.
type Result struct {
	Chapters []Chapter
	Warnings []string
}

// FromProject computes chapters from a parsed .kdenlive document, shifting every
// guide by offsetSec (for renders that don't start at timeline 00:00). It
// returns an error if the guides can't be read or violate YouTube's rules.
func FromProject(root *kdenlive.Node, offsetSec float64) (*Result, error) {
	num, den, err := root.FPS()
	if err != nil {
		return nil, err
	}
	guides, err := root.Guides()
	if err != nil {
		return nil, err
	}
	if len(guides) == 0 {
		return nil, fmt.Errorf("no guides in project — add timeline guides in Kdenlive first")
	}

	res := &Result{}
	for i, g := range guides {
		sec := float64(g.Pos)*float64(den)/float64(num) + offsetSec
		if sec < 0 {
			sec = 0
		}
		title := strings.TrimSpace(g.Comment)
		if title == "" {
			title = fmt.Sprintf("Chapter %d", i+1)
		}
		res.Chapters = append(res.Chapters, Chapter{StartSec: sec, Title: title})
	}
	sort.SliceStable(res.Chapters, func(i, j int) bool {
		return res.Chapters[i].StartSec < res.Chapters[j].StartSec
	})

	// YouTube requires the first chapter at 00:00; synthesize one if needed.
	if res.Chapters[0].StartSec >= 0.5 {
		res.Chapters = append([]Chapter{{StartSec: 0, Title: "Intro"}}, res.Chapters...)
		res.Warnings = append(res.Warnings, "first guide was not at 00:00; synthesized \"00:00 Intro\"")
	}

	if len(res.Chapters) < minChapters {
		return nil, fmt.Errorf("only %d chapter(s); YouTube requires at least %d", len(res.Chapters), minChapters)
	}

	// Every chapter must be at least 10s long (gap to the next start).
	var offenders []string
	for i := 0; i+1 < len(res.Chapters); i++ {
		gap := res.Chapters[i+1].StartSec - res.Chapters[i].StartSec
		if gap < minChapterSec {
			offenders = append(offenders, fmt.Sprintf("%q at %s is only %.1fs before the next",
				res.Chapters[i].Title, timestamp(res.Chapters[i].StartSec), gap))
		}
	}
	if len(offenders) > 0 {
		return nil, fmt.Errorf("chapters shorter than %.0fs (YouTube minimum):\n  - %s",
			minChapterSec, strings.Join(offenders, "\n  - "))
	}

	return res, nil
}

// Text renders the chapter list as YouTube expects: one "TIMESTAMP Title" per
// line.
func (r *Result) Text() string {
	var b strings.Builder
	for _, c := range r.Chapters {
		fmt.Fprintf(&b, "%s %s\n", timestamp(c.StartSec), c.Title)
	}
	return strings.TrimRight(b.String(), "\n")
}

// timestamp formats seconds as MM:SS under an hour, H:MM:SS at or above.
func timestamp(sec float64) string {
	total := int(sec + 0.0001)
	h := total / 3600
	m := (total % 3600) / 60
	s := total % 60
	if h > 0 {
		return fmt.Sprintf("%d:%02d:%02d", h, m, s)
	}
	return fmt.Sprintf("%d:%02d", m, s)
}

// InsertIntoDescription returns description with block placed between the
// sentinels, replacing any existing block. If the sentinels are absent, the
// block is appended. Everything outside the sentinels is left byte-stable.
func InsertIntoDescription(description, block string) string {
	managed := StartSentinel + "\n" + block + "\n" + EndSentinel
	start := strings.Index(description, StartSentinel)
	end := strings.Index(description, EndSentinel)
	if start >= 0 && end > start {
		return description[:start] + managed + description[end+len(EndSentinel):]
	}
	if description == "" {
		return managed
	}
	return strings.TrimRight(description, "\n") + "\n\n" + managed
}
