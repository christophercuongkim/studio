package dashboard

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

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
		os.MkdirAll(filepath.Join(dir, "originals"), 0o755)
		os.WriteFile(filepath.Join(dir, "originals", "a.mp4"), []byte("x"), 0o644)
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

func TestReviewMount(t *testing.T) {
	root := t.TempDir()
	dir := writeProject(t, root, "2026-08-02_b", true)
	ts := newTS(t, root)
	id := encodeID(dir)

	// The review API is mounted under the project prefix and returns the same
	// manifest the standalone serve UI would.
	res, err := http.Get(ts.URL + "/projects/" + id + "/review/api/manifest")
	if err != nil || res.StatusCode != 200 {
		t.Fatalf("review manifest: err=%v status=%v", err, res.StatusCode)
	}
	var man struct {
		Clips []struct {
			ID string `json:"id"`
		} `json:"clips"`
	}
	json.NewDecoder(res.Body).Decode(&man)
	if len(man.Clips) != 1 || man.Clips[0].ID != "c-001" {
		t.Fatalf("review manifest clips = %+v, want the one seeded clip", man.Clips)
	}

	// The review room itself serves the embedded serve frontend.
	res2, _ := http.Get(ts.URL + "/projects/" + id + "/review/")
	body, _ := io.ReadAll(res2.Body)
	if res2.StatusCode != 200 || !strings.Contains(string(body), "cliplist") {
		t.Fatalf("review root status=%d, body missing review UI", res2.StatusCode)
	}

	// An unknown project id is rejected, not mounted.
	res3, _ := http.Get(ts.URL + "/projects/" + encodeID(filepath.Join(root, "nope")) + "/review/api/manifest")
	if res3.StatusCode != 404 {
		t.Errorf("unknown review mount status = %d, want 404", res3.StatusCode)
	}
}

