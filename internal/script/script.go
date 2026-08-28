// Package script parses script.md: YAML frontmatter plus `##` sections and `-`
// bullets with a single nesting level (plan §10.1). It is hand-rolled (no
// markdown library) and rejects anything outside that grammar with a line
// number, so mistakes surface before you're in front of the camera.
package script

import (
	"fmt"
	"strings"

	"gopkg.in/yaml.v3"
)

// Script is a parsed script.md.
type Script struct {
	Title        string
	TargetLength string
	Sections     []Section
}

// Section is a `##` heading and its bullets. Section titles double as the
// expected chapter list (shared vocabulary with chapters, plan §10.1).
type Section struct {
	Title   string   `json:"title"`
	Bullets []Bullet `json:"bullets"`
}

// Bullet is a top-level talking point with an optional single level of children.
type Bullet struct {
	Text     string   `json:"text"`
	Children []string `json:"children"`
}

type frontmatter struct {
	Title        string `yaml:"title"`
	TargetLength string `yaml:"targetLength"`
}

// Parse reads a script.md document.
func Parse(data []byte) (*Script, error) {
	lines := strings.Split(string(data), "\n")
	s := &Script{}

	body := lines
	bodyOffset := 0
	// Optional frontmatter: a leading '---' block.
	if len(lines) > 0 && strings.TrimRight(lines[0], "\r") == "---" {
		end := -1
		for i := 1; i < len(lines); i++ {
			if strings.TrimRight(lines[i], "\r") == "---" {
				end = i
				break
			}
		}
		if end < 0 {
			return nil, fmt.Errorf("line 1: frontmatter opened with '---' but never closed")
		}
		var fm frontmatter
		if err := yaml.Unmarshal([]byte(strings.Join(lines[1:end], "\n")), &fm); err != nil {
			return nil, fmt.Errorf("frontmatter: %w", err)
		}
		s.Title = fm.Title
		s.TargetLength = fm.TargetLength
		body = lines[end+1:]
		bodyOffset = end + 1
	}

	var cur *Section
	var curBullet *Bullet
	for i, raw := range body {
		lineNo := bodyOffset + i + 1
		line := strings.TrimRight(raw, "\r")
		if strings.TrimSpace(line) == "" {
			continue
		}

		switch {
		case strings.HasPrefix(line, "## "):
			s.Sections = append(s.Sections, Section{Title: strings.TrimSpace(line[3:])})
			cur = &s.Sections[len(s.Sections)-1]
			curBullet = nil

		case strings.HasPrefix(line, "- "):
			if cur == nil {
				return nil, fmt.Errorf("line %d: bullet before any '## ' section", lineNo)
			}
			cur.Bullets = append(cur.Bullets, Bullet{Text: strings.TrimSpace(line[2:])})
			curBullet = &cur.Bullets[len(cur.Bullets)-1]

		case strings.HasPrefix(line, "  - "):
			if curBullet == nil {
				return nil, fmt.Errorf("line %d: nested bullet with no parent bullet", lineNo)
			}
			curBullet.Children = append(curBullet.Children, strings.TrimSpace(line[4:]))

		default:
			return nil, fmt.Errorf("line %d: expected '## ', '- ', or '  - ', got %q", lineNo, line)
		}
	}

	if len(s.Sections) == 0 {
		return nil, fmt.Errorf("script has no '## ' sections")
	}
	return s, nil
}

// SectionTitles returns the section titles in order (the expected chapters).
func (s *Script) SectionTitles() []string {
	out := make([]string, len(s.Sections))
	for i, sec := range s.Sections {
		out[i] = sec.Title
	}
	return out
}
