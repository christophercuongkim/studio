// Package steps is the single implementation of each runnable pipeline step
// (apply, scaffold, chapters, qc, thumbs, archive). Both the CLI commands and
// the dashboard call these, so there's no dashboard-only behavior — each action
// runs identical logic and produces the same output lines. Flag parsing and
// config resolution stay with the callers; steps take explicit parameters.
package steps

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/christophercuongkim/studio/internal/apply"
	"github.com/christophercuongkim/studio/internal/archive"
	"github.com/christophercuongkim/studio/internal/chapters"
	"github.com/christophercuongkim/studio/internal/fsutil"
	"github.com/christophercuongkim/studio/internal/ingest"
	"github.com/christophercuongkim/studio/internal/kdenlive"
	"github.com/christophercuongkim/studio/internal/manifest"
	"github.com/christophercuongkim/studio/internal/qc"
	"github.com/christophercuongkim/studio/internal/script"
	"github.com/christophercuongkim/studio/internal/thumbs"
	"github.com/christophercuongkim/studio/internal/videoyaml"
)

// Result is a step's human-readable output, plus a Failed flag for qc (whose
// "failure" is a checks result, not an operational error).
type Result struct {
	Lines  []string `json:"lines"`
	Failed bool     `json:"failed"`
}

func (r *Result) add(format string, args ...any) {
	r.Lines = append(r.Lines, fmt.Sprintf(format, args...))
}

// Text joins the result lines.
func (r *Result) Text() string { return strings.Join(r.Lines, "\n") }

// Ingest imports a card/dump into a project. copy leaves the source intact
// (always true for the current callers). progress, if non-nil, is called with a
// per-file copy line as it happens, so a caller streaming to a browser/terminal
// can show live progress; the final summary lines are still returned. A
// Ctrl-C-style interruption is reported as a normal line, not an error.
func Ingest(ctx context.Context, projectDir, source, camCode string, copy, clearSource bool, progress func(string)) (*Result, error) {
	sum, err := ingest.Run(ctx, ingest.Options{
		DumpDir:     source,
		ProjectDir:  projectDir,
		CamCode:     camCode,
		Copy:        copy,
		ClearSource: clearSource,
		Progress:    progress,
	})
	r := &Result{}
	if sum != nil {
		r.add("ingested %d clip(s): %d camera proxies adopted, %d generated, %d failed",
			sum.NewClips, sum.AdoptedProxies, sum.GeneratedProxies, sum.FailedProxies)
		if sum.SkippedExisting > 0 {
			r.add("  skipped %d already-ingested clip(s)", sum.SkippedExisting)
		}
		if sum.SourceCleared > 0 {
			r.add("  cleared %d source file(s) from the card", sum.SourceCleared)
		}
		for _, k := range sum.SourceKept {
			r.add("  kept on card (copy didn't verify): %s", filepath.Base(k))
		}
		for _, u := range sum.UnmatchedSidecars {
			r.add("  unmatched sidecar: %s", filepath.Base(u))
		}
		for _, p := range sum.ProbeFailures {
			r.add("  probe failed (left in dump): %s", p)
		}
		for _, p := range sum.CopyFailures {
			r.add("  copy failed (left on source): %s", p)
		}
	}
	if errors.Is(err, ingest.ErrInterrupted) {
		r.add("interrupted — re-run with append to finish")
		return r, nil
	}
	return r, err
}

// Apply renames kept clips (or previews with dryRun).
func Apply(dir string, dryRun bool) (*Result, error) {
	man, err := manifest.Load(dir)
	if err != nil {
		return nil, err
	}
	plan, err := apply.Build(man, dir)
	if err != nil {
		return nil, err
	}
	r := &Result{}
	if dryRun {
		r.add("%s", plan.DryRunTable())
		return r, nil
	}
	if plan.Empty() {
		r.add("nothing to apply (no kept, unapplied clips)")
		return r, nil
	}
	logPath, err := apply.Execute(man, dir, plan, time.Now().UTC())
	if err != nil {
		return r, err
	}
	r.add("applied %d rename(s); journal: %s", len(plan.Renames), logPath)
	return r, nil
}

