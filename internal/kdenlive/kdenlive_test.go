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

func TestScaffoldRoutesByGroup(t *testing.T) {
	root := loadTemplate(t)
	beforeProducers := countTag(root, "producer")
	mainBin := root.Find("playlist", "id", "main_bin")
	beforeFolders := countFolders(mainBin)

	clips := []ClipRef{
		{Resource: "/abs/originals/int1.mp4", DurationSec: 42.36, Group: "Interviews"}, // new folder
		{Resource: "/abs/originals/int2.mp4", DurationSec: 5.0, Group: "Interviews"},   // same new folder
		{Resource: "/abs/originals/br.mp4", DurationSec: 3.0, Group: "B-Roll"},         // reuse template folder 3
		{Resource: "/abs/originals/misc.mp4", DurationSec: 1.0, Group: ""},             // ungrouped → A-Cam (2)
	}
	if err := Scaffold(root, clips, "/proj"); err != nil {
		t.Fatal(err)
	}

	// Four new producers.
	if got := countTag(root, "producer"); got != beforeProducers+4 {
		t.Errorf("producer count = %d, want %d", got, beforeProducers+4)
	}
	// main_bin gained four entries (the template had one).
	if got := countDirectTag(mainBin, "entry"); got != 5 {
		t.Errorf("main_bin entries = %d, want 5", got)
	}
	// Exactly one new bin folder was created (Interviews); B-Roll was reused.
	if got := countFolders(mainBin); got != beforeFolders+1 {
		t.Errorf("bin folders = %d, want %d (only Interviews is new)", got, beforeFolders+1)
	}
	interviewsID := findFolderID(mainBin, "-1", "Interviews")
	if interviewsID == "" {
		t.Fatal("Interviews folder was not created")
	}

	// Routing: both interviews → the new folder; B-Roll → existing folder 3;
	// ungrouped → A-Cam (2).
	for _, res := range []string{"/abs/originals/int1.mp4", "/abs/originals/int2.mp4"} {
		if fid := propValue(findProducerByResource(root, res), "kdenlive:folderid"); fid != interviewsID {
			t.Errorf("%s folderid = %q, want %q (Interviews)", res, fid, interviewsID)
		}
	}
	if fid := propValue(findProducerByResource(root, "/abs/originals/br.mp4"), "kdenlive:folderid"); fid != "3" {
		t.Errorf("b-roll folderid = %q, want 3 (reused B-Roll)", fid)
	}
	if fid := propValue(findProducerByResource(root, "/abs/originals/misc.mp4"), "kdenlive:folderid"); fid != "2" {
		t.Errorf("ungrouped folderid = %q, want 2 (A-Cam)", fid)
	}

	// The created folder id must not collide with any producer id or existing
	// folder id (existing folders run 2–5, producers up through 7).
	if interviewsID == "2" || interviewsID == "3" || interviewsID == "4" || interviewsID == "5" {
		t.Errorf("Interviews folder id %q collides with an existing folder id", interviewsID)
	}
	for _, res := range []string{"/abs/originals/int1.mp4"} {
		p := findProducerByResource(root, res)
		if id := propValue(p, "kdenlive:id"); id == interviewsID {
			t.Errorf("producer id %q collides with the Interviews folder id", id)
		}
	}
}

// TestScaffoldPreLinksProxy checks the proxy overlay: a clip with a proxy gets
// kdenlive:proxy + kdenlive:originalurl (resource stays the original), and the
// project-level enableproxy toggle is flipped to 1. A clip without a proxy gets
// neither.
func TestScaffoldPreLinksProxy(t *testing.T) {
	root := loadTemplate(t)
	mainBin := root.Find("playlist", "id", "main_bin")

	clips := []ClipRef{
		{Resource: "/p/originals/a.MP4", Proxy: "proxy/a.mp4", DurationSec: 2}, // relative proxy
		{Resource: "/p/originals/b.MP4", Proxy: "", DurationSec: 2},            // no proxy
	}
	if err := Scaffold(root, clips, "/proj"); err != nil {
		t.Fatal(err)
	}

	a := findProducerByResource(root, "/p/originals/a.MP4")
	if got := propValue(a, "kdenlive:proxy"); got != "proxy/a.mp4" {
		t.Errorf("a kdenlive:proxy = %q, want proxy/a.mp4 (project-relative)", got)
	}
	if got := propValue(a, "resource"); got != "/p/originals/a.MP4" {
		t.Errorf("a resource = %q, want the original (proxy is an overlay)", got)
	}

	b := findProducerByResource(root, "/p/originals/b.MP4")
	if got := propValue(b, "kdenlive:proxy"); got != "" {
		t.Errorf("b kdenlive:proxy = %q, want empty (no proxy)", got)
	}

	// Project-level proxy toggle flipped on.
	if got := propValue(mainBin, "kdenlive:docproperties.enableproxy"); got != "1" {
		t.Errorf("enableproxy = %q, want 1", got)
	}

	// The document root must be repointed at the project (not the template's
	// authoring path), or every project-relative path resolves to nowhere and
	// Kdenlive reports all clips missing.
	if got, _ := root.Attr("root"); got != "/proj" {
		t.Errorf("mlt root = %q, want /proj (the project dir)", got)
	}
}

