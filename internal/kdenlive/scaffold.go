package kdenlive

import (
	"encoding/xml"
	"fmt"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

// ClipRef is one clip to place in the project bin.
type ClipRef struct {
	Resource    string // absolute path to the original media
	DurationSec float64
	Group       string // bin folder to route into; empty routes to A-Cam
}

// binACam is the default bin folder for clips with no explicit group.
const binACam = "A-Cam"

// folderRe matches a bin-folder property name: kdenlive:folder.<parent>.<id>.
var folderRe = regexp.MustCompile(`^kdenlive:folder\.-?\d+\.(\d+)$`)

// idRe pulls the trailing integer from producer/chain ids like "chain12".
var idRe = regexp.MustCompile(`(\d+)$`)

// Scaffold injects one producer per clip into the template's bin and references
// each from the main_bin playlist, routing each clip into the bin folder named
// by its Group. A folder that already exists in the template (by exact name) is
// reused; a new group name gets a fresh kdenlive:folder property created in
// main_bin. Ungrouped clips route to A-Cam if that folder exists, else the bin
// root. It mutates root in place.
func Scaffold(root *Node, clips []ClipRef) error {
	mainBin := root.Find("playlist", "id", "main_bin")
	if mainBin == nil {
		return fmt.Errorf("template has no <playlist id=\"main_bin\"> — not a Kdenlive project bin")
	}

	folders := binFolders(mainBin)
	// Folder ids live in the property *name* (kdenlive:folder.-1.<id>), which the
	// generic id scan doesn't see; fold them in so a created folder or producer
	// never reuses an existing folder's id.
	nextID := maxID(root)
	for _, fid := range folders {
		if i, err := strconv.Atoi(fid); err == nil && i > nextID {
			nextID = i
		}
	}
	nextID++

	// Create any folder named by a group that the template doesn't already have.
	// Sort the distinct new names so id assignment is deterministic.
	var newFolders []*Node
	for _, name := range distinctNewGroups(clips, folders) {
		fid := strconv.Itoa(nextID)
		nextID++
		folders[name] = fid
		newFolders = append(newFolders, prop("kdenlive:folder.-1."+fid, name))
	}

	// Build producers and their bin entries.
	var producers []*Node
	var entries []*Node
	for _, c := range clips {
		id := nextID
		nextID++
		pid := fmt.Sprintf("producer%d", id)
		tc := secondsToTimecode(c.DurationSec)

		// A grouped clip goes to its folder (guaranteed to exist now); an
		// ungrouped clip falls back to A-Cam, or the bin root if there's no A-Cam.
		want := c.Group
		if want == "" {
			want = binACam
		}
		folderID := "-1"
		if fid, ok := folders[want]; ok {
			folderID = fid
		}

		p := &Node{Name: "producer", Attrs: attrs("id", pid, "out", tc)}
		p.Children = []*Node{
			prop("resource", c.Resource),
			prop("mlt_service", "avformat-novalidate"),
			prop("kdenlive:id", strconv.Itoa(id)),
			prop("kdenlive:folderid", folderID),
			prop("kdenlive:clip_type", "0"),
		}
		producers = append(producers, p)

		e := &Node{Name: "entry", Attrs: attrs("in", "00:00:00.000", "out", tc, "producer", pid)}
		entries = append(entries, e)
	}

	// Folder properties belong in main_bin alongside the existing ones (prepend
	// so they sit with the other folder props, ahead of the entries). newFolders
	// is freshly allocated, so a plain prepend can't alias the template's slice.
	mainBin.Children = append(newFolders, mainBin.Children...)
	insertBefore(root, mainBin, producers)
	mainBin.Children = append(mainBin.Children, entries...)
	return nil
}

// distinctNewGroups returns, in stable sorted order, the group names referenced
// by clips that don't already have a folder in the template.
func distinctNewGroups(clips []ClipRef, have map[string]string) []string {
	seen := map[string]bool{}
	var out []string
	for _, c := range clips {
		if c.Group == "" {
			continue
		}
		if _, ok := have[c.Group]; ok {
			continue // reuse an existing template folder of the same name
		}
		if seen[c.Group] {
			continue
		}
		seen[c.Group] = true
		out = append(out, c.Group)
	}
	sort.Strings(out)
	return out
}

// binFolders maps a bin folder name to its numeric id, from the main_bin
// playlist's kdenlive:folder.<parent>.<id> properties.
func binFolders(mainBin *Node) map[string]string {
	out := map[string]string{}
	for _, c := range mainBin.Children {
		if c.Name != "property" {
			continue
		}
		name, ok := c.Attr("name")
		if !ok {
			continue
		}
		if m := folderRe.FindStringSubmatch(name); m != nil {
			out[strings.TrimSpace(c.Text)] = m[1]
		}
	}
	return out
}

// maxID returns the largest integer id found on any element id attribute or
// kdenlive:id property, so new producers get non-colliding ids.
func maxID(n *Node) int {
	max := 0
	var walk func(*Node)
	walk = func(nd *Node) {
		if v, ok := nd.Attr("id"); ok {
			if m := idRe.FindString(v); m != "" {
				if i, _ := strconv.Atoi(m); i > max {
					max = i
				}
			}
		}
		if nd.Name == "property" {
			if name, _ := nd.Attr("name"); name == "kdenlive:id" {
				if i, err := strconv.Atoi(strings.TrimSpace(nd.Text)); err == nil && i > max {
					max = i
				}
			}
		}
		for _, c := range nd.Children {
			walk(c)
		}
	}
	walk(n)
	return max
}

// insertBefore inserts nodes into parent's child list immediately before target.
func insertBefore(parent, target *Node, nodes []*Node) {
	for i, c := range parent.Children {
		if c == target {
			parent.Children = append(parent.Children[:i:i], append(nodes, parent.Children[i:]...)...)
			return
		}
	}
	// target not a direct child (shouldn't happen for main_bin) — append.
	parent.Children = append(parent.Children, nodes...)
}

func prop(name, value string) *Node {
	return &Node{Name: "property", Attrs: attrs("name", name), Text: value}
}

func attrs(kv ...string) []xml.Attr {
	out := make([]xml.Attr, 0, len(kv)/2)
	for i := 0; i+1 < len(kv); i += 2 {
		out = append(out, xml.Attr{Name: xml.Name{Local: kv[i]}, Value: kv[i+1]})
	}
	return out
}

// secondsToTimecode formats seconds as HH:MM:SS.mmm (MLT's timecode form).
func secondsToTimecode(sec float64) string {
	if sec < 0 {
		sec = 0
	}
	ms := int(sec*1000 + 0.5)
	h := ms / 3600000
	ms -= h * 3600000
	m := ms / 60000
	ms -= m * 60000
	s := ms / 1000
	ms -= s * 1000
	return fmt.Sprintf("%02d:%02d:%02d.%03d", h, m, s, ms)
}
