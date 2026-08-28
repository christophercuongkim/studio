// Package server implements `studio serve`: a localhost review UI backed by the
// project manifest (plan §7). It holds the manifest in memory behind a mutex,
// applies PATCHes from the browser, and saves atomically — debounced during use
// and flushed on shutdown so no edit is lost.
package server

import (
	"encoding/json"
	"fmt"
	"net/http"
	"path/filepath"
	"sync"
	"time"

	"github.com/christophercuongkim/studio/internal/manifest"
	"github.com/christophercuongkim/studio/internal/naming"
	"github.com/christophercuongkim/studio/internal/webui"
)

// saveDebounce is how long after the last edit the manifest is written (plan
// §7.1: debounced ≤ 500 ms).
const saveDebounce = 500 * time.Millisecond

// Server serves the review UI and API for one project.
type Server struct {
	dir     string
	camCode string

	mu    sync.Mutex
	man   *manifest.Manifest
	dirty bool
	timer *time.Timer
	now   func() time.Time // injectable for tests
}

// New loads the project manifest and returns a ready Server.
func New(projectDir string) (*Server, error) {
	man, err := manifest.Load(projectDir)
	if err != nil {
		return nil, err
	}
	return &Server{
		dir:     projectDir,
		camCode: man.Shoot.CamCode,
		man:     man,
		now:     func() time.Time { return time.Now().UTC() },
	}, nil
}

// Handler returns the HTTP routes. Method+wildcard patterns need Go 1.22+.
func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/manifest", s.handleManifest)
	mux.HandleFunc("PATCH /api/clips/{id}", s.handlePatch)
	mux.HandleFunc("GET /api/preview-name/{id}", s.handlePreviewName)
	mux.HandleFunc("GET /media/proxy/{id}", s.handleProxy)
	// Vendored SeaKim design system (tokens + fonts), shared by both UIs.
	mux.Handle("GET /seakim/", http.StripPrefix("/seakim/", http.FileServer(http.FS(webui.SeakimFS()))))
	// Everything else is the embedded frontend (index.html, app.js, app.css).
	mux.Handle("GET /", http.FileServer(http.FS(webui.ServeFS())))
	return mux
}

// Flush writes the manifest if it has unsaved edits. Safe to call repeatedly.
func (s *Server) Flush() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.flushLocked()
}

func (s *Server) flushLocked() error {
	if !s.dirty {
		return nil
	}
	if err := manifest.Save(s.dir, s.man); err != nil {
		return err
	}
	s.dirty = false
	return nil
}

// Close stops the debounce timer and flushes any pending edit.
func (s *Server) Close() error {
	s.mu.Lock()
	if s.timer != nil {
		s.timer.Stop()
	}
	s.mu.Unlock()
	return s.Flush()
}

// touch marks the manifest dirty and (re)schedules a debounced save. Callers
// must hold s.mu.
func (s *Server) touch() {
	s.dirty = true
	if s.timer != nil {
		s.timer.Stop()
	}
	s.timer = time.AfterFunc(saveDebounce, func() {
		if err := s.Flush(); err != nil {
			fmt.Printf("serve: background save failed: %v\n", err)
		}
	})
}

func (s *Server) handleManifest(w http.ResponseWriter, r *http.Request) {
	s.mu.Lock()
	defer s.mu.Unlock()
	writeJSON(w, http.StatusOK, s.man)
}

// patchReq is decoded field-by-field via a raw map so absent vs null vs value
// are distinguishable (take can be cleared with null).
func (s *Server) handlePatch(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	var raw map[string]json.RawMessage
	if err := json.NewDecoder(r.Body).Decode(&raw); err != nil {
		httpError(w, http.StatusBadRequest, "invalid JSON body: %v", err)
		return
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	clip := s.man.ClipByID(id)
	if clip == nil {
		httpError(w, http.StatusNotFound, "no clip %q", id)
		return
	}

	if v, ok := raw["status"]; ok {
		var status string
		if err := json.Unmarshal(v, &status); err != nil {
			httpError(w, http.StatusBadRequest, "status: %v", err)
			return
		}
		switch status {
		case manifest.StatusPending, manifest.StatusKept, manifest.StatusRejected:
			clip.Review.Status = status
		default:
			httpError(w, http.StatusBadRequest, "invalid status %q", status)
			return
		}
	}
	if v, ok := raw["rating"]; ok {
		var rating int
		if err := json.Unmarshal(v, &rating); err != nil {
			httpError(w, http.StatusBadRequest, "rating: %v", err)
			return
		}
		if rating < 0 || rating > 5 {
			httpError(w, http.StatusBadRequest, "rating %d out of range 0–5", rating)
			return
		}
		clip.Review.Rating = rating
	}
	if v, ok := raw["desc"]; ok {
		var desc string
		if err := json.Unmarshal(v, &desc); err != nil {
			httpError(w, http.StatusBadRequest, "desc: %v", err)
			return
		}
		if err := naming.ValidateDesc(desc); err != nil {
			httpError(w, http.StatusBadRequest, "%v", err)
			return
		}
		clip.Review.Desc = desc
	}
	if v, ok := raw["take"]; ok {
		var take *int
		if err := json.Unmarshal(v, &take); err != nil {
			httpError(w, http.StatusBadRequest, "take: %v", err)
			return
		}
		if take != nil && *take <= 0 {
			take = nil // non-positive clears the take
		}
		clip.Review.Take = take
	}

	now := s.now()
	clip.Review.ReviewedAt = &now
	s.touch()

	writeJSON(w, http.StatusOK, clip)
}

func (s *Server) handlePreviewName(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	s.mu.Lock()
	defer s.mu.Unlock()
	clip := s.man.ClipByID(id)
	if clip == nil {
		httpError(w, http.StatusNotFound, "no clip %q", id)
		return
	}
	stem := naming.FinalStem(clip.Media.CreatedAt, s.camCode, clip.Seq, clip.Review.Desc, clip.Review.Take)
	writeJSON(w, http.StatusOK, map[string]string{"name": stem})
}

func (s *Server) handleProxy(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	s.mu.Lock()
	clip := s.man.ClipByID(id)
	var proxyRel string
	if clip != nil {
		proxyRel = clip.Files.Proxy
	}
	s.mu.Unlock()

	if clip == nil {
		httpError(w, http.StatusNotFound, "no clip %q", id)
		return
	}
	if proxyRel == "" {
		httpError(w, http.StatusNotFound, "clip %q has no proxy", id)
		return
	}
	// ServeFile handles range requests (scrubbing) and sets video/mp4 by ext.
	http.ServeFile(w, r, filepath.Join(s.dir, proxyRel))
}

func writeJSON(w http.ResponseWriter, code int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(v)
}

func httpError(w http.ResponseWriter, code int, format string, args ...any) {
	http.Error(w, fmt.Sprintf(format, args...), code)
}