// TestScaffoldEnableProxyReplacesTemplateZero guards the update-in-place path: a
// template that ships enableproxy=0 must end up at 1, not gain a duplicate.
func TestScaffoldEnableProxyReplacesTemplateZero(t *testing.T) {
	root := loadTemplate(t)
	mainBin := root.Find("playlist", "id", "main_bin")
	mainBin.Children = append([]*Node{prop("kdenlive:docproperties.enableproxy", "0")}, mainBin.Children...)

	if err := Scaffold(root, []ClipRef{{Resource: "originals/a.MP4", Proxy: "proxy/a.mp4"}}, "/proj"); err != nil {
		t.Fatal(err)
	}
	n := 0
	for _, c := range mainBin.Children {
		if c.Name == "property" {
			if name, _ := c.Attr("name"); name == "kdenlive:docproperties.enableproxy" {
				n++
				if c.Text != "1" {
					t.Errorf("enableproxy = %q, want 1", c.Text)
				}
			}
		}
	}
	if n != 1 {
		t.Errorf("enableproxy property count = %d, want exactly 1 (updated in place)", n)
	}
}

func TestScaffoldNoMainBin(t *testing.T) {
	root, _ := Load([]byte(`<?xml version='1.0'?><mlt><profile/></mlt>`))
	err := Scaffold(root, []ClipRef{{Resource: "x.mp4"}}, "/proj")
	if err == nil || !strings.Contains(err.Error(), "main_bin") {
		t.Fatalf("expected main_bin error, got %v", err)
	}
}

// TestScaffoldNestedGroups covers arbitrary-depth paths: "london/b_roll" and
// "london/interviews" share a created "london" parent, a deep path nests three
// levels, and each leaf's folder is parented at the right id.
func TestScaffoldNestedGroups(t *testing.T) {
	root := loadTemplate(t)
	mainBin := root.Find("playlist", "id", "main_bin")

	clips := []ClipRef{
		{Resource: "/abs/a.mp4", DurationSec: 1, Group: "london/b_roll"},
		{Resource: "/abs/b.mp4", DurationSec: 1, Group: "london/interviews"},
		{Resource: "/abs/c.mp4", DurationSec: 1, Group: "london/b_roll"}, // reuse both levels
		{Resource: "/abs/d.mp4", DurationSec: 1, Group: "paris/day1/market"},
	}
	if err := Scaffold(root, clips, "/proj"); err != nil {
		t.Fatal(err)
	}

	// london created once at root; b_roll and interviews nested under it.
	london := findFolderID(mainBin, "-1", "london")
	if london == "" {
		t.Fatal("london folder not created at root")
	}
	bRoll := findFolderID(mainBin, london, "b_roll")
	interviews := findFolderID(mainBin, london, "interviews")
	if bRoll == "" || interviews == "" {
		t.Fatalf("nested folders missing: b_roll=%q interviews=%q", bRoll, interviews)
	}
	if bRoll == interviews {
		t.Error("b_roll and interviews must be distinct folders under london")
	}

	// Deep path paris/day1/market nests three levels, each parented correctly.
	paris := findFolderID(mainBin, "-1", "paris")
	day1 := findFolderID(mainBin, paris, "day1")
	market := findFolderID(mainBin, day1, "market")
	if paris == "" || day1 == "" || market == "" {
		t.Fatalf("deep path not built: paris=%q day1=%q market=%q", paris, day1, market)
	}

	// Routing: clips a & c → b_roll; b → interviews; d → market.
	if fid := propValue(findProducerByResource(root, "/abs/a.mp4"), "kdenlive:folderid"); fid != bRoll {
		t.Errorf("a folderid = %q, want %q (london/b_roll)", fid, bRoll)
	}
	if fid := propValue(findProducerByResource(root, "/abs/c.mp4"), "kdenlive:folderid"); fid != bRoll {
		t.Errorf("c folderid = %q, want %q (reused london/b_roll)", fid, bRoll)
	}
	if fid := propValue(findProducerByResource(root, "/abs/b.mp4"), "kdenlive:folderid"); fid != interviews {
		t.Errorf("b folderid = %q, want %q (london/interviews)", fid, interviews)
	}
	if fid := propValue(findProducerByResource(root, "/abs/d.mp4"), "kdenlive:folderid"); fid != market {
		t.Errorf("d folderid = %q, want %q (paris/day1/market)", fid, market)
	}

	// The generated project must still round-trip through the parser.
	if _, err := Load(root.Render()); err != nil {
		t.Fatalf("scaffolded project no longer parses: %v", err)
	}
}

// findFolderID returns the id of the bin folder named `name` directly under
// parent id `parent` ("-1" for root), or "" if absent.
func findFolderID(mainBin *Node, parent, name string) string {
	for _, c := range mainBin.Children {
		if c.Name != "property" {
			continue
		}
		pn, _ := c.Attr("name")
		if m := folderRe.FindStringSubmatch(pn); m != nil && m[1] == parent && strings.TrimSpace(c.Text) == name {
			return m[2]
		}
	}
	return ""
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

// countFolders counts kdenlive:folder.* properties directly under a node.
func countFolders(n *Node) int {
	c := 0
	for _, ch := range n.Children {
		if ch.Name == "property" {
			if name, _ := ch.Attr("name"); folderRe.MatchString(name) {
				c++
			}
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
