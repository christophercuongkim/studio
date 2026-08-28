// Package promptsrv is the `studio prompt` recording teleprompter (plan §10.2).
// State lives server-side behind a mutex so several devices (laptop + iPad by
// the camera) stay in sync by polling GET /api/state. It is the one studio
// server that defaults to a non-loopback bind.
package promptsrv

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/christophercuongkim/studio/internal/script"
	"github.com/christophercuongkim/studio/internal/webui"
)

// State is the shared prompter position, polled by every connected device.
type State struct {
	Section      int  `json:"section"`
	Bullet       int  `json:"bullet"`
	Take         int  `json:"take"`
	Blackout     bool `json:"blackout"`
	SectionCount int  `json:"sectionCount"`
}

// Server serves the prompter UI and drives shared state.
type Server struct {
	script  *script.Script
	logPath string // "" disables logging
	now     func() time.Time

	mu sync.Mutex
	st State
}

// New builds a prompter server for a parsed script. logPath enables the
// append-only event log when non-empty.
func New(s *script.Script, logPath string) *Server {
	return &Server{
		script:  s,
		logPath: logPath,
		now:     func() time.Time { return time.Now() },
		st:      State{SectionCount: len(s.Sections)},
	}
}

// Handler returns the HTTP routes.
func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/script", s.handleScript)
	mux.HandleFunc("GET /api/state", s.handleState)
	mux.HandleFunc("POST /api/action", s.handleAction)
	mux.Handle("GET /", http.FileServer(http.FS(webui.PromptFS())))
	return mux
}

func (s *Server) handleScript(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, map[string]any{
		"title":    s.script.Title,
		"sections": s.script.Sections,
	})
}

func (s *Server) handleState(w http.ResponseWriter, r *http.Request) {
	s.mu.Lock()
	defer s.mu.Unlock()
	writeJSON(w, s.st)
}

func (s *Server) handleAction(w http.ResponseWriter, r *http.Request) {
	action := r.URL.Query().Get("a")
	s.mu.Lock()
	event := s.apply(action)
	st := s.st
	s.mu.Unlock()

	if event != "" {
		s.log(event, st)
	}
	writeJSON(w, st)
}

// apply mutates state for an action and returns the log event type ("bullet",
// "section", "take") or "" for no-op/blackout.
func (s *Server) apply(action string) string {
	switch action {
	case "next":
		return s.advance()
	case "back":
		return s.retreat()
	case "nextSection":
		if s.st.Section+1 < len(s.script.Sections) {
			s.st.Section++
			s.st.Bullet = 0
			return "section"
		}
	case "prevSection":
		if s.st.Section > 0 {
			s.st.Section--
			s.st.Bullet = 0
			return "section"
		}
	case "take":
		s.st.Take++
		s.st.Bullet = 0
		return "take"
	case "blackout":
		s.st.Blackout = !s.st.Blackout
	}
	return ""
}

func (s *Server) advance() string {
	bullets := len(s.script.Sections[s.st.Section].Bullets)
	if s.st.Bullet+1 < bullets {
		s.st.Bullet++
		return "bullet"
	}
	if s.st.Section+1 < len(s.script.Sections) {
		s.st.Section++
		s.st.Bullet = 0
		return "section"
	}
	return "" // already at the end
}

func (s *Server) retreat() string {
	if s.st.Bullet > 0 {
		s.st.Bullet--
		return "bullet"
	}
	if s.st.Section > 0 {
		s.st.Section--
		s.st.Bullet = maxBullet(len(s.script.Sections[s.st.Section].Bullets))
		return "section"
	}
	return ""
}

func maxBullet(n int) int {
	if n == 0 {
		return 0
	}
	return n - 1
}

// logEvent is one line in prompt-log.jsonl.
type logEvent struct {
	TS      string `json:"ts"`
	Event   string `json:"event"`
	Section string `json:"section"`
	Take    int    `json:"take"`
}

func (s *Server) log(event string, st State) {
	if s.logPath == "" {
		return
	}
	sectionTitle := ""
	if st.Section < len(s.script.Sections) {
		sectionTitle = s.script.Sections[st.Section].Title
	}
	ev := logEvent{
		TS:      s.now().Format(time.RFC3339),
		Event:   event,
		Section: sectionTitle,
		Take:    st.Take,
	}
	line, err := json.Marshal(ev)
	if err != nil {
		return
	}
	f, err := os.OpenFile(s.logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		return
	}
	defer f.Close()
	f.Write(append(line, '\n'))
}

func writeJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(v)
}

// InterfaceURLs returns http URLs for every non-loopback IPv4 interface, so the
// startup message can point second devices at the right address.
func InterfaceURLs(port string) []string {
	var urls []string
	addrs, err := netInterfaceAddrs()
	if err != nil {
		return urls
	}
	for _, ip := range addrs {
		urls = append(urls, fmt.Sprintf("http://%s:%s", ip, port))
	}
	return urls
}
