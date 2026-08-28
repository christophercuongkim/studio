// Package apply implements `studio apply` and `studio undo`: the rename engine
// that gives kept clips their final names before Kdenlive import (plan §8).
// Renames are planned and fully validated before a single file moves, and every
// executed rename is journaled so `undo` can reverse it.
package apply

import (
	"encoding/json"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/christophercuongkim/studio/internal/manifest"
	"github.com/christophercuongkim/studio/internal/naming"
)

// rename is one planned/journaled file move, with project-relative paths.
type rename struct {
	Op    string `json:"op"`
	From  string `json:"from"`
	To    string `json:"to"`
	Group string `json:"group"` // clip id
}

// clipPlan is the post-rename state for one clip.
type clipPlan struct {
	id          string
	finalStem   string
	newOriginal string
	newProxy    string
	newSidecars []string
}

// Plan is a validated set of renames ready to execute (or print for --dry-run).
type Plan struct {
	Renames []rename
	clips   []clipPlan
}

// Empty reports whether there is nothing to apply.
func (p *Plan) Empty() bool { return len(p.Renames) == 0 }

// Build computes and validates the rename plan for every kept, not-yet-applied
// clip. It returns an error listing *all* violations if the plan is illegal,
// changing nothing.
func Build(man *manifest.Manifest, projectDir string) (*Plan, error) {
	cam := man.Shoot.CamCode
	p := &Plan{}

	// finalStem → clip id, seeded with already-applied clips so we catch
	// collisions against names that are already on disk.
	claimed := map[string]string{}
	for i := range man.Clips {
		c := &man.Clips[i]
		if c.Applied.Done && c.Applied.FinalStem != "" {
			claimed[c.Applied.FinalStem] = c.ID
		}
	}

	var violations []string
	for i := range man.Clips {
		c := &man.Clips[i]
		if c.Review.Status != manifest.StatusKept || c.Applied.Done {
			continue
		}
		if c.Review.Desc == "" {
			violations = append(violations, fmt.Sprintf("clip %s is kept but has an empty desc", c.ID))
			continue
		}
		if err := naming.ValidateDesc(c.Review.Desc); err != nil {
			violations = append(violations, fmt.Sprintf("clip %s: %v", c.ID, err))
			continue
		}

		finalStem := naming.FinalStem(c.Media.CreatedAt, cam, c.Seq, c.Review.Desc, c.Review.Take)
		if other, dup := claimed[finalStem]; dup {
			violations = append(violations, fmt.Sprintf("clip %s and %s both produce final name %q", c.ID, other, finalStem))
			continue
		}
		claimed[finalStem] = c.ID

		cp := clipPlan{id: c.ID, finalStem: finalStem}
		add := func(rel string) string {
			dir := path.Dir(rel)
			ext := path.Ext(rel)
			to := path.Join(dir, finalStem+ext)
			p.Renames = append(p.Renames, rename{Op: "rename", From: rel, To: to, Group: c.ID})
			return to
		}
		cp.newOriginal = add(c.Files.Original)
		if c.Files.Proxy != "" {
			cp.newProxy = add(c.Files.Proxy)
		}
		for _, s := range c.Files.Sidecars {
			cp.newSidecars = append(cp.newSidecars, add(s))
		}
		p.clips = append(p.clips, cp)
	}

	// Filesystem preconditions: sources exist, targets don't.
	for _, r := range p.Renames {
		if _, err := os.Stat(filepath.Join(projectDir, r.From)); err != nil {
			violations = append(violations, fmt.Sprintf("source missing: %s", r.From))
		}
		if _, err := os.Stat(filepath.Join(projectDir, r.To)); err == nil {
			violations = append(violations, fmt.Sprintf("target already exists: %s", r.To))
		}
	}

	if len(violations) > 0 {
		sort.Strings(violations)
		return nil, fmt.Errorf("apply plan is invalid; nothing changed:\n  - %s", strings.Join(violations, "\n  - "))
	}
	return p, nil
}

// DryRunTable renders the old→new rename table for --dry-run.
func (p *Plan) DryRunTable() string {
	if p.Empty() {
		return "nothing to apply (no kept, unapplied clips)"
	}
	var b strings.Builder
	for _, r := range p.Renames {
		fmt.Fprintf(&b, "%s\n  → %s\n", r.From, r.To)
	}
	fmt.Fprintf(&b, "%d file(s) across %d clip(s)", len(p.Renames), len(p.clips))
	return b.String()
}

// Execute journals and performs the renames, then updates and saves the
// manifest once. On the first rename error it stops and returns, leaving a
// recoverable state (the journal records what was done) without saving the
// manifest.
func Execute(man *manifest.Manifest, projectDir string, p *Plan, now time.Time) (logPath string, err error) {
	if p.Empty() {
		return "", nil
	}
	undoDir := filepath.Join(projectDir, "undo")
	if err := os.MkdirAll(undoDir, 0o755); err != nil {
		return "", err
	}
	logPath = filepath.Join(undoDir, fmt.Sprintf("apply-%s.jsonl", now.Format("20060102T150405Z")))
	logf, err := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o644)
	if err != nil {
		return "", err
	}
	defer logf.Close()

	enc := json.NewEncoder(logf)
	for _, r := range p.Renames {
		// Journal the intent (and fsync) before the rename so a crash leaves a
		// log that undo can replay.
		if err := enc.Encode(r); err != nil {
			return logPath, err
		}
		if err := logf.Sync(); err != nil {
			return logPath, err
		}
		from := filepath.Join(projectDir, r.From)
		to := filepath.Join(projectDir, r.To)
		if err := os.Rename(from, to); err != nil {
			return logPath, fmt.Errorf("rename %s → %s failed: %w\npartial state recorded in %s (run 'studio undo')", r.From, r.To, err, logPath)
		}
	}

	// All renames succeeded: commit the new paths and applied state.
	for _, cp := range p.clips {
		c := man.ClipByID(cp.id)
		c.Files.Original = cp.newOriginal
		c.Files.Proxy = cp.newProxy
		c.Files.Sidecars = cp.newSidecars
		applied := now
		c.Applied = manifest.Applied{Done: true, FinalStem: cp.finalStem, AppliedAt: &applied}
	}
	if err := manifest.Save(projectDir, man); err != nil {
		return logPath, fmt.Errorf("renames done but manifest save failed: %w", err)
	}
	return logPath, nil
}
