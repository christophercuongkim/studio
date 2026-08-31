// Package naming builds and validates final clip names from the filename
// template (plan §3.2). Both `serve` (live preview) and `apply` (the rename
// engine) depend on it, so the template lives in exactly one place.
package naming

import (
	"fmt"
	"regexp"
	"strings"
	"time"
)

// MaxDescLen is the maximum length of the user-supplied description slug.
const MaxDescLen = 40

// descRe is the allowed charset for a description slug: lowercase letters,
// digits, and hyphens (plan §3.2). The review UI normalizes to this; the server
// validates against it.
var descRe = regexp.MustCompile(`^[a-z0-9-]*$`)

// ValidateDesc reports whether desc is a legal description slug.
func ValidateDesc(desc string) error {
	if len(desc) > MaxDescLen {
		return fmt.Errorf("description too long (%d > %d chars)", len(desc), MaxDescLen)
	}
	if !descRe.MatchString(desc) {
		return fmt.Errorf("description must match [a-z0-9-] (got %q)", desc)
	}
	return nil
}

// MaxGroupLen bounds a whole bin-folder path (all segments plus separators).
const MaxGroupLen = 120

// groupSegRe is the allowed charset for one bin-folder path segment. Mixed case
// is allowed (these are Kdenlive display names), but no spaces — a group is
// typeable as a single CLI token — and each segment must start alphanumeric so a
// name is never blank or punctuation-only.
var groupSegRe = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9_-]*$`)

// ValidateGroup reports whether group is a legal bin-folder path. A path may nest
// with "/" (e.g. "london/b_roll") to arbitrary depth; each "/"-separated segment
// must match groupSegRe. Empty is legal and means "ungrouped" (routed to the
// default bin).
func ValidateGroup(group string) error {
	if group == "" {
		return nil
	}
	if len(group) > MaxGroupLen {
		return fmt.Errorf("group too long (%d > %d chars)", len(group), MaxGroupLen)
	}
	for _, seg := range strings.Split(group, "/") {
		if !groupSegRe.MatchString(seg) {
			return fmt.Errorf("group segment must match [A-Za-z0-9_-] and start alphanumeric (got %q in %q)", seg, group)
		}
	}
	return nil
}

// FinalStem builds the final filename stem (no extension) for a clip:
//
//	{date}_{cam}{seq}_{desc}[_t{take}]
//
// date is YYYYMMDD from created, cam is the camera code, seq is zero-padded to
// three digits, and take is appended only when non-nil. The caller is
// responsible for having validated desc.
func FinalStem(created time.Time, cam string, seq int, desc string, take *int) string {
	stem := fmt.Sprintf("%s_%s%03d_%s", created.Format("20060102"), cam, seq, desc)
	if take != nil {
		stem += fmt.Sprintf("_t%d", *take)
	}
	return stem
}
