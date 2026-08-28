package server

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/christophercuongkim/studio/internal/manifest"
)

// buildProject writes a minimal valid project (manifest + referenced files) and
// returns its dir.
func buildProject(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "originals", "clip.mp4"), "orig")
	writeFile(t, filepath.Join(dir, "proxy", "clip.mp4"), "proxydata")
	writeFile(t, filepath.Join(dir, "originals", "solo.mov"), "orig2")

	created := time.Date(2026, 8, 27, 14, 23, 1, 0, time.UTC)
	man := &manifest.Manifest{
		SchemaVersion: manifest.SchemaVersion,
		Shoot:         manifest.Shoot{Root: dir, Title: "lake-trip", CreatedAt: created, CamCode: "DJI"},
		Clips: []manifest.Clip{
			{
				ID: "c-001", Seq: 1, Stem: "clip",
				Files:     manifest.Files{Original: "originals/clip.mp4", Proxy: "proxy/clip.mp4"},
				Media:     manifest.Media{DurationSec: 42, Width: 3840, Height: 2160, FPS: "29.97", VCodec: "hevc", HasAudio: true, CreatedAt: created},
				ProxyInfo: manifest.ProxyInfo{Source: manifest.ProxyCamera},
				Review:    manifest.Review{Status: manifest.StatusPending},
			},
			{
				ID: "c-002", Seq: 2, Stem: "solo",
				Files:     manifest.Files{Original: "originals/solo.mov"},
				Media:     manifest.Media{DurationSec: 10, Width: 1920, Height: 1080, FPS: "30", VCodec: "h264", CreatedAt: created},
				ProxyInfo: manifest.ProxyInfo{Source: manifest.ProxyNone, Note: "generation failed"},
				Review:    manifest.Review{Status: manifest.StatusPending},
			},
		},
	}
	if err := manifest.Save(dir, man); err != nil {
		t.Fatal(err)
	}
	return dir
}

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func newTestServer(t *testing.T, dir string) *httptest.Server {
	t.Helper()
	s, err := New(dir)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { s.Close() })
	return httptest.NewServer(s.Handler())
}

func TestManifestEndpoint(t *testing.T) {
	ts := newTestServer(t, buildProject(t))
	defer ts.Close()

	res, err := http.Get(ts.URL + "/api/manifest")
	if err != nil || res.StatusCode != 200 {
		t.Fatalf("GET manifest: %v status %v", err, res.StatusCode)
	}
	var man manifest.Manifest
	json.NewDecoder(res.Body).Decode(&man)
	if len(man.Clips) != 2 {
		t.Errorf("got %d clips, want 2", len(man.Clips))
	}
}

func TestPatchValidatesAndPersists(t *testing.T) {
	dir := buildProject(t)
	ts := newTestServer(t, dir)
	defer ts.Close()

	// Valid patch: rating + desc + status.
	clip := patchClip(t, ts, "c-001", `{"rating":5,"desc":"sunset-dock-wide","status":"kept"}`, 200)
	if clip.Review.Rating != 5 || clip.Review.Desc != "sunset-dock-wide" || clip.Review.Status != manifest.StatusKept {
		t.Errorf("patch not applied: %+v", clip.Review)
	}
	if clip.Review.ReviewedAt == nil {
		t.Error("reviewedAt not set")
	}

	// Invalid desc charset → 400.
	patchClip(t, ts, "c-001", `{"desc":"Has Caps"}`, 400)
	// Invalid rating → 400.
	patchClip(t, ts, "c-001", `{"rating":9}`, 400)
	// Invalid status → 400.
	patchClip(t, ts, "c-001", `{"status":"maybe"}`, 400)
	// Unknown clip → 404.
	patchClip(t, ts, "c-999", `{"rating":1}`, 404)
}

func TestTakeClearWithNull(t *testing.T) {
	ts := newTestServer(t, buildProject(t))
	defer ts.Close()

	c := patchClip(t, ts, "c-001", `{"take":2}`, 200)
	if c.Review.Take == nil || *c.Review.Take != 2 {
		t.Fatalf("take not set: %+v", c.Review.Take)
	}
	c = patchClip(t, ts, "c-001", `{"take":null}`, 200)
	if c.Review.Take != nil {
		t.Errorf("take should be cleared, got %v", *c.Review.Take)
	}
}

func TestPreviewName(t *testing.T) {
	ts := newTestServer(t, buildProject(t))
	defer ts.Close()

	patchClip(t, ts, "c-001", `{"desc":"sunset-dock-wide"}`, 200)
	res, _ := http.Get(ts.URL + "/api/preview-name/c-001")
	var body map[string]string
	json.NewDecoder(res.Body).Decode(&body)
	if body["name"] != "20260827_DJI001_sunset-dock-wide" {
		t.Errorf("preview name = %q", body["name"])
	}
}

func TestProxyServingAndAbsence(t *testing.T) {
	ts := newTestServer(t, buildProject(t))
	defer ts.Close()

	res, err := http.Get(ts.URL + "/media/proxy/c-001")
	if err != nil || res.StatusCode != 200 {
		t.Fatalf("proxy GET: %v status %v", err, res.StatusCode)
	}
	b, _ := io.ReadAll(res.Body)
	if string(b) != "proxydata" {
		t.Errorf("proxy body = %q", b)
	}

	// c-002 has no proxy → 404.
	res2, _ := http.Get(ts.URL + "/media/proxy/c-002")
	if res2.StatusCode != 404 {
		t.Errorf("no-proxy clip status = %d, want 404", res2.StatusCode)
	}
}

// TestShutdownSavesEdits proves a PATCH survives to disk via Close (the
// save-on-shutdown path), independent of the debounce timer.
func TestShutdownSavesEdits(t *testing.T) {
	dir := buildProject(t)
	s, err := New(dir)
	if err != nil {
		t.Fatal(err)
	}
	ts := httptest.NewServer(s.Handler())
	defer ts.Close()

	patchClip(t, ts, "c-001", `{"rating":4,"status":"kept"}`, 200)
	if err := s.Close(); err != nil { // flush
		t.Fatal(err)
	}

	// Reload from disk: the edit must be there.
	reloaded, err := manifest.Load(dir)
	if err != nil {
		t.Fatal(err)
	}
	c := reloaded.ClipByID("c-001")
	if c.Review.Rating != 4 || c.Review.Status != manifest.StatusKept {
		t.Errorf("edit not persisted to disk: %+v", c.Review)
	}
}

func TestIndexServed(t *testing.T) {
	ts := newTestServer(t, buildProject(t))
	defer ts.Close()
	res, _ := http.Get(ts.URL + "/")
	b, _ := io.ReadAll(res.Body)
	if res.StatusCode != 200 || !strings.Contains(string(b), "studio — review") {
		t.Errorf("index not served (status %d)", res.StatusCode)
	}
}

// --- helper ---

func patchClip(t *testing.T, ts *httptest.Server, id, body string, wantStatus int) manifest.Clip {
	t.Helper()
	req, _ := http.NewRequest("PATCH", ts.URL+"/api/clips/"+id, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("PATCH %s: %v", id, err)
	}
	defer res.Body.Close()
	if res.StatusCode != wantStatus {
		msg, _ := io.ReadAll(res.Body)
		t.Fatalf("PATCH %s status = %d, want %d (%s)", id, res.StatusCode, wantStatus, msg)
	}
	var clip manifest.Clip
	if res.StatusCode == 200 {
		json.NewDecoder(res.Body).Decode(&clip)
	}
	return clip
}
