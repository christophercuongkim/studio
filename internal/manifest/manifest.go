// Package manifest defines manifest.json — the per-project footage record that
// every command reads and writes (plan §4). It is the file-based state that
// replaces a database: load it, mutate it, save it atomically.
package manifest

import (
	"time"
)

// SchemaVersion is the current on-disk schema. The loader rejects anything it
// doesn't recognize rather than guessing (plan §4). Bump this when the shape
// changes in a breaking way.
const SchemaVersion = 1

// FileName is the manifest's fixed name within a project folder.
const FileName = "manifest.json"

// Review status values.
const (
	StatusPending  = "pending"
	StatusKept     = "kept"
	StatusRejected = "rejected"
)

// Proxy source values.
const (
	ProxyCamera    = "camera"    // adopted from a camera-provided low-res file
	ProxyGenerated = "generated" // transcoded by studio
	ProxyNone      = "none"      // no proxy; Note must explain why
)

// Manifest is the top-level manifest.json document.
type Manifest struct {
	SchemaVersion int        `json:"schemaVersion"`
	Shoot         Shoot      `json:"shoot"`
	Clips         []Clip     `json:"clips"`
	ArchivedAt    *time.Time `json:"archivedAt"`
	ArchivePath   string     `json:"archivePath"`
}

// Shoot describes the project as a whole.
type Shoot struct {
	Root      string    `json:"root"` // absolute path to the project folder
	Title     string    `json:"title"`
	CreatedAt time.Time `json:"createdAt"`
	CamCode   string    `json:"camCode"`
}

// Clip is one footage group: an original plus its proxy and sidecars, all
// sharing a stem.
type Clip struct {
	ID        string    `json:"id"`   // stable for the manifest's lifetime
	Seq       int       `json:"seq"`  // ingest-order sequence within the project
	Stem      string    `json:"stem"` // current filename stem (pre-apply)
	Files     Files     `json:"files"`
	Media     Media     `json:"media"`
	ProxyInfo ProxyInfo `json:"proxyInfo"`
	Review    Review    `json:"review"`
	Applied   Applied   `json:"applied"`
}

// Files are the on-disk paths of a clip's members, relative to the project root.
type Files struct {
	Original string   `json:"original"`
	Proxy    string   `json:"proxy"` // always .mp4; "" when ProxyInfo.Source == none
	Sidecars []string `json:"sidecars"`
}

// Media is the probed technical metadata of the original.
type Media struct {
	DurationSec float64   `json:"durationSec"`
	Width       int       `json:"width"`
	Height      int       `json:"height"`
	FPS         string    `json:"fps"` // decimal string, e.g. "29.97"
	VCodec      string    `json:"vcodec"`
	HasAudio    bool      `json:"hasAudio"`
	CreatedAt   time.Time `json:"createdAt"`
	SizeBytes   int64     `json:"sizeBytes"`
	XXH64       string    `json:"xxh64"`
}

// ProxyInfo records where the proxy came from.
type ProxyInfo struct {
	Source string `json:"source"` // camera | generated | none
	Note   string `json:"note"`   // required when Source == none
}

// Review is the human review state, edited by `studio serve`.
type Review struct {
	Status     string     `json:"status"` // pending | kept | rejected
	Rating     int        `json:"rating"` // 0–5
	Desc       string     `json:"desc"`   // slug fragment; charset enforced in serve
	Take       *int       `json:"take"`   // optional take number
	ReviewedAt *time.Time `json:"reviewedAt"`
}

// Applied records the outcome of `studio apply` for this clip.
type Applied struct {
	Done      bool       `json:"done"`
	FinalStem string     `json:"finalStem"`
	AppliedAt *time.Time `json:"appliedAt"`
}

// ClipByID returns a pointer to the clip with the given id, or nil. The pointer
// is into the Manifest's slice, so mutations persist through Save.
func (m *Manifest) ClipByID(id string) *Clip {
	for i := range m.Clips {
		if m.Clips[i].ID == id {
			return &m.Clips[i]
		}
	}
	return nil
}
