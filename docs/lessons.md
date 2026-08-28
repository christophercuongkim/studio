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

## CHR-8 — serve (review UI)

**For partial PATCH, decode into `map[string]json.RawMessage`, not a struct of
pointers.**
_Why:_ A `*int` field can't tell "key absent" from "key present and null", but
the API needs `take: null` to *clear* while an omitted `take` leaves it alone.
_How to apply:_ Decode to a raw map, then per-key `Unmarshal` only the keys that
are present; nil-pointer targets then correctly distinguish null.

**Validate on the server even when the UI already normalizes.**
_Why:_ The browser lowercases/filters the desc slug, but a hand-crafted PATCH
would otherwise write an illegal name straight into the manifest.
_How to apply:_ `naming.ValidateDesc` runs in the PATCH handler; the frontend
normalization is a convenience, not the guarantee.

**A no-build embedded frontend duplicates any format string the server also
owns — treat the server copy as authoritative.**
_Why:_ The live final-name preview reimplements the naming template in JS to
avoid a round-trip per keystroke; that's a drift risk against `naming.FinalStem`.
_How to apply:_ Keep `/api/preview-name` as the source of truth; the JS mirror
is cosmetic. If the template changes, update both and lean on the server value.

**Debounced-save timer + shutdown flush: stop the timer under the lock, then
flush.**
_Why:_ A 500ms `time.AfterFunc` save can otherwise race with `Close`.
_How to apply:_ `Close` takes the mutex, stops the timer, releases, then
`Flush`; the timer callback re-locks and no-ops when not dirty. Verified by a
test that PATCHes then `Close`s and reloads from disk.

**Guard nonzero-exit commands when hand-testing servers.**
_Why:_ A verification script that ran `pkill` aborted before its assertions
because `pkill` exits 1 when nothing matched (and the shell stops on it).
_How to apply:_ Append `|| true` to `pkill`/`pgrep`/`grep` in throwaway test
scripts, or check the manifest directly instead of gating on the kill.

---

## CHR-9 — apply + undo

**After a partial apply failure the manifest is intentionally path-inconsistent,
so undo must load it schema-only.**
_Why:_ apply renames files but only saves the manifest after *all* renames
succeed; a mid-run failure leaves originals renamed on disk while the manifest
still points at the old paths. `manifest.Load` (which validates paths) then
rejects it — and undo is exactly the tool meant to run in that state.
_How to apply:_ `undo` uses `manifest.LoadFile` (schema-only). Don't "fix" it to
`Load`; that reintroduces the bug. Found only because a test asserted the
post-failure state.

**Journal the intent (write + fsync) before each rename, and commit the manifest
only after the whole plan succeeds.**
_Why:_ A crash between "renamed on disk" and "recorded in manifest" must be
recoverable; the journal is the source of truth for undo, not the manifest.
_How to apply:_ Per file: encode the rename line, `Sync()`, then `os.Rename`.
Update+save the manifest once, at the end. Never save a half-applied manifest.

**Duplicate-final-name detection must include already-applied clips, not just
the current batch.**
_Why:_ A new kept clip can collide with a name already on disk from an earlier
apply, which a peers-only check misses.
_How to apply:_ Seed the claimed-names set from every `applied.finalStem` before
checking the batch.

**Undo skips vanished targets instead of failing.**
_Why:_ Makes undo converge whether the journal was fully or partially applied,
and re-running after a successful undo is harmless.
_How to apply:_ `Stat` the target; if gone, warn and continue rather than error.

---

## CHR-10 — scaffold (Kdenlive)

**Kdenlive 7.38 uses `<chain>`, not `<producer>`, for A/V clips — but injecting a
minimal `<producer>` still works.**
_Why:_ The bin lists clips as `<entry producer="chainN">` referencing `<chain>`
elements. A `<producer>` with `mlt_service=avformat-novalidate` + `resource` is
accepted and upgraded to a chain on first save; Kdenlive computes the rest
(hash, control_uuid) on load.
_How to apply:_ Inject `<producer>` with resource, mlt_service, kdenlive:id,
kdenlive:folderid, kdenlive:clip_type. If the GUI ever warns, add the missing
property in `scaffold.go` — the node tree makes it a one-liner.

