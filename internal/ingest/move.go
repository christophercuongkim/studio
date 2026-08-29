package ingest

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"syscall"
)

// copyChunk is the read/write unit for a byte copy. It bounds how long a copy
// can run between cancellation checks — a large 4K original is copied in ~4 MiB
// slices so a Ctrl-C aborts within one slice instead of blocking for minutes
// inside a single io.Copy.
const copyChunk = 4 << 20

// moveFile relocates src to dst. It tries os.Rename first (instant, same
// filesystem) and falls back to copy+fsync+verify-size+remove when the rename
// crosses a filesystem boundary (EXDEV). forceCopy (copy-always ingest) always
// copies and leaves the source in place. The copy is cancellable via ctx; on
// cancellation the partial destination is removed and ctx.Err() is returned.
func moveFile(ctx context.Context, src, dst string, forceCopy bool) error {
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
	if err := copyFile(ctx, src, dst); err != nil {
		return err
	}
	if forceCopy {
		return nil
	}
	return os.Remove(src)
}

// copyFile copies src to dst in bounded chunks, fsyncs, and verifies the byte
// count matches. It checks ctx before each chunk so a cancelled copy stops
// promptly (mid-file) rather than blocking inside one large read/write; the
// partial destination is deleted on any failure, including cancellation.
func copyFile(ctx context.Context, src, dst string) error {
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

	fail := func(err error) error {
		out.Close()
		os.Remove(dst)
		return err
	}

	buf := make([]byte, copyChunk)
	var written int64
	for {
		if err := ctx.Err(); err != nil {
			return fail(err) // cancelled mid-copy; leave no partial behind
		}
		n, rerr := in.Read(buf)
		if n > 0 {
			if _, werr := out.Write(buf[:n]); werr != nil {
				return fail(fmt.Errorf("copy %s: %w", src, werr))
			}
			written += int64(n)
		}
		if rerr != nil {
			if rerr == io.EOF {
				break
			}
			return fail(fmt.Errorf("copy %s: %w", src, rerr))
		}
	}
	if err := out.Sync(); err != nil {
		return fail(fmt.Errorf("fsync %s: %w", dst, err))
	}
	if err := out.Close(); err != nil {
		return err
	}
	if written != si.Size() {
		os.Remove(dst)
		return fmt.Errorf("copy %s: wrote %d bytes, expected %d", src, written, si.Size())
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
