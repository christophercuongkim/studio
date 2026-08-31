package kdenlive

import (
	"encoding/xml"
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

// ClipRef is one clip to place in the project bin.
type ClipRef struct {
	Resource    string // absolute path to the original media
	Proxy       string // project-relative path to the low-res proxy, or "" if none
	DurationSec float64
	Group       string // bin folder to route into; empty routes to A-Cam
}

// binACam is the default bin folder for clips with no explicit group.
const binACam = "A-Cam"

// folderRe matches a bin-folder property name: kdenlive:folder.<parent>.<id>,
// capturing the parent id (group 1) and the folder's own id (group 2). A parent
// of -1 is a root-level folder; any other parent nests the folder under it.
var folderRe = regexp.MustCompile(`^kdenlive:folder\.(-?\d+)\.(\d+)$`)

// idRe pulls the trailing integer from producer/chain ids like "chain12".
var idRe = regexp.MustCompile(`(\d+)$`)

// Scaffold injects one producer per clip into the template's bin and references
// each from the main_bin playlist, routing each clip into the bin folder named
// by its Group. Group is a "/"-separated path (e.g. "london/b_roll") that nests
// to arbitrary depth: each segment reuses an existing folder of that name under
// the same parent, or creates a new kdenlive:folder property parented correctly.
// Ungrouped clips route to A-Cam if it exists, else the bin root. It mutates root
// in place.
func Scaffold(root *Node, clips []ClipRef) error {
	mainBin := root.Find("playlist", "id", "main_bin")
	if mainBin == nil {
		return fmt.Errorf("template has no <playlist id=\"main_bin\"> — not a Kdenlive project bin")
	}

	// index keys a folder by parent-id + name → its own id, so a name can repeat
	// under different parents (real nesting). maxFolderID seeds the id counter
	// with folder ids, which live in the property *name* and are invisible to the
	// generic id scan.
	index, maxFolderID := folderIndex(mainBin)
	nextID := max(maxID(root), maxFolderID) + 1

	folderKey := func(parent, name string) string { return parent + "\x00" + name }

	// ensurePath walks a "/"-separated group path, creating any missing folder
	// along the way (parented under the previous segment), and returns the leaf
	// folder id. An empty path routes to A-Cam if present, else the bin root.
	var newFolders []*Node
	ensurePath := func(path string) string {
		if path == "" {
			if id, ok := index[folderKey("-1", binACam)]; ok {
				return id
			}
			return "-1"
		}
		parent := "-1"
		for _, seg := range strings.Split(path, "/") {
			key := folderKey(parent, seg)
			if id, ok := index[key]; ok {
				parent = id
				continue
			}
			id := strconv.Itoa(nextID)
			nextID++
			index[key] = id
			newFolders = append(newFolders, prop("kdenlive:folder."+parent+"."+id, seg))
			parent = id
		}
		return parent
	}

	// Pass 1: create every folder the clips need (folders take the lower ids).
	leafID := make([]string, len(clips))
	for i, c := range clips {
		leafID[i] = ensurePath(c.Group)
	}

	// Pass 2: build producers and their bin entries, routed to their leaf folder.
	var producers []*Node
	var entries []*Node
	for i, c := range clips {
		id := nextID
		nextID++
		pid := fmt.Sprintf("producer%d", id)
		tc := secondsToTimecode(c.DurationSec)

		p := &Node{Name: "producer", Attrs: attrs("id", pid, "out", tc)}
		p.Children = []*Node{
			prop("resource", c.Resource),
			prop("mlt_service", "avformat-novalidate"),
			prop("kdenlive:id", strconv.Itoa(id)),
			prop("kdenlive:folderid", leafID[i]),
			prop("kdenlive:clip_type", "0"),
		}
		// Pre-link studio's proxy so Kdenlive uses it directly instead of running
		// its (unreliable) external-proxy match or regenerating one. resource
		// stays the original; kdenlive:proxy overlays the project-relative
		// low-res file — matching exactly how Kdenlive serialises a proxied clip
		// (relative path, no originalurl on the base producer). Needs
		// enableproxy=1 (set on main_bin below) to take effect.
		if c.Proxy != "" {
			p.Children = append(p.Children, prop("kdenlive:proxy", c.Proxy))
		}
		producers = append(producers, p)

		e := &Node{Name: "entry", Attrs: attrs("in", "00:00:00.000", "out", tc, "producer", pid)}
		entries = append(entries, e)
	}

	// Turn the project-level "Proxy clips" toggle on so the pre-linked
	// kdenlive:proxy overlays actually take effect when the project opens. Only
	// bother if at least one clip carries a proxy.
	if anyProxy(clips) {
		setDocProperty(mainBin, "kdenlive:docproperties.enableproxy", "1")
	}

	// Folder properties belong in main_bin alongside the existing ones (prepend
	// so they sit with the other folder props, ahead of the entries). newFolders
	// is freshly allocated, so a plain prepend can't alias the template's slice.
	mainBin.Children = append(newFolders, mainBin.Children...)
	insertBefore(root, mainBin, producers)
	mainBin.Children = append(mainBin.Children, entries...)
	return nil
}

func anyProxy(clips []ClipRef) bool {
	for _, c := range clips {
		if c.Proxy != "" {
			return true
		}
	}
	return false
}

// setDocProperty sets a kdenlive:docproperties.* property on main_bin, updating
// the value in place if it already exists (e.g. a template shipping
// enableproxy=0) or inserting a new property otherwise.
func setDocProperty(mainBin *Node, name, value string) {
	for _, c := range mainBin.Children {
		if c.Name == "property" {
			if n, _ := c.Attr("name"); n == name {
				c.Text = value
				return
			}
		}
	}
	mainBin.Children = append([]*Node{prop(name, value)}, mainBin.Children...)
}

// folderIndex maps parent-id + "\x00" + name → folder id for every bin folder in
// main_bin, and returns the largest folder id seen (for id allocation).
func folderIndex(mainBin *Node) (map[string]string, int) {
	index := map[string]string{}
	max := 0
	for _, c := range mainBin.Children {
		if c.Name != "property" {
			continue
		}
		name, ok := c.Attr("name")
		if !ok {
			continue
		}
		if m := folderRe.FindStringSubmatch(name); m != nil {
			parent, id := m[1], m[2]
			index[parent+"\x00"+strings.TrimSpace(c.Text)] = id
			if i, err := strconv.Atoi(id); err == nil && i > max {
				max = i
			}
		}
	}
	return index, max
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
