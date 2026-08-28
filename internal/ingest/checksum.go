package ingest

import (
	"encoding/hex"
	"fmt"
	"io"
	"os"

	"github.com/cespare/xxhash/v2"
)

// checksumBufSize is the streaming read buffer (1 MiB) used for hashing large
// media files without loading them into memory (plan §6 step 4).
const checksumBufSize = 1 << 20

// xxh64File streams a file through xxHash64 and returns the digest as a
// lowercase hex string (16 chars), matching the manifest's media.xxh64 field.
func xxh64File(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()

	h := xxhash.New()
	buf := make([]byte, checksumBufSize)
	if _, err := io.CopyBuffer(h, f, buf); err != nil {
		return "", fmt.Errorf("hash %s: %w", path, err)
	}
	var out [8]byte
	sum := h.Sum64()
	for i := range 8 {
		out[7-i] = byte(sum >> (8 * i))
	}
	return hex.EncodeToString(out[:]), nil
}
