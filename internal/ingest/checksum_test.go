package ingest

import (
	"os"
	"path/filepath"
	"testing"
)

func TestXXH64File(t *testing.T) {
	dir := t.TempDir()

	empty := filepath.Join(dir, "empty")
	if err := os.WriteFile(empty, nil, 0o644); err != nil {
		t.Fatal(err)
	}
	// xxHash64 of the empty input (seed 0) is a fixed, well-known value.
	if got, err := xxh64File(empty); err != nil || got != "ef46db3751d8e999" {
		t.Fatalf("xxh64(empty) = %q, %v; want ef46db3751d8e999", got, err)
	}

	// Same content hashes identically; different content differs.
	a := filepath.Join(dir, "a")
	b := filepath.Join(dir, "b")
	c := filepath.Join(dir, "c")
	os.WriteFile(a, []byte("hello world"), 0o644)
	os.WriteFile(b, []byte("hello world"), 0o644)
	os.WriteFile(c, []byte("hello worlD"), 0o644)
	ha, _ := xxh64File(a)
	hb, _ := xxh64File(b)
	hc, _ := xxh64File(c)
	if ha != hb {
		t.Errorf("identical files hashed differently: %s vs %s", ha, hb)
	}
	if ha == hc {
		t.Errorf("different files hashed the same: %s", ha)
	}
}
