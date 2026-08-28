// Package dashboard is the `studio dashboard` guided web cockpit: a localhost
// web app that shows every project's pipeline stage (from internal/pipeline) and
// where to go next. Slice 1 is read-only — it mirrors `studio status` visually,
// SeaKim-styled, reusing the same embedded-server pattern as serve/prompt.
package dashboard

import (
	"encoding/base64"
	"encoding/json"
	"net/http"
	"path/filepath"

	"github.com/christophercuongkim/studio/internal/manifest"
	"github.com/christophercuongkim/studio/internal/pipeline"
	"github.com/christophercuongkim/studio/internal/videoyaml"
	"github.com/christophercuongkim/studio/internal/webui"
)

// Server serves the dashboard over the given search roots.
type Server struct {
	roots []string
}

// New builds a dashboard server scanning the given search roots for projects.
func New(roots []string) *Server {
	return &Server{roots: roots}
}

// Handler returns the HTTP routes (localhost only — the CLI binds 127.0.0.1).
func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/projects", s.handleList)
	mux.HandleFunc("GET /api/projects/{id}", s.handleDetail)
	mux.Handle("GET /seakim/", http.StripPrefix("/seakim/", http.FileServer(http.FS(webui.SeakimFS()))))
	mux.Handle("GET /", http.FileServer(http.FS(webui.DashboardFS())))
	return mux
}

// summary is a project's pipeline state plus its opaque id.
type summary struct {
	ID string `json:"id"`
	*pipeline.State
}

// detail adds video.yaml metadata and a clip summary.
type detail struct {
	summary
	Video videoMeta `json:"video"`
	Clips clipMeta  `json:"clips"`
}

type videoMeta struct {
	Title     string   `json:"title"`
	Privacy   string   `json:"privacy"`
	Tags      []string `json:"tags"`
	Render    string   `json:"render"`
	Thumbnail string   `json:"thumbnail"`
	VideoID   string   `json:"videoId"`
}

type clipMeta struct {
	Total     int `json:"total"`
	Kept      int `json:"kept"`
	Rejected  int `json:"rejected"`
	Pending   int `json:"pending"`
	Camera    int `json:"camera"`
	Generated int `json:"generated"`
	NoProxy   int `json:"noProxy"`
}

func (s *Server) handleList(w http.ResponseWriter, r *http.Request) {
	dirs := pipeline.FindProjects(s.roots)
	out := make([]summary, 0, len(dirs))
	for _, dir := range dirs {
		out = append(out, summary{ID: encodeID(dir), State: pipeline.Detect(dir)})
	}
	writeJSON(w, out)
}

func (s *Server) handleDetail(w http.ResponseWriter, r *http.Request) {
	dir, ok := decodeID(r.PathValue("id"))
	if !ok || !s.known(dir) {
		http.Error(w, "unknown project", http.StatusNotFound)
		return
	}
	d := detail{summary: summary{ID: encodeID(dir), State: pipeline.Detect(dir)}}

	if vy, err := videoyaml.Load(dir); err == nil {
		d.Video = videoMeta{
			Title: vy.Title, Privacy: vy.Privacy, Tags: vy.Tags,
			Render: vy.Render, Thumbnail: vy.Thumbnail,
		}
		if vy.YouTube.VideoID != nil {
			d.Video.VideoID = *vy.YouTube.VideoID
		}
	}
	if man, err := manifest.LoadFile(filepath.Join(dir, manifest.FileName)); err == nil {
		d.Clips.Total = len(man.Clips)
		for _, c := range man.Clips {
			switch c.Review.Status {
			case manifest.StatusKept:
				d.Clips.Kept++
			case manifest.StatusRejected:
				d.Clips.Rejected++
			default:
				d.Clips.Pending++
			}
			switch c.ProxyInfo.Source {
			case manifest.ProxyCamera:
				d.Clips.Camera++
			case manifest.ProxyGenerated:
				d.Clips.Generated++
			case manifest.ProxyNone:
				d.Clips.NoProxy++
			}
		}
	}
	writeJSON(w, d)
}

// known reports whether dir is one of the discovered projects (guards the
// opaque id against pointing anywhere on disk).
func (s *Server) known(dir string) bool {
	for _, p := range pipeline.FindProjects(s.roots) {
		if p == dir {
			return true
		}
	}
	return false
}

func encodeID(dir string) string {
	return base64.RawURLEncoding.EncodeToString([]byte(dir))
}

func decodeID(id string) (string, bool) {
	b, err := base64.RawURLEncoding.DecodeString(id)
	if err != nil {
		return "", false
	}
	return string(b), true
}

func writeJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(v)
}
