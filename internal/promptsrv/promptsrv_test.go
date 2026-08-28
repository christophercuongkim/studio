package promptsrv

import (
	"bufio"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/christophercuongkim/studio/internal/script"
)

func sampleScript() *script.Script {
	return &script.Script{
		Title: "demo",
		Sections: []script.Section{
			{Title: "Hook", Bullets: []script.Bullet{{Text: "a"}, {Text: "b"}}},
			{Title: "Body", Bullets: []script.Bullet{{Text: "c"}}},
			{Title: "Outro", Bullets: []script.Bullet{{Text: "d"}}},
		},
	}
}

func TestAdvanceAndRetreat(t *testing.T) {
	s := New(sampleScript(), "")

	// next within a section.
	if ev := s.apply("next"); ev != "bullet" || s.st.Bullet != 1 {
		t.Fatalf("next → %s, bullet %d", ev, s.st.Bullet)
	}
	// next past the last bullet crosses into the next section.
	if ev := s.apply("next"); ev != "section" || s.st.Section != 1 || s.st.Bullet != 0 {
		t.Fatalf("cross section → %s, sec %d bullet %d", ev, s.st.Section, s.st.Bullet)
	}
	// back at bullet 0 retreats to the previous section's last bullet.
	if ev := s.apply("back"); ev != "section" || s.st.Section != 0 || s.st.Bullet != 1 {
		t.Fatalf("back across section → %s, sec %d bullet %d", ev, s.st.Section, s.st.Bullet)
	}
}

func TestSectionJumpsAndBounds(t *testing.T) {
	s := New(sampleScript(), "")
	s.apply("nextSection")
	s.apply("nextSection")
	if s.st.Section != 2 {
		t.Fatalf("section = %d, want 2", s.st.Section)
	}
	// Can't go past the last section.
	if ev := s.apply("nextSection"); ev != "" || s.st.Section != 2 {
		t.Fatalf("overshoot → %s, section %d", ev, s.st.Section)
	}
	// prevSection back to 0 and no further.
	s.apply("prevSection")
	s.apply("prevSection")
	if ev := s.apply("prevSection"); ev != "" || s.st.Section != 0 {
		t.Fatalf("undershoot → %s, section %d", ev, s.st.Section)
	}
}

func TestTakeAndBlackout(t *testing.T) {
	s := New(sampleScript(), "")
	s.apply("next")
	if ev := s.apply("take"); ev != "take" || s.st.Take != 1 || s.st.Bullet != 0 {
		t.Fatalf("take → %s, take %d bullet %d", ev, s.st.Take, s.st.Bullet)
	}
	if s.apply("blackout"); !s.st.Blackout {
		t.Fatal("blackout not set")
	}
	if s.apply("blackout"); s.st.Blackout {
		t.Fatal("blackout not toggled off")
	}
}

func TestHTTPStateAndAction(t *testing.T) {
	s := New(sampleScript(), "")
	ts := httptest.NewServer(s.Handler())
	defer ts.Close()

	// Script endpoint.
	var sc struct {
		Sections []script.Section `json:"sections"`
	}
	getJSON(t, ts.URL+"/api/script", &sc)
	if len(sc.Sections) != 3 {
		t.Errorf("script sections = %d", len(sc.Sections))
	}

	// Action advances shared state.
	var st State
	postJSON(t, ts.URL+"/api/action?a=next", &st)
	if st.Bullet != 1 {
		t.Errorf("after next, bullet = %d", st.Bullet)
	}
	// State endpoint reflects it.
	var st2 State
	getJSON(t, ts.URL+"/api/state", &st2)
	if st2.Bullet != 1 {
		t.Errorf("state bullet = %d", st2.Bullet)
	}
}

func TestLogging(t *testing.T) {
	dir := t.TempDir()
	logPath := filepath.Join(dir, "prompt-log.jsonl")
	s := New(sampleScript(), logPath)

	s.log("section", State{Section: 0, Take: 1})
	s.log("take", State{Section: 1, Take: 2})

	f, err := os.Open(logPath)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	var events []logEvent
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		var e logEvent
		if err := json.Unmarshal(sc.Bytes(), &e); err != nil {
			t.Fatalf("bad log line: %v", err)
		}
		events = append(events, e)
	}
	if len(events) != 2 || events[0].Event != "section" || events[1].Section != "Body" || events[1].Take != 2 {
		t.Errorf("log events wrong: %+v", events)
	}
	if events[0].TS == "" {
		t.Error("log event missing timestamp")
	}
}

func getJSON(t *testing.T, url string, v any) {
	t.Helper()
	res, err := http.Get(url)
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	json.NewDecoder(res.Body).Decode(v)
}

func postJSON(t *testing.T, url string, v any) {
	t.Helper()
	res, err := http.Post(url, "application/json", nil)
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	json.NewDecoder(res.Body).Decode(v)
}
