package kdenlive

import (
	"encoding/json"
	"fmt"
	"strconv"
)

// Guide property and profile field names, pinned against a real Kdenlive
// project. Kdenlive 7.38.0 stores timeline guides as a JSON array under the
// sequence property below; each guide has a frame position and a comment.
//
// Verified 2026-08-28 against real .kdenlive files on disk (Intro to OS,
// db_class). If a future Kdenlive version changes these, Guides() fails loudly
// (see GuidesRaw) rather than silently emitting zero chapters (plan §11).
const (
	guidesProperty   = "kdenlive:sequenceproperties.guides"
	frameRateNumAttr = "frame_rate_num"
	frameRateDenAttr = "frame_rate_den"
)

// Guide is one timeline marker: a frame position and its label.
type Guide struct {
	Pos     int    `json:"pos"` // frame index
	Comment string `json:"comment"`
}

// FPS returns the project's frame rate as num/den from the <profile> element.
func (n *Node) FPS() (num, den int, err error) {
	profile := n.Find("profile", "", "")
	if profile == nil {
		return 0, 0, fmt.Errorf("no <profile> element in project")
	}
	ns, ok1 := profile.Attr(frameRateNumAttr)
	ds, ok2 := profile.Attr(frameRateDenAttr)
	if !ok1 || !ok2 {
		return 0, 0, fmt.Errorf("<profile> missing %s/%s", frameRateNumAttr, frameRateDenAttr)
	}
	num, err = strconv.Atoi(ns)
	if err != nil {
		return 0, 0, fmt.Errorf("bad %s %q: %w", frameRateNumAttr, ns, err)
	}
	den, err = strconv.Atoi(ds)
	if err != nil || den == 0 {
		return 0, 0, fmt.Errorf("bad %s %q", frameRateDenAttr, ds)
	}
	return num, den, nil
}

// Guides parses the timeline guides. It fails loudly (dumping the raw property)
// if the guides property is absent or not the expected JSON array, per the
// plan's invariant — never silently return zero chapters.
func (n *Node) Guides() ([]Guide, error) {
	prop := n.Find("property", "name", guidesProperty)
	if prop == nil {
		return nil, fmt.Errorf("no %q property found — this Kdenlive version may store guides differently; refusing to emit zero chapters", guidesProperty)
	}
	raw := prop.Text
	var guides []Guide
	if err := json.Unmarshal([]byte(raw), &guides); err != nil {
		return nil, fmt.Errorf("guides property is not the expected JSON array (Kdenlive format changed?): %w\nraw value: %s", err, raw)
	}
	return guides, nil
}
