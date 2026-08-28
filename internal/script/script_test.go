package script

import (
	"strings"
	"testing"
)

const good = `---
title: "Why I ported ani-cli to Go"
targetLength: 8m
---

## Hook
- cold open: the 3-line demo
- "every anime CLI is bash. mine isn't"

## The problem
- bash scraping breaks weekly
  - example: the CSS selector churn
  - example: cloudflare

## Outro
- sub tease
`

func TestParseGood(t *testing.T) {
	s, err := Parse([]byte(good))
	if err != nil {
		t.Fatal(err)
	}
	if s.Title != "Why I ported ani-cli to Go" || s.TargetLength != "8m" {
		t.Errorf("frontmatter wrong: %+v", s)
	}
	if len(s.Sections) != 3 {
		t.Fatalf("got %d sections, want 3", len(s.Sections))
	}
	if s.Sections[0].Title != "Hook" || len(s.Sections[0].Bullets) != 2 {
		t.Errorf("Hook wrong: %+v", s.Sections[0])
	}
	prob := s.Sections[1]
	if len(prob.Bullets) != 1 || len(prob.Bullets[0].Children) != 2 {
		t.Errorf("nested bullets wrong: %+v", prob.Bullets)
	}
	if got := s.SectionTitles(); strings.Join(got, ",") != "Hook,The problem,Outro" {
		t.Errorf("section titles = %v", got)
	}
}

func TestParseErrors(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{"bullet before section", "- orphan\n", "before any"},
		{"deep nesting", "## S\n- a\n    - too deep\n", "expected"},
		{"garbage line", "## S\nnot a bullet\n", "expected"},
		{"unclosed frontmatter", "---\ntitle: x\n## S\n", "never closed"},
		{"no sections", "- a\n", "before any"},
		{"empty", "\n\n", "no '## ' sections"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := Parse([]byte(tt.in))
			if err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Errorf("Parse(%q) err = %v, want containing %q", tt.in, err, tt.want)
			}
		})
	}
}

func TestParseReportsLineNumbers(t *testing.T) {
	// The bad line is line 4 (1: ##, 2: -, 3: blank, 4: garbage).
	in := "## S\n- ok\n\ngarbage here\n"
	_, err := Parse([]byte(in))
	if err == nil || !strings.Contains(err.Error(), "line 4") {
		t.Errorf("expected line 4 in error, got %v", err)
	}
}