// Scaffold generates a Kdenlive project from templatePath. outPath "" defaults
// to <project>/<title>.kdenlive; templatePath must be non-empty (resolved by the
// caller from --template or config).
func Scaffold(dir, templatePath, outPath string) (*Result, error) {
	man, err := manifest.Load(dir)
	if err != nil {
		return nil, err
	}
	if templatePath == "" {
		return nil, errors.New("no Kdenlive template configured; pass a template or set kdenliveTemplate in config")
	}
	if outPath == "" {
		title := man.Shoot.Title
		if title == "" {
			title = "project"
		}
		outPath = filepath.Join(dir, title+".kdenlive")
	}
	if _, err := os.Stat(outPath); err == nil {
		return nil, fmt.Errorf("output already exists: %s (scaffold never overwrites)", outPath)
	}

	absProject, _ := filepath.Abs(dir)
	var clips []kdenlive.ClipRef
	for _, c := range man.Clips {
		if c.Review.Status != manifest.StatusKept || !c.Applied.Done {
			continue
		}
		clips = append(clips, kdenlive.ClipRef{
			Resource:    filepath.Join(absProject, c.Files.Original),
			DurationSec: c.Media.DurationSec,
			Group:       c.Review.Group,
		})
	}
	if len(clips) == 0 {
		return nil, errors.New("no applied, kept clips to place; run apply first")
	}

	data, err := os.ReadFile(templatePath)
	if err != nil {
		return nil, fmt.Errorf("read template: %w", err)
	}
	root, err := kdenlive.Load(data)
	if err != nil {
		return nil, err
	}
	if err := kdenlive.Scaffold(root, clips); err != nil {
		return nil, err
	}
	if err := os.WriteFile(outPath, root.Render(), 0o644); err != nil {
		return nil, fmt.Errorf("write %s: %w", outPath, err)
	}
	r := &Result{}
	r.add("scaffolded %s with %d clip(s)", outPath, len(clips))
	return r, nil
}

// Chapters turns timeline guides into a chapter list, optionally writing them
// into video.yaml. projectFile "" picks the newest .kdenlive.
func Chapters(dir, projectFile string, write bool, offset time.Duration) (*Result, error) {
	kdePath := projectFile
	var err error
	if kdePath == "" {
		kdePath, err = NewestKdenlive(dir)
		if err != nil {
			return nil, err
		}
	} else if !filepath.IsAbs(kdePath) {
		kdePath = filepath.Join(dir, kdePath)
	}

	data, err := os.ReadFile(kdePath)
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", kdePath, err)
	}
	root, err := kdenlive.Load(data)
	if err != nil {
		return nil, err
	}
	res, err := chapters.FromProject(root, offset.Seconds())
	if err != nil {
		return nil, err
	}
	r := &Result{}
	for _, w := range res.Warnings {
		r.add("warn: %s", w)
	}
	if !write {
		r.add("%s", res.Text())
		return r, nil
	}
	vy, err := videoyaml.Load(dir)
	if err != nil {
		return r, err
	}
	vy.Description = chapters.InsertIntoDescription(vy.Description, res.Text())
	if err := videoyaml.Save(dir, vy); err != nil {
		return r, err
	}
	r.add("wrote %d chapters into video.yaml", len(res.Chapters))
	return r, nil
}

