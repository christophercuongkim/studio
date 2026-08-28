package dashboard

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/christophercuongkim/studio/internal/upload"
	"github.com/christophercuongkim/studio/internal/videoyaml"
)

// uploadPreview is the publish panel's payload: the dry-run text, the inline
// validation problems, the QC-gate status, and whether it's already uploaded —
// plus the current editable metadata so the form can populate itself.
type uploadPreview struct {
	Payload  string    `json:"payload"`
	Problems []string  `json:"problems"`
	QCGate   qcGate    `json:"qcGate"`
	Uploaded bool      `json:"uploaded"`
	VideoID  string    `json:"videoId"`
	Video    videoEdit `json:"video"`
}

type qcGate struct {
	OK     bool   `json:"ok"`
	Detail string `json:"detail"`
}

type videoEdit struct {
	Title       string   `json:"title"`
	Tags        []string `json:"tags"`
	Privacy     string   `json:"privacy"`
	Description string   `json:"description"`
}

// handleUploadPreview returns the dry-run payload + validation + QC gate. No
// network — always safe to call.
func (s *Server) handleUploadPreview(w http.ResponseWriter, r *http.Request) {
	dir, ok := decodeID(r.PathValue("id"))
	if !ok || !s.known(dir) {
		http.Error(w, "unknown project", http.StatusNotFound)
		return
	}
	vy, err := videoyaml.Load(dir)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	writeJSON(w, buildPreview(vy, dir))
}

// handleVideoPatch edits the video.yaml fields the UI exposes, then returns the
// refreshed preview so validation updates inline. Edits are saved even when
// still invalid — upload is gated separately, so this is just editing a draft.
func (s *Server) handleVideoPatch(w http.ResponseWriter, r *http.Request) {
	dir, ok := decodeID(r.PathValue("id"))
	if !ok || !s.known(dir) {
		http.Error(w, "unknown project", http.StatusNotFound)
		return
	}
	vy, err := videoyaml.Load(dir)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	var req videoEdit
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid body", http.StatusBadRequest)
		return
	}
	vy.Title = req.Title
	vy.Description = req.Description
	vy.Privacy = req.Privacy
	vy.Tags = req.Tags
	if vy.Tags == nil {
		vy.Tags = []string{}
	}
	if err := videoyaml.Save(dir, vy); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(w, buildPreview(vy, dir))
}

// handleUpload runs the real upload, streaming progress. Body: {skipQC, update}.
// On first-time auth the OAuth URL is streamed to the browser (not just stdout).
func (s *Server) handleUpload(w http.ResponseWriter, r *http.Request) {
	dir, ok := decodeID(r.PathValue("id"))
	if !ok || !s.known(dir) {
		http.Error(w, "unknown project", http.StatusNotFound)
		return
	}
	var req struct {
		SkipQC bool `json:"skipQC"`
		Update bool `json:"update"`
	}
	json.NewDecoder(r.Body).Decode(&req)

	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	flusher, _ := w.(http.Flusher)
	emit := func(line string) {
		fmt.Fprintln(w, line)
		if flusher != nil {
			flusher.Flush()
		}
	}

	verb := "uploading"
	if req.Update {
		verb = "updating metadata for"
	}
	emit("▶ " + verb + " this video …")

	res, err := upload.Run(context.Background(), upload.Options{
		ProjectDir: dir,
		SkipQC:     req.SkipQC,
		Update:     req.Update,
		AuthPrompt: func(url string) {
			emit("")
			emit("First-time authorization needed. Open this URL, approve, then return here:")
			emit("  " + url)
			emit("(waiting for authorization …)")
		},
	})
	if err != nil {
		emit("ERROR: " + err.Error())
		return
	}
	if res.Updated {
		emit("✓ metadata updated for " + res.VideoID)
	} else {
		emit("✓ uploaded")
	}
	emit("  https://youtu.be/" + res.VideoID)
}

// buildPreview assembles the publish-panel payload for vy.
func buildPreview(vy *videoyaml.VideoYAML, dir string) uploadPreview {
	p := uploadPreview{
		Payload:  upload.DryRun(vy, ""),
		Problems: upload.Problems(vy, dir),
		Video: videoEdit{
			Title: vy.Title, Tags: vy.Tags, Privacy: vy.Privacy, Description: vy.Description,
		},
	}
	if p.Problems == nil {
		p.Problems = []string{}
	}
	if p.Video.Tags == nil {
		p.Video.Tags = []string{}
	}
	if err := upload.CheckQCGate(dir, false); err != nil {
		p.QCGate = qcGate{OK: false, Detail: err.Error()}
	} else {
		p.QCGate = qcGate{OK: true, Detail: "qc-report.json passing"}
	}
	if vy.YouTube.VideoID != nil {
		p.Uploaded = true
		p.VideoID = *vy.YouTube.VideoID
	}
	return p
}
