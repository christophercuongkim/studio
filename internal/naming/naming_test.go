package naming

import (
	"testing"
	"time"
)

func TestFinalStem(t *testing.T) {
	created := time.Date(2026, 8, 27, 14, 23, 1, 0, time.UTC)
	take2 := 2
	tests := []struct {
		name string
		cam  string
		seq  int
		desc string
		take *int
		want string
	}{
		{"no take", "DJI", 14, "sunset-dock-wide", nil, "20260827_DJI014_sunset-dock-wide"},
		{"with take", "A", 3, "intro", &take2, "20260827_A003_intro_t2"},
		{"pads seq", "B", 7, "x", nil, "20260827_B007_x"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := FinalStem(created, tt.cam, tt.seq, tt.desc, tt.take); got != tt.want {
				t.Errorf("FinalStem = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestValidateDesc(t *testing.T) {
	long := ""
	for range MaxDescLen + 1 {
		long += "a"
	}
	tests := []struct {
		desc    string
		wantErr bool
	}{
		{"sunset-dock-wide", false},
		{"", false},
		{"abc123", false},
		{"Has-Caps", true},
		{"has space", true},
		{"under_score", true},
		{"emoji😀", true},
		{long, true},
	}
	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			err := ValidateDesc(tt.desc)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateDesc(%q) err = %v, wantErr %v", tt.desc, err, tt.wantErr)
			}
		})
	}
}
