package dashboard

import (
	"net/http"

	"github.com/christophercuongkim/studio/internal/server"
)

// handleReview mounts the standalone review UI (internal/server + the serve
// frontend) under /projects/{id}/review/ for one project. The serve frontend
// addresses its API and media with relative paths, so the same assets that
// power `studio serve` at "/" work unchanged one level down. Each project gets a
// cached, stateful review server so debounced/shutdown saves behave identically.
func (s *Server) handleReview(w http.ResponseWriter, r *http.Request) {
	dir, ok := decodeID(r.PathValue("id"))
	if !ok || !s.known(dir) {
		http.Error(w, "unknown project", http.StatusNotFound)
		return
	}
	rs, err := s.reviewFor(dir)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	prefix := "/projects/" + r.PathValue("id") + "/review"
	http.StripPrefix(prefix, rs.Handler()).ServeHTTP(w, r)
}

// reviewFor returns the cached review server for dir, creating it on first use.
func (s *Server) reviewFor(dir string) (*server.Server, error) {
	s.reviewMu.Lock()
	defer s.reviewMu.Unlock()
	if rs, ok := s.reviews[dir]; ok {
		return rs, nil
	}
	rs, err := server.New(dir)
	if err != nil {
		return nil, err
	}
	s.reviews[dir] = rs
	return rs, nil
}
