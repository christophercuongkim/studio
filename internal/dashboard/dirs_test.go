package dashboard

import (
	"os"
	"path/filepath"
	"testing"
)

// TestWithinRoot covers the containment guard: only paths at or below a root are
// allowed, and a lexical "../" escape is rejected once the caller has Clean'd.
func TestWithinRoot(t *testing.T) {
	roots := []string{"/media/SSD", "/home/me/videos"}
	cases := []struct {
		path string
		want bool
	}{
		{"/media/SSD", true},
		{"/media/SSD/DCIM/100MEDIA", true},
		{"/home/me/videos/2026", true},
		{"/media/SSDX", false},     // prefix but not a path child
		{"/media", false},          // parent of a root is not itself allowed
		{"/etc/passwd", false},     // wholly outside
		{"/home/me/secret", false}, // sibling of a root
	}
	for _, c := range cases {
		if _, ok := withinRoot(filepath.Clean(c.path), roots); ok != c.want {
			t.Errorf("withinRoot(%q) = %v, want %v", c.path, ok, c.want)
		}
	}
}

// TestHandleDirsListsAndClampsParent drives the endpoint against a real tree:
// it lists only subdirectories (not files), sorts them, and offers no parent at
// the root boundary but a valid one a level down.
func TestHandleDirsListsAndClampsParent(t *testing.T) {
	root := t.TempDir()
	must(t, os.MkdirAll(filepath.Join(root, "DCIM", "100MEDIA"), 0o755))
	must(t, os.MkdirAll(filepath.Join(root, "Music"), 0o755))
	must(t, os.MkdirAll(filepath.Join(root, ".Spotlight-V100"), 0o755)) // dot-dir: hidden
	must(t, os.WriteFile(filepath.Join(root, "readme.txt"), []byte("x"), 0o644))

	roots := []string{root}

	// At the root: two dirs (sorted), no file, no parent.
	got, ok, err := browseListing(root, roots)
	if !ok || err != nil {
		t.Fatalf("browseListing(root) ok=%v err=%v", ok, err)
	}
	if got.Parent != "" {
		t.Errorf("parent at root = %q, want empty", got.Parent)
	}
	if len(got.Dirs) != 2 || got.Dirs[0].Name != "DCIM" || got.Dirs[1].Name != "Music" {
		t.Fatalf("dirs = %+v, want [DCIM Music]", got.Dirs)
	}

	// One level down: parent points back up and stays inside the root.
	sub := filepath.Join(root, "DCIM")
	gotSub, ok, err := browseListing(sub, roots)
	if !ok || err != nil {
		t.Fatalf("browseListing(sub) ok=%v err=%v", ok, err)
	}
	if gotSub.Parent != root {
		t.Errorf("parent of %q = %q, want %q", sub, gotSub.Parent, root)
	}
	if len(gotSub.Dirs) != 1 || gotSub.Dirs[0].Name != "100MEDIA" {
		t.Errorf("dirs = %+v, want [100MEDIA]", gotSub.Dirs)
	}
}

func must(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatal(err)
	}
}
