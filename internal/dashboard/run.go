package dashboard

import (
	"fmt"
	"net/http"

	"github.com/christophercuongkim/studio/internal/config"
	"github.com/christophercuongkim/studio/internal/steps"
)

// runnable are the steps the dashboard can execute (slice 2). ingest, review,
// and upload have their own flows; new isn't a step.
var runnable = map[string]bool{
	"apply": true, "scaffold": true, "chapters": true, "qc": true, "thumbs": true, "archive": true,
}

// handleRun executes a pipeline step and streams its output lines as plain text
// (chunked, flushed per line). ?dry=1 previews apply/archive; ?force=1 lets
// archive proceed on an unpublished video.
func (s *Server) handleRun(w http.ResponseWriter, r *http.Request) {
	dir, ok := decodeID(r.PathValue("id"))
	if !ok || !s.known(dir) {
		http.Error(w, "unknown project", http.StatusNotFound)
		return
	}
	step := r.PathValue("step")
	if !runnable[step] {
		http.Error(w, "step not runnable from the dashboard", http.StatusBadRequest)
		return
	}
	dry := r.URL.Query().Get("dry") == "1"
	force := r.URL.Query().Get("force") == "1"

	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	flusher, _ := w.(http.Flusher)
	emit := func(line string) {
		fmt.Fprintln(w, line)
		if flusher != nil {
			flusher.Flush()
		}
	}

	label := step
	if dry {
		label += " (dry run)"
	}
	emit("▶ running " + label + " …")

	res, err := runStep(dir, step, dry, force)
	if res != nil {
		for _, l := range res.Lines {
			emit(l)
		}
	}
	switch {
	case err != nil:
		emit("ERROR: " + err.Error())
	case res != nil && res.Failed:
		emit("✗ " + step + " reported failures")
	default:
		emit("✓ " + step + " done")
	}
}

// runStep dispatches to internal/steps — the same code the CLI runs.
func runStep(dir, step string, dry, force bool) (*steps.Result, error) {
	switch step {
	case "apply":
		return steps.Apply(dir, dry)
	case "scaffold":
		cfg, err := config.Load()
		if err != nil {
			return nil, err
		}
		return steps.Scaffold(dir, cfg.KdenliveTemplate, "")
	case "chapters":
		return steps.Chapters(dir, "", true, 0) // the dashboard writes chapters into video.yaml
	case "qc":
		return steps.QC(dir, "", "")
	case "thumbs":
		return steps.Thumbs(dir, "render", 12)
	case "archive":
		cfg, err := config.Load()
		if err != nil {
			return nil, err
		}
		return steps.Archive(dir, cfg.ArchiveRoot, dry, false, force)
	}
	return nil, fmt.Errorf("unknown step %q", step)
}
