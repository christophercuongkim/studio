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

## CHR-7 — ingest

**`os.Rename` moving files out of the dump means `--append` on the same dump is
a no-op — dedupe is by content, for `--copy`/overlapping dumps.**
_Why:_ After a move-mode ingest the originals are gone from the dump, so
re-running finds nothing; the checksum dedupe exists for `--copy` runs or a
partially-overlapping second card.
_How to apply:_ Test append idempotency in `--copy` mode. Don't expect
move-mode re-runs to "skip existing" — there's nothing left to skip.

**ffmpeg is deterministic: identical settings produce byte-identical output and
therefore identical checksums.**
_Why:_ Two fixture clips generated with the same command hashed the same and
the xxh64 dedupe (correctly) treated them as one clip.
_How to apply:_ When a test needs distinct files, vary a real input (duration,
tone, size) — not just the filename.

**Go's `flag` package stops at the first non-flag argument.**
_Why:_ `studio ingest <dir> --project x` silently dropped `--project` because
`<dir>` ended flag parsing. Only caught by running the built binary, not by
tests calling the pipeline directly.
_How to apply:_ Use the resume-parse helper (`parseFlags`): parse, capture the
positional, parse the rest, repeat. Applies to every subcommand with both flags
and positionals.

**Don't hardcode a real command name as the "unbuilt" case in a dispatcher
test.**
_Why:_ The exit-code test used `ingest` as an example of an unimplemented
command; building ingest flipped its exit code and broke the test.
_How to apply:_ Inject a throwaway `*Command` with a nil `Run` into the registry
in the test instead of naming a real one.

**zsh does not word-split unquoted parameters (bash does).**
_Why:_ A shell test helper passed `-tag:v hvc1` as one `$5`; zsh kept it as a
single mangled ffmpeg token, so the fixture silently wasn't created and a group
lost its original.
_How to apply:_ In zsh, pass multi-token args as separate positionals or use an
array; when a fixture "disappears", list the dir before the step, don't assume.

**Bounded parallelism doesn't need a dependency.**
_Why:_ `golang.org/x/sync` latest raised the module's Go floor; errgroup was
only "optional" in the plan.
_How to apply:_ A `chan struct{}` semaphore + `sync.WaitGroup`, with each
goroutine writing only its own slice index, covers fan-out without x/sync.

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
