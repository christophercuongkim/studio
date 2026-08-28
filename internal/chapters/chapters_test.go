package chapters

import (
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/christophercuongkim/studio/internal/kdenlive"
)

// doc builds a parsed .kdenlive with the given fps and guides JSON.
func doc(t *testing.T, num, den int, guidesJSON string) *kdenlive.Node {
	t.Helper()
	xml := fmt.Sprintf(`<?xml version='1.0'?>
<mlt>
 <profile frame_rate_num="%d" frame_rate_den="%d"/>
 <tractor>
  <property name="kdenlive:sequenceproperties.guides">%s</property>
 </tractor>
</mlt>`, num, den, guidesJSON)
	root, err := kdenlive.Load([]byte(xml))
	if err != nil {
		t.Fatal(err)
	}
	return root
}

func TestFromProjectBasic(t *testing.T) {
	// 30fps: frames 0, 300, 900 → 0s, 10s, 30s.
	root := doc(t, 30, 1, `[{"pos":0,"comment":"Intro"},{"pos":300,"comment":"Setup"},{"pos":900,"comment":"Demo"}]`)
	res, err := FromProject(root, 0)
	if err != nil {
		t.Fatal(err)
	}
	want := "0:00 Intro\n0:10 Setup\n0:30 Demo"
	if res.Text() != want {
		t.Errorf("Text() =\n%s\nwant\n%s", res.Text(), want)
	}
	if len(res.Warnings) != 0 {
		t.Errorf("unexpected warnings: %v", res.Warnings)
	}
}

func TestSynthesizeIntroWhenFirstNotZero(t *testing.T) {
	// First guide at 10s → synthesize 0:00 Intro, warn.
	root := doc(t, 30, 1, `[{"pos":300,"comment":"Setup"},{"pos":900,"comment":"Demo"},{"pos":1500,"comment":"End"}]`)
	res, err := FromProject(root, 0)
	if err != nil {
		t.Fatal(err)
	}
	if res.Chapters[0].StartSec != 0 || res.Chapters[0].Title != "Intro" {
		t.Errorf("no synthesized intro: %+v", res.Chapters[0])
	}
	if len(res.Warnings) != 1 {
		t.Errorf("expected a warning about the synthesized intro")
	}
}

func TestMinChapters(t *testing.T) {
	root := doc(t, 30, 1, `[{"pos":0,"comment":"A"},{"pos":900,"comment":"B"}]`)
	_, err := FromProject(root, 0)
	if err == nil || !strings.Contains(err.Error(), "at least 3") {
		t.Fatalf("expected min-chapter error, got %v", err)
	}
}

func TestShortChapterRejected(t *testing.T) {
	// 0s, 5s (150 frames), 30s → the 0→5s gap is too short.
	root := doc(t, 30, 1, `[{"pos":0,"comment":"A"},{"pos":150,"comment":"B"},{"pos":900,"comment":"C"}]`)
	_, err := FromProject(root, 0)
	if err == nil || !strings.Contains(err.Error(), "shorter than") {
		t.Fatalf("expected short-chapter error, got %v", err)
	}
}

func TestOffsetShifts(t *testing.T) {
	// Offset -10s makes frame 300 (10s) land at 0.
	root := doc(t, 30, 1, `[{"pos":300,"comment":"A"},{"pos":900,"comment":"B"},{"pos":1500,"comment":"C"}]`)
	res, err := FromProject(root, -10)
	if err != nil {
		t.Fatal(err)
	}
	if res.Chapters[0].StartSec != 0 {
		t.Errorf("offset not applied: first at %v", res.Chapters[0].StartSec)
	}
	if len(res.Warnings) != 0 {
		t.Errorf("no intro synthesis expected when offset lands first at 0: %v", res.Warnings)
	}
}

func TestMissingGuidesFailsLoud(t *testing.T) {
	root, _ := kdenlive.Load([]byte(`<?xml version='1.0'?><mlt><profile frame_rate_num="30" frame_rate_den="1"/></mlt>`))
	_, err := FromProject(root, 0)
	if err == nil || !strings.Contains(err.Error(), "guides") {
		t.Fatalf("expected loud guides error, got %v", err)
	}
}

func TestHourFormat(t *testing.T) {
	// 30fps: frame 108000 = 3600s = 1:00:00.
	root := doc(t, 30, 1, `[{"pos":0,"comment":"A"},{"pos":54000,"comment":"B"},{"pos":108000,"comment":"C"}]`)
	res, err := FromProject(root, 0)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(res.Text(), "1:00:00 C") {
		t.Errorf("hour format missing:\n%s", res.Text())
	}
}

func TestInsertIntoDescription(t *testing.T) {
	block := "0:00 Intro\n0:10 Setup"

	// Append when no sentinels.
	got := InsertIntoDescription("My video.", block)
	if !strings.HasPrefix(got, "My video.") || !strings.Contains(got, StartSentinel) || !strings.Contains(got, block) {
		t.Errorf("append form wrong:\n%s", got)
	}

	// Replace between existing sentinels, leaving surrounding text byte-stable.
	desc := "Head text\n\n" + StartSentinel + "\nOLD\n" + EndSentinel + "\n\nTail text"
	got = InsertIntoDescription(desc, block)
	if !strings.HasPrefix(got, "Head text\n\n") || !strings.HasSuffix(got, "\n\nTail text") {
		t.Errorf("surrounding text not preserved:\n%s", got)
	}
	if strings.Contains(got, "OLD") {
		t.Errorf("old block not replaced:\n%s", got)
	}
	if !strings.Contains(got, block) {
		t.Errorf("new block missing:\n%s", got)
	}
}

// TestRealProjectGuides runs against a real .kdenlive with guides when present.
func TestRealProjectGuides(t *testing.T) {
	path := "/home/chriskim/Videos/Intro to OS/intro_to_os_presentation.kdenlive"
	data, err := os.ReadFile(path)
	if err != nil {
		t.Skip("real project not present")
	}
	root, err := kdenlive.Load(data)
	if err != nil {
		t.Fatal(err)
	}
	res, err := FromProject(root, 0)
	if err != nil {
		t.Fatalf("FromProject on real guides: %v", err)
	}
	if len(res.Chapters) < 3 {
		t.Errorf("expected >=3 chapters, got %d", len(res.Chapters))
	}
	if res.Chapters[0].StartSec != 0 {
		t.Errorf("first real chapter should be at 0:00, got %v", res.Chapters[0].StartSec)
	}
	t.Logf("real chapters:\n%s", res.Text())
}
