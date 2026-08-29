package dashboard

import (
	"context"
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
// {source, clearSource}. Ingest always copies — the source is never moved — so
// a card is safe to eject; clearSource empties it after each copy verifies.
func (s *Server) handleIngest(w http.ResponseWriter, r *http.Request) {
	dir, ok := decodeID(r.PathValue("id"))
	if !ok || !s.known(dir) {
		http.Error(w, "unknown project", http.StatusNotFound)
		return
	}
	var req struct {
		Source      string `json:"source"`
		ClearSource bool   `json:"clearSource"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Source == "" {
		http.Error(w, "a source is required", http.StatusBadRequest)
		return
	}
	cfg, _ := config.Load()

	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	flusher, _ := w.(http.Flusher)
	emit := func(line string) {
		fmt.Fprintln(w, line)
		if flusher != nil {
			flusher.Flush()
		}
	}

	mode := "copying (card stays intact)"
	if req.ClearSource {
		mode = "copying, then emptying the card"
	}
	emit(fmt.Sprintf("▶ ingesting from %s — %s …", req.Source, mode))

	// context.Background so a browser disconnect doesn't abort a long copy
	// mid-run; emit streams each per-file progress line to the page live.
	res, err := steps.Ingest(context.Background(), dir, req.Source, cfg.CamCode, true, req.ClearSource, emit)
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
