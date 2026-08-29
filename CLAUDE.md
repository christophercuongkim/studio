# studio — Working Agreement

Repo-specific rules. Layers on top of the global `~/CLAUDE.md`.

`studio` is one local Go binary covering the full life of a YouTube video:
`new → script/prompt → shoot → ingest → serve → apply → scaffold → edit →
chapters → qc → thumbs → upload → archive`, plus `studio search`.

## Reference Docs

- `docs/studio-unified-implementation-plan.md` — **the single build spec.** Data
  model, filename templates, per-command behavior, milestone build order.
  Any milestone should be buildable from it without clarifying questions. Read
  the relevant section before implementing a command; follow its milestone order
  when unsure what's next.
- `docs/lessons.md` — accumulated gotchas (see Lessons Loop).

## Stack

- **Go ≥ 1.22, stdlib-first.** Linux, target NixOS. Dev shell via `flake.nix`
  devShell: `go`, `ffmpeg-full`, `gopls`.
- **One static binary.** Frontends (review + prompter) embedded via `embed.FS`.
  `cmd/studio/main.go` is dispatch only; logic lives in `internal/<area>/`.
- **External binaries on PATH:** `ffmpeg`, `ffprobe` (≥ 6.x), `rsync` (archive),
  `mpv` (optional, `search --play`). Invoke via `os/exec`.
- **Sanctioned third-party deps — everything else is stdlib:**
  `cespare/xxhash/v2`, `gopkg.in/yaml.v3`, `golang.org/x/sync/errgroup`,
  `golang.org/x/oauth2`, `google.golang.org/api/youtube/v3` (upload only).
  Adding any dep outside this list needs discussion first.

## Invariants (Law — not taste)

Pulled from the plan's design principles. Violating one is a bug, not a style nit.

- **Never link libav.** Shell out to `ffmpeg`/`ffprobe`; parse their output.
- **File-based state, no DB.** Per-video state is files in the project folder:
  `manifest.json`, `video.yaml`, `script.md`. `video.yaml` is the single source
  of truth for publish metadata.
- **Every destructive step has `--dry-run`.** Renames (`apply`) write a JSONL
  undo log; `undo` reverses it.
- **Renames happen before Kdenlive import.** The tool never edits an existing
  `.kdenlive` — it only creates new ones (`scaffold`).
- **Never delete user media.** The only deletions ever performed are of files the
  tool itself created (`proxy/`, `thumbs/raw/`), and only in `archive`.
- **Servers bind `127.0.0.1`** — sole exception `studio prompt`, which defaults
  to `0.0.0.0` for second-device (iPad/Tailscale) use.
- **`ingest` always copies, never moves** — the dump is left intact so a card is
  safe to eject. `--clear-source` empties the card afterward, deleting each source
  only once its copy verifies. Flattened-name collisions are a fatal error —
  report and abort before writing anything.
- Kdenlive XML parsing must **fail loudly** (dump the property) on an untested
  version, never silently mangle.

## Conventions

- `gofmt` + `go vet` clean before every commit.
- **Table-driven unit tests.** Parsers get real fixtures: ffprobe JSON for DJI
  HEVC / GoPro / no-audio; schema round-trip for `config`/`manifest`.
- Each command validates the pieces it needs up front and says **exactly** what's
  missing. A "project" arg = path to the project folder.
- Document non-obvious formats (e.g. the DJI filename pattern) in the README
  after testing against real files.

## Delivery

- **Branch** `feat/`|`fix/`|`chore/`|`docs/` from `main`. Never work on `main`.
  Sync by **rebase**, never merge.
- No `Co-Authored-By` / Claude Code trailer in commits or PR body (global rule).
- **Self-review pass** on each PR — line-anchored comments on the non-obvious
  bits (workarounds, sentinels, "why not the obvious approach"). Comments on the
  PR, not in source.
- **Doc-only PRs self-merge** (`docs/**`, `*.md`, `CLAUDE.md`, `.gitignore`).
  Anything touching code → open, self-review, then wait for manual review.
- **Done bar:** verify with `git status` / `git log`, not memory. Tooling
  changes: run it end-to-end (a real command against real media), not just
  `go build`.

## Lessons Loop

- Read `docs/lessons.md` at session start.
- **Never interrupt a task to update lessons.** Finish first; discuss corrections
  between tasks, then update `docs/lessons.md` before starting the next.
- Format: rule + **Why:** + **How to apply:**. Short, specific, non-obvious.
  Merge duplicates; skip generic advice — only capture surprises.
- "Correction" = "no, don't" / "we don't do that here" / "you missed X". Not
  every review nit.
