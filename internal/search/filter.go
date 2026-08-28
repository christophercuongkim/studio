package search

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

// Filters narrows a record set. Zero-valued fields are inactive.
type Filters struct {
	Terms     []string // lowercased; all must match (AND) desc + shoot title
	MinRating int      // 0 = off
	Status    string   // "" = off
	Cam       string   // "" = off (case-insensitive exact)
	Since     time.Time
	UntilEnd  time.Time // exclusive upper bound
	MinDur    float64   // <0 = off
	MaxDur    float64   // <0 = off
	Shoot     string    // substring on shoot title
}

// NewFilters returns Filters with the duration bounds disabled (-1). Use this
// rather than a zero value, whose MaxDur of 0 would reject everything.
func NewFilters() Filters {
	return Filters{MinDur: -1, MaxDur: -1}
}

// Match reports whether a record passes every active filter.
func (f Filters) Match(r Record) bool {
	if len(f.Terms) > 0 {
		hay := strings.ToLower(r.Desc + " " + r.Shoot)
		for _, term := range f.Terms {
			if !strings.Contains(hay, term) {
				return false
			}
		}
	}
	if f.MinRating > 0 && r.Rating < f.MinRating {
		return false
	}
	if f.Status != "" && r.Status != f.Status {
		return false
	}
	if f.Cam != "" && !strings.EqualFold(r.Cam, f.Cam) {
		return false
	}
	if !f.Since.IsZero() && r.CreatedAt.Before(f.Since) {
		return false
	}
	if !f.UntilEnd.IsZero() && !r.CreatedAt.Before(f.UntilEnd) {
		return false
	}
	if f.MinDur >= 0 && r.DurationSec < f.MinDur {
		return false
	}
	if f.MaxDur >= 0 && r.DurationSec > f.MaxDur {
		return false
	}
	if f.Shoot != "" && !strings.Contains(strings.ToLower(r.Shoot), strings.ToLower(f.Shoot)) {
		return false
	}
	return true
}

// Apply returns the records passing the filters, sorted.
func Apply(records []Record, f Filters) []Record {
	var out []Record
	for _, r := range records {
		if f.Match(r) {
			out = append(out, r)
		}
	}
	Sort(out)
	return out
}

// ParseRating parses a "4+" or "4" rating filter into a minimum rating.
func ParseRating(s string) (int, error) {
	s = strings.TrimSpace(s)
	s = strings.TrimSuffix(s, "+")
	n, err := strconv.Atoi(s)
	if err != nil || n < 0 || n > 5 {
		return 0, fmt.Errorf("invalid --rating %q (want 0–5, optional +)", s)
	}
	return n, nil
}

// ParsePeriodStart parses YYYY, YYYY-MM, or YYYY-MM-DD into the start instant of
// that period (for --since).
func ParsePeriodStart(s string) (time.Time, error) {
	t, _, err := parsePeriod(s)
	return t, err
}

// ParsePeriodEnd parses the same forms into the exclusive end instant (for
// --until: the first moment after the period).
func ParsePeriodEnd(s string) (time.Time, error) {
	_, end, err := parsePeriod(s)
	return end, err
}

func parsePeriod(s string) (start, end time.Time, err error) {
	s = strings.TrimSpace(s)
	switch len(s) {
	case 4: // YYYY
		t, e := time.Parse("2006", s)
		if e != nil {
			return start, end, e
		}
		return t, t.AddDate(1, 0, 0), nil
	case 7: // YYYY-MM
		t, e := time.Parse("2006-01", s)
		if e != nil {
			return start, end, e
		}
		return t, t.AddDate(0, 1, 0), nil
	case 10: // YYYY-MM-DD
		t, e := time.Parse("2006-01-02", s)
		if e != nil {
			return start, end, e
		}
		return t, t.AddDate(0, 0, 1), nil
	default:
		return start, end, fmt.Errorf("invalid date %q (want YYYY, YYYY-MM, or YYYY-MM-DD)", s)
	}
}

// ParseDurRange parses "5s..2m", "..2m", or "30s.." into min/max seconds. A
// missing bound is returned as -1 (inactive).
func ParseDurRange(s string) (min, max float64, err error) {
	lo, hi, found := strings.Cut(s, "..")
	if !found {
		return -1, -1, fmt.Errorf("invalid --dur %q (want lo..hi, e.g. 5s..2m)", s)
	}
	min, max = -1, -1
	if strings.TrimSpace(lo) != "" {
		d, e := time.ParseDuration(strings.TrimSpace(lo))
		if e != nil {
			return -1, -1, fmt.Errorf("invalid --dur lower bound %q: %w", lo, e)
		}
		min = d.Seconds()
	}
	if strings.TrimSpace(hi) != "" {
		d, e := time.ParseDuration(strings.TrimSpace(hi))
		if e != nil {
			return -1, -1, fmt.Errorf("invalid --dur upper bound %q: %w", hi, e)
		}
		max = d.Seconds()
	}
	return min, max, nil
}
