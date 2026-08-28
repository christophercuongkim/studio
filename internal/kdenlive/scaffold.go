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
	DurationSec float64
	Rating      int // 0–5; ≥4 goes to the Selects bin, else A-Cam
}

// Bin folder names created in the template (plan §9.3). Rating routes a clip to
// one of these.
const (
	binSelects = "Selects"
	binACam    = "A-Cam"
)

// folderRe matches a bin-folder property name: kdenlive:folder.<parent>.<id>.
var folderRe = regexp.MustCompile(`^kdenlive:folder\.-?\d+\.(\d+)$`)

// idRe pulls the trailing integer from producer/chain ids like "chain12".
var idRe = regexp.MustCompile(`(\d+)$`)

// Scaffold injects one producer per clip into the template's bin and references
// each from the main_bin playlist, routing by rating into the Selects or A-Cam
// folder when those exist. It mutates root in place.
func Scaffold(root *Node, clips []ClipRef) error {
	mainBin := root.Find("playlist", "id", "main_bin")
	if mainBin == nil {
		return fmt.Errorf("template has no <playlist id=\"main_bin\"> — not a Kdenlive project bin")
	}

	folders := binFolders(mainBin)
	nextID := maxID(root) + 1

	// Build producers and their bin entries.
	var producers []*Node
	var entries []*Node
	for _, c := range clips {
		id := nextID
		nextID++
		pid := fmt.Sprintf("producer%d", id)
		tc := secondsToTimecode(c.DurationSec)

		folderID := "-1"
		bin := binACam
		if c.Rating >= 4 {
			bin = binSelects
		}
		if fid, ok := folders[bin]; ok {
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

	insertBefore(root, mainBin, producers)
	mainBin.Children = append(mainBin.Children, entries...)
	return nil
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
