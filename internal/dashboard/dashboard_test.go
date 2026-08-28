package dashboard

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/christophercuongkim/studio/internal/manifest"
	"github.com/christophercuongkim/studio/internal/videoyaml"
)

func writeProject(t *testing.T, root, name string, withClips bool) string {
	t.Helper()
	dir := filepath.Join(root, name)
	os.MkdirAll(dir, 0o755)
	vy := videoyaml.Default(name+" title", "27", "unlisted")
	if err := videoyaml.Save(dir, &vy); err != nil {
		t.Fatal(err)
	}
	if withClips {
		man := &manifest.Manifest{
			SchemaVersion: manifest.SchemaVersion,
			Shoot:         manifest.Shoot{Root: dir, Title: name, CamCode: "DJI"},
			Clips: []manifest.Clip{
				{ID: "c-001", Seq: 1, Stem: "a",
					Files:     manifest.Files{Original: "originals/a.mp4"},
					ProxyInfo: manifest.ProxyInfo{Source: manifest.ProxyCamera},
					Review:    manifest.Review{Status: manifest.StatusKept, Desc: "x"},
					Applied:   manifest.Applied{Done: true, FinalStem: "s"}},
			},
		}
		if err := manifest.Save(dir, man); err != nil {
			t.Fatal(err)
		}
	}
	return dir
}

func newTS(t *testing.T, root string) *httptest.Server {
	t.Helper()
	ts := httptest.NewServer(New([]string{root}).Handler())
	t.Cleanup(ts.Close)
	return ts
}

func TestProjectsList(t *testing.T) {
	root := t.TempDir()
	writeProject(t, root, "2026-08-01_a", false) // fresh → next ingest
	writeProject(t, root, "2026-08-02_b", true)  // ingested+applied → next scaffold
	ts := newTS(t, root)

	res, _ := http.Get(ts.URL + "/api/projects")
	var got []struct {
		ID    string `json:"id"`
		Title string `json:"title"`
		Next  string `json:"next"`
	}
	json.NewDecoder(res.Body).Decode(&got)
	if len(got) != 2 {
		t.Fatalf("got %d projects, want 2", len(got))
	}
	byTitle := map[string]string{} // title → next
	for _, p := range got {
		if p.ID == "" {
			t.Error("empty project id")
		}
		byTitle[p.Title] = p.Next
	}
	if byTitle["2026-08-01_a title"] != "ingest" {
		t.Errorf("fresh project next = %q, want ingest", byTitle["2026-08-01_a title"])
	}
	if byTitle["2026-08-02_b"] != "scaffold" {
		t.Errorf("applied project next = %q, want scaffold", byTitle["2026-08-02_b"])
	}
}

func TestProjectDetail(t *testing.T) {
	root := t.TempDir()
	writeProject(t, root, "2026-08-02_b", true)
	ts := newTS(t, root)

	// Get the id from the list.
	res, _ := http.Get(ts.URL + "/api/projects")
	var list []struct {
		ID string `json:"id"`
	}
	json.NewDecoder(res.Body).Decode(&list)
	id := list[0].ID

	res2, _ := http.Get(ts.URL + "/api/projects/" + id)
	if res2.StatusCode != 200 {
		t.Fatalf("detail status %d", res2.StatusCode)
	}
	var d struct {
		Video struct {
			Privacy string `json:"privacy"`
		} `json:"video"`
		Clips struct {
			Total  int `json:"total"`
			Kept   int `json:"kept"`
			Camera int `json:"camera"`
		} `json:"clips"`
		Steps []struct {
			Name string `json:"name"`
		} `json:"steps"`
	}
	json.NewDecoder(res2.Body).Decode(&d)
	if d.Video.Privacy != "unlisted" {
		t.Errorf("video privacy = %q", d.Video.Privacy)
	}
	if d.Clips.Total != 1 || d.Clips.Kept != 1 || d.Clips.Camera != 1 {
		t.Errorf("clip meta wrong: %+v", d.Clips)
	}
	if len(d.Steps) != 10 {
		t.Errorf("want 10 steps, got %d", len(d.Steps))
	}
}

func TestDetailRejectsUnknownID(t *testing.T) {
	root := t.TempDir()
	writeProject(t, root, "2026-08-02_b", true)
	ts := newTS(t, root)

	// An id pointing outside the known projects must 404 (not read arbitrary dirs).
	bad := encodeID("/etc")
	res, _ := http.Get(ts.URL + "/api/projects/" + bad)
	if res.StatusCode != 404 {
		t.Errorf("unknown id status = %d, want 404", res.StatusCode)
	}
	if res2, _ := http.Get(ts.URL + "/api/projects/not-base64!!"); res2.StatusCode != 404 {
		t.Errorf("garbage id status = %d, want 404", res2.StatusCode)
	}
}

func TestIndexServed(t *testing.T) {
	ts := newTS(t, t.TempDir())
	res, _ := http.Get(ts.URL + "/")
	b, _ := io.ReadAll(res.Body)
	if res.StatusCode != 200 || !strings.Contains(string(b), "studio — dashboard") {
		t.Errorf("index not served (status %d)", res.StatusCode)
	}
}
