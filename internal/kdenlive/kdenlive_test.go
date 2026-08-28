package kdenlive

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func loadTemplate(t *testing.T) *Node {
	t.Helper()
	data, err := os.ReadFile(filepath.Join("testdata", "empty-template.kdenlive"))
	if err != nil {
		t.Fatal(err)
	}
	root, err := Load(data)
	if err != nil {
		t.Fatal(err)
	}
	return root
}

func TestLoadRejectsNonMLT(t *testing.T) {
	_, err := Load([]byte(`<?xml version="1.0"?><project><clip/></project>`))
	if err == nil || !strings.Contains(err.Error(), "expected <mlt>") {
		t.Fatalf("expected fail-loud on non-mlt root, got %v", err)
	}
}

func TestRoundTripStable(t *testing.T) {
	root := loadTemplate(t)
	rendered := root.Render()
	reparsed, err := Load(rendered)
	if err != nil {
		t.Fatalf("re-parse rendered output: %v", err)
	}
	// Structural equality: same producer count and main_bin entries survive.
	if got := countTag(reparsed, "producer"); got != countTag(root, "producer") {
		t.Errorf("producer count changed across round-trip: %d vs %d", got, countTag(root, "producer"))
	}
	if got := countTag(reparsed, "entry"); got != countTag(root, "entry") {
		t.Errorf("entry count changed across round-trip")
	}
}

func TestScaffoldInjectsAndRoutesByRating(t *testing.T) {
	root := loadTemplate(t)
	beforeProducers := countTag(root, "producer")

	clips := []ClipRef{
		{Resource: "/abs/originals/hero.mp4", DurationSec: 42.36, Rating: 5}, // → Selects
		{Resource: "/abs/originals/b.mp4", DurationSec: 3.0, Rating: 2},      // → A-Cam
	}
	if err := Scaffold(root, clips); err != nil {
		t.Fatal(err)
	}

	// Two new producers.
	if got := countTag(root, "producer"); got != beforeProducers+2 {
		t.Errorf("producer count = %d, want %d", got, beforeProducers+2)
	}
	// main_bin gained two entries (had one).
	mainBin := root.Find("playlist", "id", "main_bin")
	if got := countDirectTag(mainBin, "entry"); got != 3 {
		t.Errorf("main_bin entries = %d, want 3", got)
	}

	// Ids must not collide with the existing producer7.
	if root.Find("producer", "id", "producer8") == nil {
		t.Error("expected new producer id to continue past existing max (producer8)")
	}

	// Rating routing: hero → Selects (folder 5), b → A-Cam (folder 2).
	hero := findProducerByResource(root, "/abs/originals/hero.mp4")
	if hero == nil {
		t.Fatal("hero producer missing")
	}
	if fid := propValue(hero, "kdenlive:folderid"); fid != "5" {
		t.Errorf("hero folderid = %q, want 5 (Selects)", fid)
	}
	if out, _ := hero.Attr("out"); out != "00:00:42.360" {
		t.Errorf("hero out timecode = %q, want 00:00:42.360", out)
	}
	b := findProducerByResource(root, "/abs/originals/b.mp4")
	if fid := propValue(b, "kdenlive:folderid"); fid != "2" {
		t.Errorf("b folderid = %q, want 2 (A-Cam)", fid)
	}
}

func TestScaffoldNoMainBin(t *testing.T) {
	root, _ := Load([]byte(`<?xml version='1.0'?><mlt><profile/></mlt>`))
	err := Scaffold(root, []ClipRef{{Resource: "x.mp4"}})
	if err == nil || !strings.Contains(err.Error(), "main_bin") {
		t.Fatalf("expected main_bin error, got %v", err)
	}
}

func TestSecondsToTimecode(t *testing.T) {
	cases := map[float64]string{
		0:       "00:00:00.000",
		42.36:   "00:00:42.360",
		3661.5:  "01:01:01.500",
		59.9994: "00:00:59.999",
	}
	for sec, want := range cases {
		if got := secondsToTimecode(sec); got != want {
			t.Errorf("secondsToTimecode(%v) = %q, want %q", sec, got, want)
		}
	}
}

// TestParseRealKdenliveFiles validates the parser against real .kdenlive files
// on this machine when present (the actual "diff procedure" reference). Skips
// cleanly when they aren't there.
func TestParseRealKdenliveFiles(t *testing.T) {
	candidates := []string{
		"/home/chriskim/Videos/projects/db_class_project.kdenlive",
		"/home/chriskim/Videos/Templates/Fast Montage Template.kdenlive",
		"/home/chriskim/Videos/Intro to OS/intro_to_os_presentation.kdenlive",
	}
	ran := 0
	for _, path := range candidates {
		data, err := os.ReadFile(path)
		if err != nil {
			continue
		}
		ran++
		root, err := Load(data)
		if err != nil {
			t.Errorf("Load(%s): %v", filepath.Base(path), err)
			continue
		}
		if root.Find("playlist", "id", "main_bin") == nil {
			t.Errorf("%s: no main_bin found", filepath.Base(path))
		}
		// Render → re-parse must preserve producer/chain counts.
		re, err := Load(root.Render())
		if err != nil {
			t.Errorf("%s: re-parse failed: %v", filepath.Base(path), err)
			continue
		}
		for _, tag := range []string{"producer", "chain", "playlist", "tractor"} {
			if countTag(re, tag) != countTag(root, tag) {
				t.Errorf("%s: %s count changed across round-trip", filepath.Base(path), tag)
			}
		}
	}
	if ran == 0 {
		t.Skip("no real .kdenlive files present")
	}
}

// --- helpers ---

func countTag(n *Node, tag string) int {
	c := 0
	if n.Name == tag {
		c++
	}
	for _, ch := range n.Children {
		c += countTag(ch, tag)
	}
	return c
}

func countDirectTag(n *Node, tag string) int {
	c := 0
	for _, ch := range n.Children {
		if ch.Name == tag {
			c++
		}
	}
	return c
}

func findProducerByResource(n *Node, resource string) *Node {
	for _, c := range n.Children {
		if c.Name == "producer" && propValue(c, "resource") == resource {
			return c
		}
		if got := findProducerByResource(c, resource); got != nil {
			return got
		}
	}
	return nil
}

func propValue(n *Node, name string) string {
	for _, c := range n.Children {
		if c.Name == "property" {
			if v, _ := c.Attr("name"); v == name {
				return strings.TrimSpace(c.Text)
			}
		}
	}
	return ""
}
