package dashboard

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/christophercuongkim/studio/internal/config"
	"github.com/christophercuongkim/studio/internal/drives"
	"github.com/christophercuongkim/studio/internal/project"
	"github.com/christophercuongkim/studio/internal/steps"
)

// handleDrives lists connected external volumes for the destination/source
// pickers.
func (s *Server) handleDrives(w http.ResponseWriter, r *http.Request) {
	ds := drives.External()
	if ds == nil {
		ds = []drives.Drive{} // encode as [] not null so the client can always .map
	}
	writeJSON(w, ds)
}

// handleCreate creates a new project. Body: {slug, title, date, root}. root is
// a chosen drive path, or "" to use externalRoot-if-connected / projectsRoot.
func (s *Server) handleCreate(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Slug  string `json:"slug"`
		Title string `json:"title"`
		Date  string `json:"date"`
		Root  string `json:"root"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid body", http.StatusBadRequest)
		return
	}
	cfg, err := config.Load()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	root, _ := cfg.ProjectsRootFor(req.Root)

	dir, err := project.Create(project.Options{
		ProjectsRoot: root,
		Slug:         req.Slug,
		Title:        req.Title,
		Date:         req.Date,
		CategoryID:   cfg.UploadDefaults.CategoryID,
		Privacy:      cfg.UploadDefaults.Privacy,
	}, time.Now())
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	writeJSON(w, map[string]string{"id": encodeID(dir), "dir": dir})
}

// handleIngest imports a card into a project, streaming output. Body:
// {source, copy, move}. Copy defaults on for removable sources unless move.
func (s *Server) handleIngest(w http.ResponseWriter, r *http.Request) {
	dir, ok := decodeID(r.PathValue("id"))
	if !ok || !s.known(dir) {
		http.Error(w, "unknown project", http.StatusNotFound)
		return
	}
	var req struct {
		Source      string `json:"source"`
		Copy        bool   `json:"copy"`
		Move        bool   `json:"move"`
		ClearSource bool   `json:"clearSource"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Source == "" {
		http.Error(w, "a source is required", http.StatusBadRequest)
		return
	}
	cfg, _ := config.Load()
	copyFiles := req.Copy || req.ClearSource || (!req.Move && drives.IsRemovable(req.Source))

	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	flusher, _ := w.(http.Flusher)
	emit := func(line string) {
		fmt.Fprintln(w, line)
		if flusher != nil {
			flusher.Flush()
		}
	}

	mode := "moving"
	if copyFiles {
		mode = "copying (card stays intact)"
		if req.ClearSource {
			mode = "copying, then emptying the card"
		}
	}
	emit(fmt.Sprintf("▶ ingesting from %s — %s …", req.Source, mode))

	res, err := steps.Ingest(dir, req.Source, cfg.CamCode, copyFiles, req.ClearSource)
	if res != nil {
		for _, l := range res.Lines {
			emit(l)
		}
	}
	if err != nil {
		emit("ERROR: " + err.Error())
		return
	}
	emit("✓ ingest done")
}
