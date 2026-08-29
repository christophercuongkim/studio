package dashboard

import (
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/christophercuongkim/studio/internal/config"
	"github.com/christophercuongkim/studio/internal/drives"
)

// dirEntry is one subfolder offered by the drill-down picker.
type dirEntry struct {
	Name string `json:"name"`
	Path string `json:"path"`
}

// dirListing is the GET /api/dirs response.
type dirListing struct {
	Path   string     `json:"path"`   // the folder listed (cleaned, absolute)
	Parent string     `json:"parent"` // parent folder, or "" at a browse-root boundary
	Dirs   []dirEntry `json:"dirs"`   // immediate subfolders, sorted
}

// rootEntry is a starting point for the picker: a drive or a project root.
type rootEntry struct {
	Label string `json:"label"`
	Path  string `json:"path"`
}

// browseRoots are the folders the picker may descend from: every connected
// external drive plus the configured project roots. Browsing is confined to
// these so the localhost UI can't wander the whole filesystem.
func browseRoots() []string {
	var roots []string
	for _, d := range drives.External() {
		roots = append(roots, d.Path)
	}
	if cfg, err := config.Load(); err == nil {
		for _, r := range []string{cfg.ProjectsRoot, cfg.ExternalRoot} {
			if r != "" {
				roots = append(roots, r)
			}
		}
	}
	return roots
}

// withinRoot reports whether path is at or below one of roots, returning the
// matching root. path must already be filepath.Clean'd so ".." can't escape.
func withinRoot(path string, roots []string) (string, bool) {
	for _, r := range roots {
		if path == r || strings.HasPrefix(path, r+string(filepath.Separator)) {
			return r, true
		}
	}
	return "", false
}

// handleRoots lists the picker's starting points: the internal default root
// plus each connected drive. The client needs this because it doesn't otherwise
// know where ProjectsRoot lives.
func (s *Server) handleRoots(w http.ResponseWriter, r *http.Request) {
	out := []rootEntry{}
	if cfg, err := config.Load(); err == nil && cfg.ProjectsRoot != "" {
		out = append(out, rootEntry{Label: "Internal (default)", Path: cfg.ProjectsRoot})
	}
	for _, d := range drives.External() {
		out = append(out, rootEntry{Label: d.Label, Path: d.Path})
	}
	writeJSON(w, out)
}

// handleDirs lists the immediate subfolders of ?path= for the drill-down picker.
// It refuses any path outside the browse roots (see browseRoots).
func (s *Server) handleDirs(w http.ResponseWriter, r *http.Request) {
	raw := r.URL.Query().Get("path")
	if raw == "" {
		http.Error(w, "path is required", http.StatusBadRequest)
		return
	}
	listing, ok, err := browseListing(filepath.Clean(raw), browseRoots())
	if !ok {
		http.Error(w, "path is not under a connected drive or project root", http.StatusForbidden)
		return
	}
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	writeJSON(w, listing)
}

// browseListing lists path's immediate subfolders and computes its in-bounds
// parent. ok is false when path escapes the roots (a 403); err is a read error
// (a 400). Split out from handleDirs so the containment and parent-clamp logic
// is unit-testable without a live config or HTTP server.
func browseListing(path string, roots []string) (dirListing, bool, error) {
	root, ok := withinRoot(path, roots)
	if !ok {
		return dirListing{}, false, nil
	}
	entries, err := os.ReadDir(path)
	if err != nil {
		return dirListing{}, true, err
	}
	dirs := []dirEntry{}
	for _, e := range entries {
		// Hide dot-directories (.Spotlight-V100, .Trash-1000, .fseventsd …) —
		// picker noise the user never wants as a project dest or footage source.
		if e.IsDir() && !strings.HasPrefix(e.Name(), ".") {
			dirs = append(dirs, dirEntry{Name: e.Name(), Path: filepath.Join(path, e.Name())})
		}
	}
	sort.Slice(dirs, func(i, j int) bool {
		return strings.ToLower(dirs[i].Name) < strings.ToLower(dirs[j].Name)
	})
	// Offer "up" only while it stays within the same root — the user can't climb
	// above the drive/root they started from.
	parent := ""
	if path != root {
		if p := filepath.Dir(path); p != path {
			if _, ok := withinRoot(p, roots); ok {
				parent = p
			}
		}
	}
	return dirListing{Path: path, Parent: parent, Dirs: dirs}, true, nil
}
