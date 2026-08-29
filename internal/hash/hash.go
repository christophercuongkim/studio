// Package hash provides studio's content hashing: streamed xxHash64, rendered
// as the 16-char hex string stored in the manifest. Shared by ingest (compute)
// and archive (verify) so both agree on the format.
package hash

import (
	"encoding/hex"
	"fmt"
	"io"
	"os"

	"github.com/cespare/xxhash/v2"
)

// bufSize is the streaming read buffer (1 MiB) for hashing large media without
// loading it into memory.
const bufSize = 1 << 20

// XXH64File streams a file through xxHash64 and returns the lowercase hex
// digest (16 chars), big-endian to match the manifest's media.xxh64.
func XXH64File(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()

	h := xxhash.New()
	buf := make([]byte, bufSize)
	if _, err := io.CopyBuffer(h, f, buf); err != nil {
		return "", fmt.Errorf("hash %s: %w", path, err)
	}
	return Format(h.Sum64()), nil
}

// Format renders an xxHash64 sum as studio's 16-char big-endian lowercase hex
// digest — the exact string stored in manifest media.xxh64. Callers that hash a
// stream themselves (e.g. ingest hashing bytes as it copies them) use this so
// the format never diverges from XXH64File.
func Format(sum uint64) string {
	var out [8]byte
	for i := range 8 {
		out[7-i] = byte(sum >> (8 * i))
	}
	return hex.EncodeToString(out[:])
}
