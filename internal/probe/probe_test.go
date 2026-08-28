package probe

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestParseFixtures(t *testing.T) {
	tests := []struct {
		file        string
		wantW       int
		wantH       int
		wantCodec   string
		wantAudio   bool
		wantDur     float64
		wantSize    int64
		wantFPS     string
		wantCreated string // RFC3339, "" if none expected
	}{
		{
			file: "dji-hevc.json", wantW: 3840, wantH: 2160, wantCodec: "hevc",
			wantAudio: true, wantDur: 42.358667, wantSize: 512034441, wantFPS: "29.97",
			wantCreated: "2026-08-27T14:23:01Z",
		},
		{
			file: "gopro-h264.json", wantW: 1920, wantH: 1080, wantCodec: "h264",
			wantAudio: true, wantDur: 12.362350, wantSize: 98230011, wantFPS: "59.94",
			wantCreated: "2026-07-04T09:15:42Z",
		},
		{
			file: "no-audio.json", wantW: 1280, wantH: 720, wantCodec: "h264",
			wantAudio: false, wantDur: 8.0, wantSize: 4194304, wantFPS: "25",
			wantCreated: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.file, func(t *testing.T) {
			data, err := os.ReadFile(filepath.Join("testdata", tt.file))
			if err != nil {
				t.Fatal(err)
			}
			r, err := Parse(data)
			if err != nil {
				t.Fatalf("Parse: %v", err)
			}
			if r.Width != tt.wantW || r.Height != tt.wantH {
				t.Errorf("dims = %dx%d, want %dx%d", r.Width, r.Height, tt.wantW, tt.wantH)
			}
			if r.VCodec != tt.wantCodec {
				t.Errorf("vcodec = %q, want %q", r.VCodec, tt.wantCodec)
			}
			if r.HasAudio != tt.wantAudio {
				t.Errorf("hasAudio = %v, want %v", r.HasAudio, tt.wantAudio)
			}
			if r.DurationSec != tt.wantDur {
				t.Errorf("duration = %v, want %v", r.DurationSec, tt.wantDur)
			}
			if r.SizeBytes != tt.wantSize {
				t.Errorf("size = %d, want %d", r.SizeBytes, tt.wantSize)
			}
			if got := r.FPS(); got != tt.wantFPS {
				t.Errorf("fps = %q, want %q", got, tt.wantFPS)
			}
			if tt.wantCreated == "" {
				if !r.CreatedAt.IsZero() {
					t.Errorf("createdAt = %v, want zero", r.CreatedAt)
				}
			} else {
				want, _ := time.Parse(time.RFC3339, tt.wantCreated)
				if !r.CreatedAt.Equal(want) {
					t.Errorf("createdAt = %v, want %v", r.CreatedAt, want)
				}
			}
		})
	}
}

func TestParseRejectsNoVideo(t *testing.T) {
	_, err := Parse([]byte(`{"streams":[{"codec_type":"audio","codec_name":"aac"}],"format":{}}`))
	if err == nil {
		t.Fatal("expected error for audio-only input, got nil")
	}
}

func TestParseRejectsGarbage(t *testing.T) {
	if _, err := Parse([]byte("not json")); err == nil {
		t.Fatal("expected parse error, got nil")
	}
}
