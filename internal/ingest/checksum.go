package ingest

import "github.com/christophercuongkim/studio/internal/hash"

// xxh64File streams a file through xxHash64 (plan §6 step 4). It delegates to
// internal/hash so ingest (compute) and archive (verify) share one format.
func xxh64File(path string) (string, error) {
	return hash.XXH64File(path)
}
