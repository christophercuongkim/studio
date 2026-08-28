package ingest

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"syscall"
)

// moveFile relocates src to dst. It tries os.Rename first (instant, same
// filesystem) and falls back to copy+fsync+verify-size+remove when the rename
// crosses a filesystem boundary (EXDEV). forceCopy (from --copy) always copies
// and leaves the source in place.
func moveFile(src, dst string, forceCopy bool) error {
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return err
	}
	if !forceCopy {
		err := os.Rename(src, dst)
		if err == nil {
			return nil
		}
		if !errors.Is(err, syscall.EXDEV) {
			return fmt.Errorf("move %s: %w", src, err)
		}
		// Cross-device: fall through to copy, then remove the source.
	}
	if err := copyFile(src, dst); err != nil {
		return err
	}
	if forceCopy {
		return nil
	}
	return os.Remove(src)
}

// copyFile copies src to dst, fsyncs, and verifies the byte count matches.
func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	si, err := in.Stat()
	if err != nil {
		return err
	}

	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	n, err := io.Copy(out, in)
	if err != nil {
		out.Close()
		os.Remove(dst)
		return fmt.Errorf("copy %s: %w", src, err)
	}
	if err := out.Sync(); err != nil {
		out.Close()
		os.Remove(dst)
		return fmt.Errorf("fsync %s: %w", dst, err)
	}
	if err := out.Close(); err != nil {
		return err
	}
	if n != si.Size() {
		os.Remove(dst)
		return fmt.Errorf("copy %s: wrote %d bytes, expected %d", src, n, si.Size())
	}
	return nil
}

// collision is a pair of source files that would flatten to the same
// destination name.
type collision struct {
	dst  string
	srcA string
	srcB string
}

// detectCollisions reports source files that would land on the same
// destination when subdirectories are flattened into originals/ (plan §6:
// flattened-name collisions are fatal, detected before anything moves).
//
// dests maps intended destination path → source path, built by the caller. It
// returns every distinct clash, sorted, so the user sees all of them at once.
func detectCollisions(pairs []movePair) []collision {
	byDst := map[string]string{}
	var clashes []collision
	// Stable input order for deterministic "first" src.
	sort.Slice(pairs, func(i, j int) bool { return pairs[i].src < pairs[j].src })
	for _, p := range pairs {
		if prev, ok := byDst[p.dst]; ok {
			clashes = append(clashes, collision{dst: p.dst, srcA: prev, srcB: p.src})
			continue
		}
		byDst[p.dst] = p.src
	}
	sort.Slice(clashes, func(i, j int) bool { return clashes[i].dst < clashes[j].dst })
	return clashes
}

// movePair is one planned relocation.
type movePair struct {
	src string
	dst string
}
