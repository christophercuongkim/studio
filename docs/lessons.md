# Lessons

Accumulated gotchas from building `studio`. Format: **rule** + _Why_ + _How to
apply_. Short, specific, non-obvious — merge duplicates, skip generic advice.
Newest milestone last.

---

## CHR-5 — Nix flake + Go bootstrap

**A repo with zero commits can't be branched or PR'd — establish the default
branch on the remote first.**
_Why:_ `gh pr create` needs a base branch that exists on the remote; you can't
rebase/branch from nothing. Pushing the seed commit straight to the default
branch may also be blocked by push policy.
_How to apply:_ Make one seed commit, have the human push it to `main` (or fold
the seed into the first feature PR), then branch every milestone off `main`.

**Nix flakes only see git-tracked files.**
_Why:_ `nix flake lock` / `nix eval` on an untracked `flake.nix` fails with
"not tracked by Git" — flakes read the git tree, not the working dir.
_How to apply:_ `git add flake.nix` (staging is enough; commit can come later)
before running any `nix` command against it.

**`go run` collapses every non-zero exit to 1 — verify exact exit codes with a
built binary.**
_Why:_ Our CLI contract is 0 success / 1 command error / 2 usage error, but
`go run ./cmd/studio bogus` always reports exit 1 regardless.
_How to apply:_ `go build -o /tmp/x ./cmd/studio && /tmp/x bogus; echo $?` when
asserting exit codes by hand; the unit test calls `cli.Run` directly.

---

## CHR-6 — config + manifest + probe

**ffprobe emits `r_frame_rate: "0/0"` for audio and data streams.**
_Why:_ Naively parsing the first stream's rate, or dividing by the denominator,
yields garbage or a divide-by-zero.
_How to apply:_ Pick the first `codec_type == "video"` stream explicitly, and
have the rational parser reject a zero numerator or denominator.

**Store frame rate as an exact rational; render the decimal only at the edge.**
_Why:_ 29.97 and 23.976 are lossy decimals; downstream frame math (chapters,
frame-step) needs the exact `30000/1001`.
_How to apply:_ Probe keeps `FPSNum/FPSDen`; the manifest's display string comes
from a `FPS()` formatter. Don't round-trip through float.

**yaml.v3 can't emit comments, so a commented defaults file must be
hand-authored — and can silently drift from the struct.**
_Why:_ We want the first-run `config.yaml` to be self-documenting, but
`yaml.Marshal(Default())` produces a bare, comment-less file.
_How to apply:_ Keep a hand-written `defaultYAML` const next to `Default()`; a
test asserts the comment header survives a load. Changing defaults means editing
both — there is no compiler check tying them together.

**Split schema validation from filesystem/path validation.**
_Why:_ Read-only tooling (search, M7) needs to parse a manifest whose tree has
moved without erroring on every missing file; ingest/serve want the full check.
_How to apply:_ `ValidateSchema()` (pure) vs `ValidatePaths(root)` (touches
disk). `Load` runs both; `LoadFile` runs only the former.

---

## Process (applies to every milestone)

**Before cutting a milestone PR: `go build ./...`, `go vet ./...`,
`gofmt -l .`, and full `go test ./...` (including integration), then exercise
the real flow locally.**
_Why:_ Tests passing isn't the same as the command working end-to-end against
real media/binaries.
_How to apply:_ Run the built binary through the milestone's actual path (real
ffprobe, a real project folder) before opening the PR — not just `go run`.

**Each milestone = one PR of atomic commits, auto-merged after self-review;
then a follow-up docs PR appending lessons here.**
_Why:_ Keeps history readable (one logical change per commit) and captures
gotchas while they're fresh.
_How to apply:_ Rebase-merge to keep the atomic commits linear; doc-only PRs
(this file) self-merge per the working agreement.
