package ingest

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"testing"
)

// TestCopyFileHappyPath copies a file and verifies the bytes land intact.
func TestCopyFileHappyPath(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "src.bin")
	dst := filepath.Join(dir, "dst.bin")
	want := bytes.Repeat([]byte("studio"), 2*copyChunk/6) // spans several chunks
	if err := os.WriteFile(src, want, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := copyFile(context.Background(), src, dst); err != nil {
		t.Fatalf("copyFile: %v", err)
	}
	got, err := os.ReadFile(dst)
	if err != nil {
		t.Fatalf("read dst: %v", err)
	}
	if !bytes.Equal(got, want) {
		t.Errorf("copied bytes differ: got %d bytes, want %d", len(got), len(want))
	}
}

// TestCopyFileCancelledLeavesNoPartial is the core of the freeze fix: a copy
// under a cancelled context aborts and never leaves a partial destination
// behind (which would otherwise block a later --append with a "destination
// already exists" error).
func TestCopyFileCancelledLeavesNoPartial(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "src.bin")
	dst := filepath.Join(dir, "dst.bin")
	if err := os.WriteFile(src, bytes.Repeat([]byte{0xab}, copyChunk*3), 0o644); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // cancelled before the first chunk

	err := copyFile(ctx, src, dst)
	if err != context.Canceled {
		t.Fatalf("copyFile err = %v, want context.Canceled", err)
	}
	if _, err := os.Stat(dst); !os.IsNotExist(err) {
		t.Errorf("partial destination left behind (stat err = %v); it should be removed", err)
	}
}
