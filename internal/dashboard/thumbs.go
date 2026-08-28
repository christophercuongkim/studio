package dashboard

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/christophercuongkim/studio/internal/fsutil"
	"github.com/christophercuongkim/studio/internal/steps"
	"github.com/christophercuongkim/studio/internal/videoyaml"
)

// thumbnailName is the chosen thumbnail's fixed filename inside thumbs/ (also
// videoyaml's default). Picking a candidate copies it here.
const thumbnailName = "thumbnail.png"

// handleThumbsRun extracts thumbnail candidates, streaming progress. Body:
// {from: "render"|"clips", count}. Reuses steps.Thumbs — same code as the CLI.
func (s *Server) handleThumbsRun(w http.ResponseWriter, r *http.Request) {
	dir, ok := decodeID(r.PathValue("id"))
	if !ok || !s.known(dir) {
		http.Error(w, "unknown project", http.StatusNotFound)
		return
	}
	var req struct {
		From  string `json:"from"`
		Count int    `json:"count"`
	}
	json.NewDecoder(r.Body).Decode(&req)
	if req.From == "" {
		req.From = "render"
	}
	if req.Count == 0 {
		req.Count = 12
	}

	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	flusher, _ := w.(http.Flusher)
	emit := func(line string) {
		fmt.Fprintln(w, line)
		if flusher != nil {
			flusher.Flush()
		}
	}

	emit(fmt.Sprintf("▶ extracting thumbnail candidates from %s …", req.From))
	res, err := steps.Thumbs(dir, req.From, req.Count)
	if res != nil {
		for _, l := range res.Lines {
			emit(l)
		}
	}
	if err != nil {
		emit("ERROR: " + err.Error())
		return
	}
	emit("✓ thumbs done")
}

// candidatesResp is the gallery payload: candidate filenames (relative to
// thumbs/), the contact sheet, and which candidate is the current pick.
type candidatesResp struct {
	Candidates   []string `json:"candidates"`
	ContactSheet string   `json:"contactSheet"`
	Current      string   `json:"current"`
}

// handleCandidates lists the extracted candidate PNGs for the gallery.
func (s *Server) handleCandidates(w http.ResponseWriter, r *http.Request) {
	dir, ok := decodeID(r.PathValue("id"))
	if !ok || !s.known(dir) {
		http.Error(w, "unknown project", http.StatusNotFound)
		return
	}
	thumbsDir := filepath.Join(dir, "thumbs")
	entries, _ := os.ReadDir(thumbsDir)
	resp := candidatesResp{Candidates: []string{}}
	for _, e := range entries {
		name := e.Name()
		switch {
		case strings.HasPrefix(name, "candidate-") && strings.HasSuffix(name, ".png"):
			resp.Candidates = append(resp.Candidates, name)
		case name == "contact-sheet.png":
			resp.ContactSheet = name
		}
	}
	sort.Strings(resp.Candidates)

	// The current pick is whichever candidate matches thumbnail.png byte-for-byte.
	if want, err := os.ReadFile(filepath.Join(thumbsDir, thumbnailName)); err == nil {
		for _, c := range resp.Candidates {
			if got, err := os.ReadFile(filepath.Join(thumbsDir, c)); err == nil && bytes.Equal(got, want) {
				resp.Current = c
				break
			}
		}
	}
	writeJSON(w, resp)
}

// handleThumbMedia serves a PNG out of the project's thumbs/ directory. Only a
// bare filename is honored (no path separators) so the id can't escape thumbs/.
func (s *Server) handleThumbMedia(w http.ResponseWriter, r *http.Request) {
	dir, ok := decodeID(r.PathValue("id"))
	if !ok || !s.known(dir) {
		http.Error(w, "unknown project", http.StatusNotFound)
		return
	}
	file := r.PathValue("file")
	if file != filepath.Base(file) || !strings.HasSuffix(file, ".png") {
		http.Error(w, "bad file", http.StatusBadRequest)
		return
	}
	http.ServeFile(w, r, filepath.Join(dir, "thumbs", file))
}

// handleSetThumbnail copies the chosen candidate to thumbs/thumbnail.png and
// records it in video.yaml, so the thumbs step goes done and upload picks it up.
func (s *Server) handleSetThumbnail(w http.ResponseWriter, r *http.Request) {
	dir, ok := decodeID(r.PathValue("id"))
	if !ok || !s.known(dir) {
		http.Error(w, "unknown project", http.StatusNotFound)
		return
	}
	var req struct {
		File string `json:"file"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid body", http.StatusBadRequest)
		return
	}
	// Only an extracted candidate is a valid pick (guards against arbitrary paths).
	if req.File != filepath.Base(req.File) ||
		!strings.HasPrefix(req.File, "candidate-") || !strings.HasSuffix(req.File, ".png") {
		http.Error(w, "not a candidate", http.StatusBadRequest)
		return
	}
	src := filepath.Join(dir, "thumbs", req.File)
	data, err := os.ReadFile(src)
	if err != nil {
		http.Error(w, "candidate not found", http.StatusNotFound)
		return
	}
	if err := fsutil.WriteFileAtomic(filepath.Join(dir, "thumbs", thumbnailName), data, 0o644); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Point video.yaml at it if it isn't already (default is thumbs/thumbnail.png).
	if vy, err := videoyaml.Load(dir); err == nil {
		want := filepath.Join("thumbs", thumbnailName)
		if vy.Thumbnail != want {
			vy.Thumbnail = want
			_ = videoyaml.Save(dir, vy)
		}
	}
	writeJSON(w, map[string]string{"current": req.File})
}