func TestThumbnailPicker(t *testing.T) {
	root := t.TempDir()
	dir := writeProject(t, root, "2026-08-02_b", true)
	// Seed two extracted candidates + a contact sheet (no ffmpeg needed).
	tdir := filepath.Join(dir, "thumbs")
	os.MkdirAll(tdir, 0o755)
	os.WriteFile(filepath.Join(tdir, "candidate-01-0m10s.png"), []byte("AAA"), 0o644)
	os.WriteFile(filepath.Join(tdir, "candidate-02-1m20s.png"), []byte("BBB"), 0o644)
	os.WriteFile(filepath.Join(tdir, "contact-sheet.png"), []byte("SHEET"), 0o644)
	ts := newTS(t, root)
	id := encodeID(dir)

	// Candidate list, nothing selected yet.
	var cands candidatesResp
	res, _ := http.Get(ts.URL + "/api/projects/" + id + "/thumbs/candidates")
	json.NewDecoder(res.Body).Decode(&cands)
	if len(cands.Candidates) != 2 || cands.ContactSheet != "contact-sheet.png" {
		t.Fatalf("candidates = %+v, want 2 + a contact sheet", cands)
	}
	if cands.Current != "" {
		t.Errorf("current = %q, want empty before any pick", cands.Current)
	}

	// Pick candidate 2.
	res2, _ := http.Post(ts.URL+"/api/projects/"+id+"/thumbnail", "application/json",
		strings.NewReader(`{"file":"candidate-02-1m20s.png"}`))
	if res2.StatusCode != 200 {
		t.Fatalf("set thumbnail status %d", res2.StatusCode)
	}
	// thumbnail.png now exists with the chosen bytes.
	got, err := os.ReadFile(filepath.Join(tdir, "thumbnail.png"))
	if err != nil || string(got) != "BBB" {
		t.Fatalf("thumbnail.png = %q err=%v, want the picked candidate's bytes", got, err)
	}
	// The list now reports it as current.
	var after candidatesResp
	res3, _ := http.Get(ts.URL + "/api/projects/" + id + "/thumbs/candidates")
	json.NewDecoder(res3.Body).Decode(&after)
	if after.Current != "candidate-02-1m20s.png" {
		t.Errorf("current = %q, want candidate-02", after.Current)
	}

	// Media serves a candidate; path traversal is rejected.
	res4, _ := http.Get(ts.URL + "/media/thumb/" + id + "/candidate-01-0m10s.png")
	body, _ := io.ReadAll(res4.Body)
	if res4.StatusCode != 200 || string(body) != "AAA" {
		t.Errorf("media serve status=%d body=%q", res4.StatusCode, body)
	}
	// A non-candidate file can't be set as the thumbnail.
	res5, _ := http.Post(ts.URL+"/api/projects/"+id+"/thumbnail", "application/json",
		strings.NewReader(`{"file":"../video.yaml"}`))
	if res5.StatusCode != 400 {
		t.Errorf("setting a non-candidate returned %d, want 400", res5.StatusCode)
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

func writeF(t *testing.T, dir, rel, content string) {
	t.Helper()
	p := filepath.Join(dir, rel)
	os.MkdirAll(filepath.Dir(p), 0o755)
	if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

// projectReadyToApply writes a project with one kept, named, not-yet-applied
// clip (plus its files on disk) — so run/apply has something to do.
func projectReadyToApply(t *testing.T, root string) string {
	t.Helper()
	dir := filepath.Join(root, "2026-08-01_x")
	os.MkdirAll(dir, 0o755)
	writeF(t, dir, "originals/a.MP4", "orig")
	writeF(t, dir, "proxy/a.mp4", "proxy")
	vy := videoyaml.Default("X", "27", "unlisted")
	videoyaml.Save(dir, &vy)
	created := time.Date(2026, 8, 1, 10, 0, 0, 0, time.UTC)
	man := &manifest.Manifest{
		SchemaVersion: manifest.SchemaVersion,
		Shoot:         manifest.Shoot{Root: dir, Title: "X", CamCode: "DJI", CreatedAt: created},
		Clips: []manifest.Clip{{
			ID: "c-001", Seq: 1, Stem: "a",
			Files:     manifest.Files{Original: "originals/a.MP4", Proxy: "proxy/a.mp4"},
			Media:     manifest.Media{CreatedAt: created, VCodec: "hevc"},
			ProxyInfo: manifest.ProxyInfo{Source: manifest.ProxyGenerated},
			Review:    manifest.Review{Status: manifest.StatusKept, Desc: "keeper", Rating: 5},
		}},
	}
	manifest.Save(dir, man)
	return dir
}

func firstID(t *testing.T, ts *httptest.Server) string {
	t.Helper()
	res, _ := http.Get(ts.URL + "/api/projects")
	var list []struct {
		ID string `json:"id"`
	}
	json.NewDecoder(res.Body).Decode(&list)
	if len(list) == 0 {
		t.Fatal("no projects")
	}
	return list[0].ID
}

func TestRunApplyAdvancesState(t *testing.T) {
	root := t.TempDir()
	projectReadyToApply(t, root)
	ts := newTS(t, root)
	id := firstID(t, ts)

	res, err := http.Post(ts.URL+"/api/projects/"+id+"/run/apply", "text/plain", nil)
	if err != nil || res.StatusCode != 200 {
		t.Fatalf("run apply: %v status %v", err, res.StatusCode)
	}
	body, _ := io.ReadAll(res.Body)
	if !strings.Contains(string(body), "applied") || !strings.Contains(string(body), "done") {
		t.Errorf("apply output unexpected:\n%s", body)
	}

	// The clip's original was renamed → next advances past apply.
	res2, _ := http.Get(ts.URL + "/api/projects/" + id)
	var d struct {
		Next string `json:"next"`
	}
	json.NewDecoder(res2.Body).Decode(&d)
	if d.Next != "scaffold" {
		t.Errorf("after apply, next = %q, want scaffold", d.Next)
	}
}

func TestRunApplyDryRunChangesNothing(t *testing.T) {
	root := t.TempDir()
	projectReadyToApply(t, root)
	ts := newTS(t, root)
	id := firstID(t, ts)

	res, _ := http.Post(ts.URL+"/api/projects/"+id+"/run/apply?dry=1", "text/plain", nil)
	body, _ := io.ReadAll(res.Body)
	if !strings.Contains(string(body), "→") { // the dry-run table uses arrows
		t.Errorf("dry-run should show the rename table:\n%s", body)
	}
	// Still at apply (nothing changed).
	res2, _ := http.Get(ts.URL + "/api/projects/" + id)
	var d struct {
		Next string `json:"next"`
	}
	json.NewDecoder(res2.Body).Decode(&d)
	if d.Next != "apply" {
		t.Errorf("dry-run advanced the state to %q", d.Next)
	}
}

func TestRunGuards(t *testing.T) {
	root := t.TempDir()
	projectReadyToApply(t, root)
	ts := newTS(t, root)
	id := firstID(t, ts)

	// A non-runnable step → 400.
	if res, _ := http.Post(ts.URL+"/api/projects/"+id+"/run/ingest", "text/plain", nil); res.StatusCode != 400 {
		t.Errorf("run/ingest status = %d, want 400", res.StatusCode)
	}
	// Unknown project → 404.
	if res, _ := http.Post(ts.URL+"/api/projects/"+encodeID("/etc")+"/run/apply", "text/plain", nil); res.StatusCode != 404 {
		t.Errorf("unknown project run status = %d, want 404", res.StatusCode)
	}
}

// tempConfig points config.Load at an isolated config whose projectsRoot is
// projectsRoot, so create tests don't touch the real ~/.config or ~/Videos.
func tempConfig(t *testing.T, projectsRoot string) {
	t.Helper()
	cfgHome := t.TempDir()
	os.MkdirAll(filepath.Join(cfgHome, "studio"), 0o755)
	os.WriteFile(filepath.Join(cfgHome, "studio", "config.yaml"),
		[]byte("projectsRoot: "+projectsRoot+"\ncamCode: DJI\n"), 0o644)
	t.Setenv("XDG_CONFIG_HOME", cfgHome)
}

func TestCreateProject(t *testing.T) {
	root := t.TempDir()
	tempConfig(t, root)
	ts := newTS(t, root)

	res, err := http.Post(ts.URL+"/api/projects", "application/json",
		strings.NewReader(`{"slug":"lake-trip","title":"Lake","date":"2026-08-01"}`))
	if err != nil || res.StatusCode != 200 {
		t.Fatalf("create: %v status %v", err, res.StatusCode)
	}
	var out struct{ ID string }
	json.NewDecoder(res.Body).Decode(&out)
	if out.ID == "" {
		t.Fatal("no id returned")
	}
	if _, err := os.Stat(filepath.Join(root, "2026-08-01_lake-trip", "video.yaml")); err != nil {
		t.Errorf("project not created: %v", err)
	}

	// Invalid slug → 400.
	if res, _ := http.Post(ts.URL+"/api/projects", "application/json",
		strings.NewReader(`{"slug":"Bad Slug"}`)); res.StatusCode != 400 {
		t.Errorf("bad slug status = %d, want 400", res.StatusCode)
	}
}

func TestDrivesEndpoint(t *testing.T) {
	ts := newTS(t, t.TempDir())
	res, err := http.Get(ts.URL + "/api/drives")
	if err != nil || res.StatusCode != 200 {
		t.Fatalf("drives: %v status %v", err, res.StatusCode)
	}
	body, _ := io.ReadAll(res.Body)
	// Must be [] (not null) so the client can always .map over it.
	if !strings.HasPrefix(strings.TrimSpace(string(body)), "[") {
		t.Errorf("drives should be a JSON array, got %q", body)
	}
}

func TestIngestEndpoint(t *testing.T) {
	if testing.Short() {
		t.Skip("skips ffmpeg ingest in -short mode")
	}
	if _, err := exec.LookPath("ffmpeg"); err != nil {
		t.Skip("ffmpeg not on PATH")
	}
	root := t.TempDir()
	tempConfig(t, root)

	// Create a project, then a card to ingest from.
	dir := filepath.Join(root, "2026-08-01_c")
	os.MkdirAll(dir, 0o755)
	vy := videoyaml.Default("C", "27", "unlisted")
	videoyaml.Save(dir, &vy)

	card := t.TempDir()
	gen := exec.Command("ffmpeg", "-y", "-f", "lavfi", "-i", "testsrc2=s=320x180:r=30:d=1",
		"-f", "lavfi", "-i", "sine=frequency=440:duration=1",
		"-c:v", "libx264", "-pix_fmt", "yuv420p", "-c:a", "aac", "-shortest",
		"-f", "mp4", filepath.Join(card, "DJI_0001.MP4"))
	if out, err := gen.CombinedOutput(); err != nil {
		t.Fatalf("gen: %v\n%s", err, out)
	}

	ts := newTS(t, root)
	id := encodeID(dir)
	res, err := http.Post(ts.URL+"/api/projects/"+id+"/ingest", "application/json",
		strings.NewReader(`{"source":"`+card+`","copy":true}`))
	if err != nil || res.StatusCode != 200 {
		t.Fatalf("ingest: %v status %v", err, res.StatusCode)
	}
	body, _ := io.ReadAll(res.Body)
	if !strings.Contains(string(body), "ingested 1 clip") {
		t.Errorf("ingest output unexpected:\n%s", body)
	}
	// Now ingested → next is review.
	res2, _ := http.Get(ts.URL + "/api/projects/" + id)
	var d struct{ Next string }
	json.NewDecoder(res2.Body).Decode(&d)
	if d.Next != "review" {
		t.Errorf("after ingest, next = %q, want review", d.Next)
	}
}
