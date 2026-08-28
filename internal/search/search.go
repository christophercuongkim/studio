// Package search scans searchRoots for manifests and filters clips across every
// shoot ever ingested (plan §12). It is read-only and keeps no index — it walks
// and filters in memory each run.
package search

import (
	"io/fs"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/christophercuongkim/studio/internal/manifest"
)

// depthCap bounds how deep the walk descends below each root (plan §12).
const depthCap = 6

// Record is one clip flattened with its shoot context, ready to filter/print.
type Record struct {
	Rating       int
	DurationSec  float64
	CreatedAt    time.Time
	Cam          string
	Shoot        string
	Desc         string
	Stem         string
	Status       string
	OriginalPath string // absolute
	ProxyPath    string // absolute; "" if none
}

// Name is the human label: the review description, or the raw stem if unnamed.
func (r Record) Name() string {
	if r.Desc != "" {
		return r.Desc
	}
	return r.Stem
}

// Collect walks the roots for manifest.json files and returns every clip as a
// Record, plus a warning per manifest that could not be read (never fatal).
// Paths are resolved against each manifest's own directory, so a moved tree
// still yields correct absolute paths.
func Collect(roots []string) (records []Record, warnings []string) {
	seen := map[string]bool{}
	for _, root := range roots {
		absRoot, err := filepath.Abs(root)
		if err != nil {
			absRoot = root
		}
		filepath.WalkDir(absRoot, func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				return nil // unreadable dir: skip quietly
			}
			if d.IsDir() {
				name := d.Name()
				if path != absRoot && strings.HasPrefix(name, ".") {
					return fs.SkipDir
				}
				if depthBelow(absRoot, path) > depthCap {
					return fs.SkipDir
				}
				return nil
			}
			if d.Name() != manifest.FileName {
				return nil
			}
			if seen[path] {
				return nil
			}
			seen[path] = true

			man, err := manifest.LoadFile(path)
			if err != nil {
				warnings = append(warnings, path+": "+err.Error())
				return nil
			}
			projectDir := filepath.Dir(path)
			for _, c := range man.Clips {
				records = append(records, toRecord(projectDir, man.Shoot, c))
			}
			return nil
		})
	}
	return records, warnings
}

func toRecord(projectDir string, shoot manifest.Shoot, c manifest.Clip) Record {
	rec := Record{
		Rating:       c.Review.Rating,
		DurationSec:  c.Media.DurationSec,
		CreatedAt:    c.Media.CreatedAt,
		Cam:          shoot.CamCode,
		Shoot:        shoot.Title,
		Desc:         c.Review.Desc,
		Stem:         c.Stem,
		Status:       c.Review.Status,
		OriginalPath: filepath.Join(projectDir, c.Files.Original),
	}
	if c.Files.Proxy != "" {
		rec.ProxyPath = filepath.Join(projectDir, c.Files.Proxy)
	}
	return rec
}

// depthBelow returns how many path segments path is below root.
func depthBelow(root, path string) int {
	rel, err := filepath.Rel(root, path)
	if err != nil || rel == "." {
		return 0
	}
	return strings.Count(rel, string(filepath.Separator)) + 1
}

// Sort orders records rating desc, then creation date desc (plan §12).
func Sort(records []Record) {
	sort.SliceStable(records, func(i, j int) bool {
		if records[i].Rating != records[j].Rating {
			return records[i].Rating > records[j].Rating
		}
		return records[i].CreatedAt.After(records[j].CreatedAt)
	})
}
