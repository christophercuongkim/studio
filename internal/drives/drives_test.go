package drives

import (
	"os"
	"os/user"
	"path/filepath"
	"strings"
	"testing"
)

func TestScan(t *testing.T) {
	base := t.TempDir()
	// Two "volumes" and a stray file that must be ignored.
	os.MkdirAll(filepath.Join(base, "SANDISK64"), 0o755)
	os.MkdirAll(filepath.Join(base, "MySSD"), 0o755)
	os.WriteFile(filepath.Join(base, "notadir"), []byte("x"), 0o644)

	got := scan([]string{base, "/nonexistent/base"})
	if len(got) != 2 {
		t.Fatalf("got %d drives, want 2: %+v", len(got), got)
	}
	// Sorted by label: MySSD before SANDISK64.
	if got[0].Label != "MySSD" || got[1].Label != "SANDISK64" {
		t.Errorf("not sorted by label: %+v", got)
	}
	if got[0].Path != filepath.Join(base, "MySSD") {
		t.Errorf("path wrong: %s", got[0].Path)
	}
}

// TestDropContainers covers the NixOS/bare-udisks case: scanning the bare
// /media root turns up both a real volume (/media/Extreme SSD) and the per-user
// container dir (/media/chriskim); the container must be dropped, the volume kept.
func TestDropContainers(t *testing.T) {
	ds := []Drive{
		{Label: "Extreme SSD", Path: "/media/Extreme SSD"},
		{Label: "chriskim", Path: "/media/chriskim"}, // the per-user container
		{Label: "SDCARD", Path: "/media/chriskim/SDCARD"},
	}
	got := dropContainers(ds, []string{"/run/media/chriskim", "/media/chriskim"})
	if len(got) != 2 {
		t.Fatalf("got %d drives, want 2: %+v", len(got), got)
	}
	for _, d := range got {
		if d.Path == "/media/chriskim" {
			t.Errorf("per-user container dir not dropped: %+v", got)
		}
	}
}

func TestPrompt(t *testing.T) {
	ds := []Drive{{Label: "A", Path: "/run/media/me/A"}, {Label: "B", Path: "/run/media/me/B"}}

	// Explicit choice.
	if d, err := Prompt(io_discard{}, strings.NewReader("2\n"), ds); err != nil || d.Label != "B" {
		t.Errorf("choice 2 = %+v, %v", d, err)
	}
	// Empty input → first.
	if d, err := Prompt(io_discard{}, strings.NewReader("\n"), ds); err != nil || d.Label != "A" {
		t.Errorf("empty → %+v, %v", d, err)
	}
	// Out of range.
	if _, err := Prompt(io_discard{}, strings.NewReader("9\n"), ds); err == nil {
		t.Error("out-of-range choice should error")
	}
	// No drives.
	if _, err := Prompt(io_discard{}, strings.NewReader("1\n"), nil); err == nil {
		t.Error("no drives should error")
	}
}

func TestIsRemovable(t *testing.T) {
	u, err := user.Current()
	if err != nil {
		t.Skip("no current user")
	}
	removable := filepath.Join("/run/media", u.Username, "SDCARD", "DCIM")
	if !IsRemovable(removable) {
		t.Errorf("%s should be removable", removable)
	}
	if IsRemovable("/home/" + u.Username + "/videos") {
		t.Error("home path should not be removable")
	}
}

// io_discard is a tiny io.Writer sink (avoids importing io just for Discard).
type io_discard struct{}

func (io_discard) Write(p []byte) (int, error) { return len(p), nil }
