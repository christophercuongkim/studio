// Package pipeline reports where a project is in the studio workflow by reading
// the file-based state it already keeps (manifest.json, video.yaml, and the
// files each stage produces). Nothing extra is tracked — "where am I / what's
// next" is computed, so you can stop at any point and pick up later. This is the
// shared brain any guided front-end (status view, wizard, TUI) reads from.
package pipeline

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/christophercuongkim/studio/internal/chapters"
	"github.com/christophercuongkim/studio/internal/manifest"
	"github.com/christophercuongkim/studio/internal/videoyaml"
)

// depthCap bounds the project walk below each search root.
const depthCap = 6

// Step is one pipeline stage and whether it's complete.
type Step struct {
	Name   string `json:"name"` // short id: ingest, review, apply, …
	Done   bool   `json:"done"`
	Detail string `json:"detail"` // human note, e.g. "2 kept · 1 rejected"
}

// State is a project's full pipeline position.
type State struct {
	Dir   string `json:"dir"`
	Title string `json:"title"`
	Steps []Step `json:"steps"`
	Next  string `json:"next"` // name of the first incomplete step; "" when done
}

// NextStep returns the first incomplete step, or nil when the project is
// complete.
func (s *State) nextStep() string {
	for _, st := range s.Steps {
		if !st.Done {
			return st.Name
		}
	}
	return ""
}

// Detect computes a project's pipeline state from its files.
func Detect(projectDir string) *State {
	s := &State{Dir: projectDir, Title: filepath.Base(projectDir)}

	// Load what's present; missing pieces just mean early stages aren't done.
	man, _ := manifest.LoadFile(filepath.Join(projectDir, manifest.FileName))
	vy, _ := videoyaml.Load(projectDir)
	if man != nil && man.Shoot.Title != "" {
		s.Title = man.Shoot.Title
	} else if vy != nil && vy.Title != "" {
		s.Title = vy.Title
	}

	// Review tallies.
	var pending, kept, rejected, unappliedKept int
	if man != nil {
		for _, c := range man.Clips {
			switch c.Review.Status {
			case manifest.StatusKept:
				kept++
				if !c.Applied.Done {
					unappliedKept++
				}
			case manifest.StatusRejected:
				rejected++
			default:
				pending++
			}
		}
	}

	ingested := man != nil && len(man.Clips) > 0
	reviewed := ingested && pending == 0
	applied := reviewed && unappliedKept == 0
	scaffolded := hasGlob(projectDir, "*.kdenlive")
	rendered := vy != nil && fileExists(filepath.Join(projectDir, vy.Render))
	chaptered := vy != nil && strings.Contains(vy.Description, chapters.StartSentinel)
	qcOK := qcPassed(projectDir)
	thumbed := vy != nil && vy.Thumbnail != "" && fileExists(filepath.Join(projectDir, vy.Thumbnail))
	uploaded := vy != nil && vy.YouTube.VideoID != nil && *vy.YouTube.VideoID != ""
	archived := man != nil && man.ArchivedAt != nil

	s.Steps = []Step{
		{Name: "ingest", Done: ingested, Detail: countDetail(man)},
		{Name: "review", Done: reviewed, Detail: fmt.Sprintf("%d kept · %d rejected · %d pending", kept, rejected, pending)},
		{Name: "apply", Done: applied, Detail: appliedDetail(kept, unappliedKept)},
		{Name: "scaffold", Done: scaffolded, Detail: ""},
		{Name: "render", Done: rendered, Detail: renderDetail(vy)},
		{Name: "chapters", Done: chaptered, Detail: ""},
		{Name: "qc", Done: qcOK, Detail: ""},
		{Name: "thumbs", Done: thumbed, Detail: ""},
		{Name: "upload", Done: uploaded, Detail: uploadDetail(vy)},
		{Name: "archive", Done: archived, Detail: archiveDetail(man)},
	}
	s.Next = s.nextStep()
	return s
}

// FindProjects walks the search roots for project folders (those holding a
// manifest.json or video.yaml), sorted by path.
func FindProjects(roots []string) []string {
	seen := map[string]bool{}
	var out []string
	for _, root := range roots {
		absRoot, err := filepath.Abs(root)
		if err != nil {
			absRoot = root
		}
		filepath.WalkDir(absRoot, func(path string, d os.DirEntry, err error) error {
			if err != nil || !d.IsDir() {
				return nil
			}
			name := d.Name()
			if path != absRoot && strings.HasPrefix(name, ".") {
				return filepath.SkipDir
			}
			if depthBelow(absRoot, path) > depthCap {
				return filepath.SkipDir
			}
			if fileExists(filepath.Join(path, manifest.FileName)) || fileExists(filepath.Join(path, videoyaml.FileName)) {
				if !seen[path] {
					seen[path] = true
					out = append(out, path)
				}
			}
			return nil
		})
	}
	sort.Strings(out)
	return out
}

// --- detail helpers ---

func countDetail(man *manifest.Manifest) string {
	if man == nil {
		return ""
	}
	return fmt.Sprintf("%d clips", len(man.Clips))
}

func appliedDetail(kept, unapplied int) string {
	if kept == 0 {
		return "no kept clips"
	}
	return fmt.Sprintf("%d/%d kept applied", kept-unapplied, kept)
}

func renderDetail(vy *videoyaml.VideoYAML) string {
	if vy == nil {
		return ""
	}
	return vy.Render
}

func uploadDetail(vy *videoyaml.VideoYAML) string {
	if vy != nil && vy.YouTube.VideoID != nil && *vy.YouTube.VideoID != "" {
		return "youtu.be/" + *vy.YouTube.VideoID
	}
	return ""
}

func archiveDetail(man *manifest.Manifest) string {
	if man != nil && man.ArchivedAt != nil {
		return man.ArchivedAt.Format("2006-01-02")
	}
	return ""
}

// --- fs helpers ---

func fileExists(p string) bool {
	_, err := os.Stat(p)
	return err == nil
}

func hasGlob(dir, pattern string) bool {
	m, _ := filepath.Glob(filepath.Join(dir, pattern))
	return len(m) > 0
}

func qcPassed(dir string) bool {
	data, err := os.ReadFile(filepath.Join(dir, "qc-report.json"))
	if err != nil {
		return false
	}
	var r struct {
		Passed bool `json:"passed"`
	}
	return json.Unmarshal(data, &r) == nil && r.Passed
}

func depthBelow(root, path string) int {
	rel, err := filepath.Rel(root, path)
	if err != nil || rel == "." {
		return 0
	}
	return strings.Count(rel, string(filepath.Separator)) + 1
}
