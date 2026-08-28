// Package kdenlive parses and edits .kdenlive project files. Kdenlive documents
// are MLT XML; this package models them as a tolerant generic node tree so we
// can inject producers (scaffold) and read timeline guides (chapters) without a
// schema, round-tripping everything we don't touch.
//
// The parser fails loudly on anything that isn't an <mlt> document (plan
// invariant: never silently mangle an untested Kdenlive version).
package kdenlive

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"strings"
)

// Node is a generic XML element: a tag, ordered attributes, character content,
// and children. It carries no schema, so unknown Kdenlive/MLT constructs pass
// through untouched.
type Node struct {
	Name     string
	Attrs    []xml.Attr
	Text     string // trimmed character data (property values, etc.)
	Children []*Node
}

// Load parses a .kdenlive document. It requires an <mlt> root element.
func Load(data []byte) (*Node, error) {
	dec := xml.NewDecoder(bytes.NewReader(data))
	var root *Node
	stack := []*Node{}

	for {
		tok, err := dec.Token()
		if err != nil {
			if err.Error() == "EOF" {
				break
			}
			return nil, fmt.Errorf("parse kdenlive xml: %w", err)
		}
		switch t := tok.(type) {
		case xml.StartElement:
			n := &Node{Name: t.Name.Local, Attrs: cloneAttrs(t.Attr)}
			if len(stack) > 0 {
				parent := stack[len(stack)-1]
				parent.Children = append(parent.Children, n)
			} else {
				root = n
			}
			stack = append(stack, n)
		case xml.EndElement:
			if len(stack) > 0 {
				stack = stack[:len(stack)-1]
			}
		case xml.CharData:
			if len(stack) > 0 {
				cur := stack[len(stack)-1]
				cur.Text += string(t)
			}
		}
	}

	if root == nil {
		return nil, fmt.Errorf("empty or invalid kdenlive document")
	}
	if root.Name != "mlt" {
		return nil, fmt.Errorf("not a kdenlive/MLT document: root element is <%s>, expected <mlt> (untested format — refusing to mangle it)", root.Name)
	}
	// Trim whitespace-only text collected between child elements.
	trimText(root)
	return root, nil
}

func cloneAttrs(a []xml.Attr) []xml.Attr {
	if len(a) == 0 {
		return nil
	}
	out := make([]xml.Attr, len(a))
	copy(out, a)
	return out
}

// trimText clears text that is only inter-element whitespace, keeping real
// leaf values (e.g. a <property> body).
func trimText(n *Node) {
	if len(n.Children) > 0 {
		n.Text = ""
	} else {
		n.Text = strings.TrimSpace(n.Text)
	}
	for _, c := range n.Children {
		trimText(c)
	}
}

// Attr returns the value of the named attribute and whether it was present.
func (n *Node) Attr(name string) (string, bool) {
	for _, a := range n.Attrs {
		if a.Name.Local == name {
			return a.Value, true
		}
	}
	return "", false
}

// SetAttr sets (or adds) an attribute, preserving order for existing ones.
func (n *Node) SetAttr(name, value string) {
	for i, a := range n.Attrs {
		if a.Name.Local == name {
			n.Attrs[i].Value = value
			return
		}
	}
	n.Attrs = append(n.Attrs, xml.Attr{Name: xml.Name{Local: name}, Value: value})
}

// Find returns the first descendant (depth-first) element with the given tag
// whose attribute attr equals value. If attr is "", matches on tag alone.
func (n *Node) Find(tag, attr, value string) *Node {
	for _, c := range n.Children {
		if c.Name == tag && (attr == "" || attrEquals(c, attr, value)) {
			return c
		}
		if got := c.Find(tag, attr, value); got != nil {
			return got
		}
	}
	return nil
}

func attrEquals(n *Node, attr, value string) bool {
	v, ok := n.Attr(attr)
	return ok && v == value
}

// Render serializes the tree back to indented XML with an XML declaration.
func (n *Node) Render() []byte {
	var b bytes.Buffer
	b.WriteString("<?xml version='1.0' encoding='utf-8'?>\n")
	n.render(&b, 0)
	return b.Bytes()
}

func (n *Node) render(b *bytes.Buffer, depth int) {
	indent := strings.Repeat(" ", depth)
	b.WriteString(indent)
	b.WriteByte('<')
	b.WriteString(n.Name)
	for _, a := range n.Attrs {
		fmt.Fprintf(b, " %s=%q", a.Name.Local, a.Value)
	}
	if len(n.Children) == 0 && n.Text == "" {
		b.WriteString("/>\n")
		return
	}
	b.WriteByte('>')
	if len(n.Children) == 0 {
		b.WriteString(escape(n.Text))
		b.WriteString("</")
		b.WriteString(n.Name)
		b.WriteString(">\n")
		return
	}
	b.WriteByte('\n')
	for _, c := range n.Children {
		c.render(b, depth+1)
	}
	b.WriteString(indent)
	b.WriteString("</")
	b.WriteString(n.Name)
	b.WriteString(">\n")
}

func escape(s string) string {
	var b bytes.Buffer
	xml.EscapeText(&b, []byte(s))
	return b.String()
}
