package manifest

import (
	"path/filepath"
	"reflect"
	"testing"
	"time"
)

// sample builds a valid two-clip manifest for round-trip and validation tests.
func sample(t *testing.T) *Manifest {
	t.Helper()
	created := time.Date(2026, 8, 27, 14, 23, 1, 0, time.UTC)
	take := 2
	reviewed := time.Date(2026, 8, 27, 19, 0, 0, 0, time.UTC)
	return &Manifest{
		SchemaVersion: SchemaVersion,
		Shoot: Shoot{
			Root:      "/abs/path/2026-08-27_lake-trip",
			Title:     "lake-trip",
			CreatedAt: created,
			CamCode:   "DJI",
		},
		Clips: []Clip{
			{
				ID:   "c-014",
				Seq:  14,
				Stem: "DJI_20260827_142301_0014_D",
				Files: Files{
					Original: "originals/DJI_20260827_142301_0014_D.MP4",
					Proxy:    "proxy/DJI_20260827_142301_0014_D.mp4",
					Sidecars: []string{"originals/DJI_20260827_142301_0014_D.SRT"},
				},
				Media: Media{
					DurationSec: 42.36, Width: 3840, Height: 2160, FPS: "29.97",
					VCodec: "hevc", HasAudio: true, CreatedAt: created,
					SizeBytes: 512034441, XXH64: "9f2c7c1a55aa1b02",
				},
				ProxyInfo: ProxyInfo{Source: ProxyCamera},
				Review: Review{
					Status: StatusKept, Rating: 5, Desc: "sunset-dock-wide",
					Take: &take, ReviewedAt: &reviewed,
				},
				Applied: Applied{Done: false},
			},
			{
				ID:        "c-015",
				Seq:       15,
				Stem:      "DJI_0015",
				Files:     Files{Original: "originals/DJI_0015.MP4"},
				Media:     Media{DurationSec: 3.0, Width: 1920, Height: 1080, VCodec: "hevc"},
				ProxyInfo: ProxyInfo{Source: ProxyNone, Note: "ffmpeg failed: invalid data"},
				Review:    Review{Status: StatusPending},
			},
		},
	}
}

func TestRoundTrip(t *testing.T) {
	dir := t.TempDir()
	orig := sample(t)

	// Save then load through the filesystem. Create the referenced files so
	// ValidatePaths passes.
	writeReferencedFiles(t, dir, orig)
	if err := Save(dir, orig); err != nil {
		t.Fatalf("Save: %v", err)
	}
	got, err := Load(dir)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if !reflect.DeepEqual(orig, got) {
		t.Errorf("round-trip mismatch:\n orig = %+v\n got  = %+v", orig, got)
	}
}

func TestClipByID(t *testing.T) {
	m := sample(t)
	c := m.ClipByID("c-015")
	if c == nil {
		t.Fatal("ClipByID(c-015) = nil")
	}
	// Pointer must alias the slice element so mutations persist.
	c.Review.Rating = 4
	if m.Clips[1].Review.Rating != 4 {
		t.Error("ClipByID did not return an aliasing pointer")
	}
	if m.ClipByID("nope") != nil {
		t.Error("ClipByID(nope) should be nil")
	}
}

func TestValidateSchema(t *testing.T) {
	tests := []struct {
		name    string
		mutate  func(*Manifest)
		wantErr bool
	}{
		{"valid", func(*Manifest) {}, false},
		{"bad schema version", func(m *Manifest) { m.SchemaVersion = 99 }, true},
		{"bad status", func(m *Manifest) { m.Clips[0].Review.Status = "maybe" }, true},
		{"none without note", func(m *Manifest) {
			m.Clips[1].ProxyInfo = ProxyInfo{Source: ProxyNone, Note: ""}
		}, true},
		{"bad proxy source", func(m *Manifest) { m.Clips[0].ProxyInfo.Source = "weird" }, true},
		{"duplicate id", func(m *Manifest) { m.Clips[1].ID = m.Clips[0].ID }, true},
		{"empty id", func(m *Manifest) { m.Clips[0].ID = "" }, true},
		{"rating out of range", func(m *Manifest) { m.Clips[0].Review.Rating = 6 }, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := sample(t)
			tt.mutate(m)
			err := m.ValidateSchema()
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateSchema() err = %v, wantErr = %v", err, tt.wantErr)
			}
		})
	}
}

func TestLoadRejectsUnknownFields(t *testing.T) {
	dir := t.TempDir()
	bad := `{"schemaVersion":1,"shoot":{},"clips":[],"archivedAt":null,"archivePath":"","mystery":true}`
	if err := writeFile(dir, FileName, bad); err != nil {
		t.Fatal(err)
	}
	if _, err := LoadFile(filepath.Join(dir, FileName)); err == nil {
		t.Error("expected error for unknown field, got nil")
	}
}

func TestValidatePathsMissingFile(t *testing.T) {
	dir := t.TempDir()
	m := sample(t)
	// Do not create referenced files; ValidatePaths must fail.
	if err := m.ValidatePaths(dir); err == nil {
		t.Error("expected missing-file error, got nil")
	}
}
