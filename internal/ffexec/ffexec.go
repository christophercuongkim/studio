// Package ffexec runs the external ffmpeg/ffprobe binaries and surfaces their
// failures usefully. studio never links libav — every media operation shells
// out through here (plan invariant §2, error rules §17).
package ffexec

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os/exec"
	"strings"
)

// ErrNotFound reports that a required external binary is missing from PATH.
type ErrNotFound struct{ Name string }

func (e *ErrNotFound) Error() string {
	return fmt.Sprintf("%s not found on PATH (install ffmpeg >= 6, e.g. via the flake devShell)", e.Name)
}

// Output runs name with args and returns its stdout and stderr separately.
// stderr is returned even on success because some ffmpeg filters (e.g.
// loudnorm's JSON block, used by qc) print their payload there.
//
// On a non-zero exit the returned error embeds the last ~10 lines of stderr so
// callers get an actionable message without the full ffmpeg firehose.
func Output(ctx context.Context, name string, args ...string) (stdout, stderr []byte, err error) {
	if _, lookErr := exec.LookPath(name); lookErr != nil {
		return nil, nil, &ErrNotFound{Name: name}
	}

	var outBuf, errBuf bytes.Buffer
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Stdout = &outBuf
	cmd.Stderr = &errBuf

	runErr := cmd.Run()
	stdout, stderr = outBuf.Bytes(), errBuf.Bytes()
	if runErr != nil {
		return stdout, stderr, fmt.Errorf("%s failed: %w\n%s", name, runErr, tail(stderr, 10))
	}
	return stdout, stderr, nil
}

// Run is Output for the common case where only stdout matters.
func Run(ctx context.Context, name string, args ...string) ([]byte, error) {
	stdout, _, err := Output(ctx, name, args...)
	return stdout, err
}

// tail returns the last n non-empty lines of b, indented for readability.
func tail(b []byte, n int) string {
	lines := strings.Split(strings.TrimRight(string(b), "\n"), "\n")
	nonEmpty := lines[:0]
	for _, l := range lines {
		if strings.TrimSpace(l) != "" {
			nonEmpty = append(nonEmpty, l)
		}
	}
	if len(nonEmpty) > n {
		nonEmpty = nonEmpty[len(nonEmpty)-n:]
	}
	if len(nonEmpty) == 0 {
		return "  (no stderr output)"
	}
	return "  " + strings.Join(nonEmpty, "\n  ")
}

// IsNotFound reports whether err is an ErrNotFound (missing binary).
func IsNotFound(err error) bool {
	var e *ErrNotFound
	return errors.As(err, &e)
}
