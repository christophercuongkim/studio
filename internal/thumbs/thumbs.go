package thumbs

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/christophercuongkim/studio/internal/manifest"
	"github.com/christophercuongkim/studio/internal/probe"
)

// renderDuration probes the render's duration, defaulting to 120s if unknown so
// extraction still samples a reasonable number of frames.
func renderDuration(ctx context.Context, path string) float64 {
	if r, err := probe.File(ctx, path); err == nil && r.DurationSec > 0 {
		return r.DurationSec
	}
	return 120
}

// Source selects which frames to sample.
type Source string

const (
	FromRender Source = "render"
	FromClips  Source = "clips"
)

// Options configures a thumbs run.
type Options struct {
	ProjectDir string
	RenderPath string // used when Source == render
	Source     Source
	Count      int
}

// Result reports the outcome.
type Result struct {
	Candidates   []string // written candidate PNG paths
	ContactSheet string
	Scanned      int
}

// Run extracts candidate frames, scores and spreads them, writes the winners
// plus a contact sheet, and deletes the scratch raw/ dir (plan §14).
func Run(ctx context.Context, opts Options, man *manifest.Manifest) (*Result, error) {
	thumbsDir := filepath.Join(opts.ProjectDir, "thumbs")
	rawDir := filepath.Join(thumbsDir, "raw")
	if err := os.MkdirAll(thumbsDir, 0o755); err != nil {
		return nil, err
	}

	var frames []Frame
	var err error
	switch opts.Source {
	case FromRender:
		dur := renderDuration(ctx, opts.RenderPath)
		frames, err = ExtractRender(ctx, opts.RenderPath, rawDir, dur)
	case FromClips:
		sources := ratingFiveSources(opts.ProjectDir, man)
		if len(sources) == 0 {
			return nil, fmt.Errorf("no rating-5 clips to sample; rate some in 'studio serve' first")
		}
		frames, err = ExtractClips(ctx, sources, rawDir)
	default:
		return nil, fmt.Errorf("unknown source %q", opts.Source)
	}
	if err != nil {
		return nil, err
	}
	if len(frames) == 0 {
		return nil, fmt.Errorf("no frames extracted")
	}

	// Score every frame's sharpness.
	for i := range frames {
		img, err := loadPNG(frames[i].Path)
		if err != nil {
			return nil, err
		}
		frames[i].Sharpness = Sharpness(img)
	}

	winners := Select(frames, opts.Count)

	res := &Result{Scanned: len(frames)}
	for i, f := range winners {
		dst := filepath.Join(thumbsDir, fmt.Sprintf("candidate-%02d-%s.png", i+1, tsSlug(f.TimestampSec)))
		if err := copyFile(f.Path, dst); err != nil {
			return nil, err
		}
		// Rewrite the winner's path so the contact sheet loads the kept copy.
		winners[i].Path = dst
		res.Candidates = append(res.Candidates, dst)
	}

	sheet, err := contactSheet(winners, 4)
	if err != nil {
		return nil, err
	}
	res.ContactSheet = filepath.Join(thumbsDir, "contact-sheet.png")
	if err := writePNG(res.ContactSheet, sheet); err != nil {
		return nil, err
	}

	// raw/ is tool-created scratch — safe to delete (plan §14, only tool files).
	if err := os.RemoveAll(rawDir); err != nil {
		return nil, err
	}
	return res, nil
}

func ratingFiveSources(projectDir string, man *manifest.Manifest) []clipSource {
	var out []clipSource
	if man == nil {
		return out
	}
	for _, c := range man.Clips {
		if c.Review.Rating == 5 {
			out = append(out, clipSource{
				Path:        filepath.Join(projectDir, c.Files.Original),
				DurationSec: c.Media.DurationSec,
				ID:          c.ID,
			})
		}
	}
	return out
}

func tsSlug(sec float64) string {
	s := int(sec + 0.5)
	return fmt.Sprintf("%02dm%02ds", s/60, s%60)
}

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()
	_, err = io.Copy(out, in)
	return err
}