**When you can't script the GUI's "add a clip, save, diff" step, use a real
existing project as the diff reference.**
_Why:_ The plan's procedure needs the Kdenlive GUI; headless validation isn't
available (no `melt` on PATH).
_How to apply:_ Read a real `.kdenlive` to learn the exact producer/entry/folder
format, and add a test that round-trips real project files (skip when absent).
The final "opens with zero warnings" remains a manual GUI check.

**Bins are `kdenlive:folder.<parent>.<id>` properties in `main_bin`; clips are
`<entry producer="id">` there.**
_Why:_ Rating→bin routing needs the folder *id*, not its name.
_How to apply:_ Build a name→id map from the folder properties; route rating ≥ 4
to Selects, else A-Cam; fall back to `-1` (root) if the folder is absent.

**A semantic (not byte-exact) XML round-trip is fine for .kdenlive.**
_Why:_ Kdenlive reformats on every save, so preserving element/attr/text but
normalizing whitespace loses nothing that matters.
_How to apply:_ The node tree preserves structure and attribute order; tests
assert counts/values, not bytes.

---

## CHR-11 — new + video.yaml

**Initialize slices you want to serialize as `[]`, not leave nil.**
_Why:_ A nil `[]string` marshals to `null` in YAML/JSON; `tags:` and
`playlistIds:` reading as `null` is ambiguous and annoying to hand-edit.
_How to apply:_ `Default()` sets them to `[]string{}` so a fresh video.yaml has
real empty lists.

**Inject the clock (`now time.Time`) into anything that date-defaults.**
_Why:_ `studio new` derives the folder date from "today"; a test can't assert a
stable path if the code calls `time.Now()` internally.
_How to apply:_ Business logic takes `now time.Time`; only the CLI passes
`time.Now()`. Same pattern already used in ingest/apply.

---

## CHR-12 — search

**Resolve clip paths against the manifest's own directory, not its stored
`shoot.root`.**
_Why:_ `shoot.root` is an absolute path captured at ingest; if the project
folder is later moved, it's stale. The directory the manifest was just found in
is always correct.
_How to apply:_ `filepath.Join(filepath.Dir(manifestPath), files.original)`. Pair
with the schema-only loader so a moved tree doesn't fail path validation.

**A "disabled" numeric filter needs a sentinel, not the zero value.**
_Why:_ `Filters{}` has `MaxDur == 0`, which as an upper bound rejects every
clip. Silent and total.
_How to apply:_ `NewFilters()` sets duration bounds to -1 (off); Match treats
`>= 0` as active. Never construct the struct literal directly.

**Model an inclusive `--until <period>` as an exclusive end instant.**
_Why:_ "until 2026-07" should include all of July regardless of time-of-day.
_How to apply:_ Parse the period to the first moment *after* it and test
`!createdAt.Before(end)`; avoids fencepost bugs from end-of-month math.

---

## CHR-13 — chapters

**Kdenlive stores timeline guides in `kdenlive:sequenceproperties.guides` as a
JSON array of `{pos (frames), comment, type, duration}`.**
_Why:_ Pinned so a future format change is caught, not silently swallowed.
_How to apply:_ Constants in `internal/kdenlive/guides.go` with a dated
"verified against real files" comment; `Guides()` errors (dumping the raw value)
if the JSON doesn't parse. Frame→sec uses `<profile>` frame_rate_num/den, whose
attribute order varies between files — read them by name, not position.

**The last chapter's length is unknowable from guides alone.**
_Why:_ The ≥10s rule needs each chapter's duration, but the final one runs to
the end of the render, which the .kdenlive doesn't state.
_How to apply:_ Check the gap to the *next* start; leave the last chapter
unchecked (plan accepts this).

**A sentinel-delimited managed block must require both sentinels before
replacing.**
_Why:_ A description with only a start sentinel (hand-mangled) would otherwise
compute a bad range.
_How to apply:_ Replace only when `end > start`; otherwise append a fresh block.
Keeps text outside the sentinels byte-stable.

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