// QC runs the render checks. renderOverride "" uses video.yaml's render. The
// returned Result.Failed is true when a check fails (not an operational error);
// qc-report.json is written only on pass.
func QC(dir, renderOverride, profile string) (*Result, error) {
	renderPath := renderOverride
	if renderPath == "" {
		vy, err := videoyaml.Load(dir)
		if err != nil {
			return nil, fmt.Errorf("no render given and %w", err)
		}
		renderPath = filepath.Join(dir, vy.Render)
	} else if !filepath.IsAbs(renderPath) {
		renderPath = filepath.Join(dir, renderPath)
	}
	if _, err := os.Stat(renderPath); err != nil {
		return nil, fmt.Errorf("render not found: %s", renderPath)
	}

	rep, err := qc.Run(context.Background(), renderPath, profile, targetLengthSeconds(dir), qc.Default())
	if err != nil {
		return nil, err
	}
	r := &Result{}
	for _, c := range rep.Checks {
		r.add("%-5s %-13s %s", c.Status, c.Name, c.Detail)
		if c.Repro != "" {
			r.add("            %s", c.Repro)
		}
	}
	if rep.Failed() {
		r.Failed = true
		return r, nil
	}
	rep.Timestamp = time.Now().UTC().Format(time.RFC3339)
	data, _ := json.MarshalIndent(rep, "", "  ")
	if err := fsutil.WriteFileAtomic(filepath.Join(dir, "qc-report.json"), append(data, '\n'), 0o644); err != nil {
		return r, err
	}
	r.add("qc-report.json written")
	return r, nil
}

// Thumbs extracts and ranks thumbnail candidates.
func Thumbs(dir, from string, count int) (*Result, error) {
	src := thumbs.Source(from)
	if src != thumbs.FromRender && src != thumbs.FromClips {
		return nil, fmt.Errorf("from must be 'render' or 'clips', got %q", from)
	}
	opts := thumbs.Options{ProjectDir: dir, Source: src, Count: count}
	var man *manifest.Manifest
	var err error
	if src == thumbs.FromClips {
		if man, err = manifest.Load(dir); err != nil {
			return nil, err
		}
	} else {
		vy, err := videoyaml.Load(dir)
		if err != nil {
			return nil, err
		}
		opts.RenderPath = filepath.Join(dir, vy.Render)
	}
	res, err := thumbs.Run(context.Background(), opts, man)
	if err != nil {
		return nil, err
	}
	r := &Result{}
	r.add("scanned %d frames → %d candidates", res.Scanned, len(res.Candidates))
	for _, c := range res.Candidates {
		r.add("  %s", filepath.Base(c))
	}
	r.add("contact sheet: %s", res.ContactSheet)
	return r, nil
}

// Archive verifies, cold-stores, and prunes.
func Archive(dir, archiveRoot string, dryRun, keepProxies, force bool) (*Result, error) {
	res, err := archive.Run(context.Background(), archive.Options{
		ProjectDir:  dir,
		ArchiveRoot: archiveRoot,
		DryRun:      dryRun,
		KeepProxies: keepProxies,
		Force:       force,
	})
	if err != nil {
		return nil, err
	}
	r := &Result{}
	if res.DryRun {
		r.add("dry run: %d originals verified", res.Hashed)
		r.add("  would rsync → %s", res.ArchivePath)
		for _, p := range res.Pruned {
			r.add("  would prune %s/", p)
		}
		return r, nil
	}
	r.add("archived %d originals → %s", res.Hashed, res.ArchivePath)
	for _, p := range res.Pruned {
		r.add("  pruned %s/", p)
	}
	return r, nil
}

// NewestKdenlive returns the most recently modified *.kdenlive in dir.
func NewestKdenlive(dir string) (string, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return "", err
	}
	type cand struct {
		path string
		mod  time.Time
	}
	var cands []cand
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".kdenlive") {
			continue
		}
		info, err := e.Info()
		if err != nil {
			continue
		}
		cands = append(cands, cand{filepath.Join(dir, e.Name()), info.ModTime()})
	}
	if len(cands) == 0 {
		return "", fmt.Errorf("no .kdenlive file in %s (run scaffold first)", dir)
	}
	sort.Slice(cands, func(i, j int) bool { return cands[i].mod.After(cands[j].mod) })
	return cands[0].path, nil
}

// targetLengthSeconds reads script.md's targetLength (e.g. "8m") in seconds.
func targetLengthSeconds(dir string) float64 {
	data, err := os.ReadFile(filepath.Join(dir, "script.md"))
	if err != nil {
		return 0
	}
	sc, err := script.Parse(data)
	if err != nil || sc.TargetLength == "" {
		return 0
	}
	d, err := time.ParseDuration(sc.TargetLength)
	if err != nil {
		return 0
	}
	return d.Seconds()
}
