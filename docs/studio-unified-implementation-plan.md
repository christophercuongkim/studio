# `studio` — Unified Implementation Plan

Single spec for the whole personal YouTube pipeline. Supersedes
`dailies-implementation-plan.md` and `studio-tools-implementation-plan.md`.
A developer should be able to build any milestone from this document without
asking clarifying questions.

---

## 1. Purpose

One local Go binary covering the full life of a video:

```
studio new → script/prompt → shoot → ingest → serve (review) → apply (rename)
  → scaffold (.kdenlive) → edit → chapters → qc → thumbs → upload → archive
```

Plus `studio search` across everything ever ingested.

**Design principles**

- One static binary, one repo, stdlib-first. Frontends embedded via `embed.FS`.
- `ffmpeg`/`ffprobe` are external dependencies invoked via `os/exec`; never link libav.
- Per-video state lives in files inside the project folder (`manifest.json`,
  `video.yaml`, `script.md`). No database.
- Every destructive step has `--dry-run`; renames write an undo log.
- Renames happen **before** anything is imported into Kdenlive. The tool never
  edits an existing `.kdenlive`; it only creates new ones.
- Servers bind `127.0.0.1` only — sole exception `studio prompt` (§10), which
  defaults to `0.0.0.0` for second-device (iPad/Tailscale) use.
- The suite never deletes user media. The only deletions ever performed are of
  files the tool itself created (`proxy/`, `thumbs/raw/`), and only in `archive`.

**Non-goals (v1):** AI tagging, transcription, editing features, multi-user,
watch-folder daemons, Windows, analytics.

---

## 2. Environment, dependencies, repo layout

- Go ≥ 1.22, Linux (target: NixOS). Dev shell (`flake.nix` devShell): `go`,
  `ffmpeg-full`, `gopls`.
- External binaries on PATH: `ffmpeg`, `ffprobe` (≥ 6.x); `rsync` for archive;
  `mpv` optional for `search --play`.
- Allowed third-party Go deps — everything else is stdlib:
  - `github.com/cespare/xxhash/v2` (checksums)
  - `gopkg.in/yaml.v3` (config, `video.yaml`, script frontmatter)
  - `golang.org/x/sync/errgroup` (bounded parallelism, optional)
  - `golang.org/x/oauth2`, `google.golang.org/api/youtube/v3` (upload only)

```
studio/
  cmd/studio/main.go         # CLI dispatch only
  internal/config/           # ~/.config/studio/config.yaml load/defaults
  internal/manifest/         # types, load/save, validation
  internal/probe/            # ffprobe wrapper
  internal/ingest/           # walk, group, checksum, proxy resolution
  internal/proxy/            # ffmpeg proxy generation
  internal/server/           # review server (serve)
  internal/apply/            # rename engine + undo log
  internal/kdenlive/         # tolerant XML tree, template injection, guide parsing
  internal/script/           # script.md parser
  internal/promptsrv/        # prompter server
  internal/search/           # cross-shoot search
  internal/qc/               # render checks
  internal/thumbs/           # frame extraction + scoring + contact sheet
  internal/upload/           # OAuth + YouTube API
  internal/archive/          # verify + rsync + prune
  internal/webui/            # embedded frontends (review + prompter)
  templates/                 # user-provided empty .kdenlive templates
  flake.nix
```

### 2.1 Global config

`~/.config/studio/config.yaml` (created with commented defaults on first run):

```yaml
projectsRoot: ~/videos            # where `studio new` creates projects
searchRoots: [~/videos]           # dirs scanned for manifest.json (recursive)
kdenliveTemplate: ~/videos/templates/empty-25.12.kdenlive
camCode: DJI                      # default; per-run --cam overrides
player: mpv
archiveRoot: ""                   # empty = archive command disabled
uploadDefaults: {privacy: private, categoryId: "27"}
```

---

## 3. Project layout & naming conventions

### 3.1 Project folder (one per video; created by `studio new`, populated by `ingest`)

```
<projectsRoot>/<yyyy-mm-dd>_<slug>/
  video.yaml           # publish metadata — single source of truth (§9.2)
  script.md            # outline/prompter script (§10.1)
  <slug>.kdenlive      # created by `studio scaffold` after ingest
  manifest.json        # footage manifest (§4)
  undo/                # apply logs (JSONL) + ingest logs
  originals/           # high-quality camera files + sidecars
  proxy/               # edit/scrub proxies (camera-adopted or generated)
  render/              # Kdenlive exports land here by convention
  thumbs/              # thumbnail candidates + chosen thumbnail.png
```

`ingest` moves (not copies) files from the dump location into `originals/`
unless `--copy` is passed. Source subdirectories are flattened; name collisions
among flattened source files are a fatal ingest error (report and abort before
moving anything). A "project" argument to any command = path to this folder;
every command validates the pieces it needs and says exactly what's missing.

### 3.2 Filename template

Final clip name (applied in `apply`):

```
{date}_{cam}{seq}_{desc}[_t{take}].{ext}
  date : YYYYMMDD from media creation time (probe; fallback file mtime)
  cam  : camera code (config default, --cam override), e.g. A, B, DJI
  seq  : 3-digit ingest-order sequence within the project, zero-padded
  desc : user text from review UI; lowercased, spaces→"-",
         charset [a-z0-9-] only, max 40 chars (UI enforces)
  take : optional integer, only if user sets it
```

Example group after apply:

```
originals/20260827_DJI014_sunset-dock-wide.MP4
originals/20260827_DJI014_sunset-dock-wide.SRT
proxy/20260827_DJI014_sunset-dock-wide.mp4
```

**Rule: every file in a clip group shares the same stem.** The proxy is always
`.mp4`; sidecars keep their extensions.

### 3.3 Kdenlive external proxy pattern

Because original and proxy share a stem in fixed sibling dirs, configure
Kdenlive (Settings → Configure Kdenlive → Proxy Clips → External proxies) with a
profile matching: original in `originals/`, proxy at `../proxy/<stem>.mp4`.
Document the exact pattern string in the README after testing against the
installed Kdenlive version — verify on first run, don't assume the settings UI.

---

## 4. Manifest schema

`manifest.json`, UTF-8, written atomically (temp file in same dir + fsync +
rename). Top-level:

```json
{
  "schemaVersion": 1,
  "shoot": {
    "root": "/abs/path/2026-08-27_lake-trip",
    "title": "lake-trip",
    "createdAt": "2026-08-27T18:03:11Z",
    "camCode": "DJI"
  },
  "clips": [
    {
      "id": "c-014",
      "seq": 14,
      "stem": "DJI_20260827_142301_0014_D",
      "files": {
        "original": "originals/DJI_20260827_142301_0014_D.MP4",
        "proxy": "proxy/DJI_20260827_142301_0014_D.mp4",
        "sidecars": ["originals/DJI_20260827_142301_0014_D.SRT"]
      },
      "media": {
        "durationSec": 42.36,
        "width": 3840, "height": 2160, "fps": "29.97",
        "vcodec": "hevc", "hasAudio": true,
        "createdAt": "2026-08-27T14:23:01Z",
        "sizeBytes": 512034441,
        "xxh64": "9f2c7c1a55aa1b02"
      },
      "proxyInfo": { "source": "camera", "note": "" },
      "review": {
        "status": "pending",
        "rating": 0,
        "desc": "",
        "take": null,
        "reviewedAt": null
      },
      "applied": { "done": false, "finalStem": "", "appliedAt": null }
    }
  ],
  "archivedAt": null,
  "archivePath": ""
}
```

Field rules:

- `review.status` ∈ `pending | kept | rejected`. `apply` skips `rejected`
  (files stay under original names) and errors on kept clips with empty `desc`.
- `proxyInfo.source` ∈ `camera | generated | none`. `none` requires a non-empty
  `note` (e.g. generation failed) and degrades `serve` gracefully for that clip
  (metadata card shown, no video).
- IDs are stable for the manifest's lifetime; never renumber.
- Loader rejects unknown `schemaVersion` and validates all relative paths exist.

---

## 5. CLI surface

```
studio new      <slug> [--title "…"] [--date YYYY-MM-DD]
studio ingest   <dump-dir> --project <dir> [--cam DJI] [--copy] [--jobs N] [--append]
studio serve    <project> [--addr 127.0.0.1:7723]
studio apply    <project> [--dry-run]
studio undo     <project> [--log undo/<file>.jsonl]      # default: latest log
studio scaffold <project> [--template <path>] [--out <path>]
studio status   <project>                # counts: pending/kept/rejected/applied
studio search   <text> [filters…]
studio prompt   <project> [--addr 0.0.0.0:7724] [--log]
studio chapters <project> [--project-file <x.kdenlive>] [--write] [--offset 0s]
studio qc       <project> [--render <file>] [--profile 4k30|1080p30|1080p60]
studio thumbs   <project> [--from render|clips] [--count 12]
studio upload   <project> [--dry-run] [--privacy …] [--update] [--skip-qc]
studio archive  <project> [--dry-run] [--keep-proxies] [--force]
```

All commands: `--help`, non-zero exit on any error, human-readable errors on
stderr, no partial silent failures. `--jobs` defaults to `runtime.NumCPU()/2`,
min 1.

---

## 6. `ingest`

Steps, in order. Abort with no filesystem changes if any pre-move step fails.

1. **Walk** `<dump-dir>` recursively. Collect files with extensions
   (case-insensitive): media `.mp4 .mov .mxf .avi .mts .m4v`; sidecars
   `.srt .lrf .lrv .thm .xml .wav`. Ignore hidden files and zero-byte files (warn).
2. **Group by stem** (filename minus extension, case-insensitive):
   - Exact stem match ⇒ same group.
   - DJI: `X.LRF` groups with `X.MP4` — covered by exact match.
   - GoPro: `GL010042.LRV` pairs with `GX010042.MP4` (2nd char differs).
     Implement a small table of *pairing rules* `{pattern, transform}`; ship DJI
     (identity) and GoPro (`GL↔GX`, `.LRV→.MP4`). Unmatched sidecars ⇒ warning
     list, moved to `originals/_unmatched/`, not added to the manifest.
   - Exactly one media file per group is the **original** (largest media file);
     a second media-like file matching a pairing rule is a **camera proxy
     candidate**.
3. **Probe** every media file:
   `ffprobe -v error -print_format json -show_format -show_streams <f>` →
   `internal/probe.Result` (duration, streams, codec, dims, fps,
   `format.tags.creation_time`). Probe failure on an original ⇒ fatal for that
   group: record in the ingest error report, skip the group, leave files in the dump.
4. **Checksum** originals with xxh64 (streamed, 1 MiB buffer). Parallel, `--jobs`.
5. **Move** groups into layout (§3.1): originals+sidecars → `originals/`; the
   camera proxy candidate is held aside for step 6. `os.Rename`; on `EXDEV`
   fall back to copy+fsync+verify-size+remove. `--copy` forces copy mode.
6. **Resolve proxy** per group:

   ```
   resolveProxy(group):
     if camera proxy candidate exists:
         p := probe(candidate)
         ok := p.vcodec == "h264"
            && p.hasAudio == group.original.hasAudio
            && abs(p.duration - original.duration) <= 1.5s
         if ok:
             place at proxy/<stem>.mp4      # .LRF/.LRV → .mp4
             proxyInfo = {source: "camera"}; return
         else:
             move candidate to originals/_rejected-proxies/ ; log reason
     generate:
         ffmpeg -y -i originals/<orig> \
           -vf "scale=-2:480:flags=bicubic" -c:v libx264 -profile:v high \
           -preset veryfast -crf 26 -g 48 -pix_fmt yuv420p \
           -c:a aac -b:a 96k -movflags +faststart proxy/<stem>.mp4
         success → proxyInfo = {source: "generated"}
         failure → proxyInfo = {source: "none", note: "<ffmpeg stderr tail>"}; warn
   ```

   Generation runs in parallel (`--jobs`); each job's ffmpeg stderr goes to
   `undo/ingest-<ts>.log`.
7. **Write manifest** (atomic). Print summary: groups, adopted/generated/failed
   proxies, unmatched sidecars, total bytes, elapsed.

**Idempotency:** if `manifest.json` exists, `ingest` refuses to run without
`--append`; `--append` only adds new stems (checksum-dedupe against existing
entries) and never touches reviewed clips.

---

## 7. `serve` — review UI

### 7.1 Backend

`net/http`, bind `127.0.0.1` only.

- `GET  /` → embedded `index.html`
- `GET  /api/manifest` → full manifest JSON
- `PATCH /api/clips/{id}` → body `{status?, rating?, desc?, take?}`; validates
  (`desc` charset per §3.2), sets `reviewedAt`, saves atomically, returns clip
- `GET  /media/proxy/{id}` → `http.ServeContent` on the proxy file (range
  requests free; `Content-Type: video/mp4`)
- `GET  /api/preview-name/{id}` → final name the current fields would produce

Manifest access behind a mutex; save debounced ≤ 500 ms after last PATCH plus
save-on-shutdown (SIGINT handled).

### 7.2 Frontend (single `index.html` + `app.js` + `app.css`; no framework, no CDN)

Layout: left sidebar clip list (name, duration, status glyph, rating stars);
main pane `<video>` + metadata line (res/fps/codec/proxy source) + name form
(`desc` input, `take` number input, live final-name preview) + progress footer
`kept/rejected/pending`.

Keymap (active except when an input is focused; `Esc` blurs):

```
j / k        next / previous clip (autoplays muted)
Space        play/pause
← / →        seek −2s / +2s      Shift+←/→  −10s / +10s
, / .        frame step back/forward (pause first; fps from manifest)
1–5          set rating
x            toggle rejected
r            focus desc field
Enter        (in desc) save via PATCH, mark kept, advance to next pending
u            jump to next pending clip
```

Client keeps manifest in memory, PATCHes on change, optimistic UI with error
toast on non-200. Plain ES modules, no build step.

---

## 8. `apply` + `undo` — rename engine

1. Build the plan for every `kept && !applied.done` clip: compute `finalStem`
   (§3.2); map every file in the group old→new (same dir, stem swap, extensions
   preserved; proxy stays `.mp4`).
2. **Validate the whole plan before touching disk:** no empty `desc` on kept
   clips; no duplicate `finalStem` across the project (including
   already-applied); no target path exists; every source path exists. Any
   violation ⇒ print all violations, exit 1, change nothing.
3. `--dry-run`: print the full old→new table, exit 0.
4. Execute: open `undo/apply-<RFC3339>.jsonl`. Per file: write intent line
   `{"op":"rename","from":"…","to":"…","group":"c-014"}`, fsync, then
   `os.Rename`. On any error: stop immediately, point at the log, exit 1
   (partial state is recoverable via `undo`).
5. On group success set `applied = {done:true, finalStem, appliedAt}`; save the
   manifest once at the end. Print summary.

`undo` replays the chosen log **in reverse**, renaming `to→from`, skipping
lines whose `to` no longer exists (warn), and resets `applied` on affected clips.

---

## 9. `new` + `scaffold` — project creation and Kdenlive generation

### 9.1 `studio new`

Creates the §3.1 layout: directories, `script.md` skeleton (frontmatter +
`## Hook` / `## Outro`), and `video.yaml` from the schema below with
`uploadDefaults` merged. Refuses if the target dir exists. Prints the
next-steps checklist (script → shoot → ingest → serve → apply → scaffold →
edit → chapters → qc → thumbs → upload → archive).

### 9.2 `video.yaml` schema

```yaml
schemaVersion: 1
title: ""              # ≤100 chars enforced at upload
description: ""        # chapters block managed by `studio chapters --write`
tags: []               # total ≤500 chars enforced at upload
categoryId: "27"
privacy: private       # private|unlisted|public
publishAt: null        # RFC3339; requires privacy: private
playlistIds: []
thumbnail: thumbs/thumbnail.png
render: render/final.mp4
youtube:               # written back by `studio upload`
  videoId: null
  uploadedAt: null
```

### 9.3 `studio scaffold`

Template injection, never from-scratch generation.

1. One-time setup (README): in the installed Kdenlive, create an empty project
   with the desired profile and bin folders (`A-Cam`, `B-Roll`, `Audio`,
   `Selects`), save as `templates/empty-<ver>.kdenlive`.
2. Parse the template with a tolerant XML Node tree (`internal/kdenlive`:
   generic `Node{XMLName, Attrs, Content, Children}`) so round-tripping
   preserves everything untouched.
3. For each applied, kept clip: insert an MLT `<producer>` (unique id,
   `resource` = absolute path to the original) plus the kdenlive properties
   observed in the template's own example clip. **Mechanism: before writing
   this code, add one clip to the template project in Kdenlive, save, diff the
   XML, and replicate exactly those elements/attributes.** Reference each
   producer from the `main_bin` playlist the same way the diff shows. Bin
   assignment: rating ≥ 4 → `Selects`, else `A-Cam` (v1).
4. Never overwrite an existing output; require `--out` to a non-existent path.
5. Acceptance: opens in Kdenlive with zero warnings, all clips online, proxies
   attach via the external-proxy profile (§3.3).

If Kdenlive's format shifts on upgrade: regenerate the template, re-run the
diff; nothing else changes.

---

## 10. `prompt` — bullet script + recording prompter

### 10.1 `script.md` format

Plain markdown, no custom syntax beyond frontmatter:

```markdown
---
title: "Why I ported ani-cli to Go"
targetLength: 8m
---

## Hook
- cold open: the 3-line demo
- "every anime CLI is bash. mine isn't. here's why"

## The problem
- bash scraping breaks weekly
- what I actually wanted: …

## Outro
- sub tease: next video = the release pipeline
```

Rules: `##` sections are prompter pages **and** the expected chapter list
(shared vocabulary with §11 — guide comments in Kdenlive should match section
names; `chapters` warns on mismatch as a later nicety). Bullets = talking
points; one nesting level allowed. Parser: hand-rolled line parser
(frontmatter via yaml.v3, then `## ` / `- ` / `  - `); reject anything else
with line numbers. No markdown library.

### 10.2 Prompter server

- Same embedded-frontend pattern as `serve`. Default bind `0.0.0.0` **only
  here** (iPad over Tailscale); print URLs for every non-loopback interface on
  startup.
- UI: one section per screen. Section title large, bullets very large
  (readable at 2 m on an iPad), high-contrast dark theme, current bullet
  highlighted, next section title ghosted at bottom. Progress bar
  (sections done/total), running clock, per-section timer resetting on change.
- Keys — and equivalent on-screen tap zones (left third back, right third
  forward), since the iPad has no keyboard by the camera:
  `→/Space` next bullet (advances section when exhausted), `←` back,
  `n/p` whole section, `t` new take (increments counter, flashes it, resets
  section timer), `.` blackout toggle between takes.
- `--log`: append events to `prompt-log.jsonl` in the project:
  `{"ts":"…","event":"section|bullet|take","section":"Hook","take":2}` —
  wall-clock timestamps for later correlation with footage creation times.
  v1 only writes the log (future consumer: pre-placing Kdenlive guides).
- State is server-side (one mutex-guarded struct) so multiple devices stay in
  sync via 1 s polling of `GET /api/state`. No websockets in v1.

---

## 11. `chapters` — Kdenlive markers → YouTube chapters

Default prints to stdout; `--write` updates `video.yaml:description` between
sentinels `<!-- chapters:start -->` / `<!-- chapters:end -->` (append the
block to the description if sentinels are absent).

1. Locate the `.kdenlive` (newest in project root unless `--project-file`).
2. Parse with the §9.3 Node tree. Timeline guides live in a document property
   as a JSON array with a frame position and comment. **Verification
   procedure (one-time, first):** add two guides in the installed Kdenlive,
   save, diff the XML, and pin the exact property name and JSON field names in
   `internal/kdenlive/guides.go` constants, commented with the Kdenlive
   version tested. Parsing must fail loudly (dumping the property) if the
   structure differs — never silently emit zero chapters.
3. Frame → seconds via the profile fps from the project XML
   (`frame_rate_num/frame_rate_den` on `<profile>`).
4. Apply `--offset` (renders not starting at timeline 00:00; default 0).
5. Enforce YouTube's chapter rules, erroring with fix hints:
   first chapter must be `00:00` (if the first guide isn't at 0, synthesize
   `00:00 Intro` and warn); minimum 3 chapters; every chapter ≥ 10 s
   (no automatic merging; list offenders).
6. Format `MM:SS Title` under an hour, `H:MM:SS` above. Title = guide comment,
   trimmed; fallback `Chapter N`.

---

## 12. `search` — cross-shoot clip search (read-only)

```
studio search <text> [--rating 4+] [--status kept] [--cam DJI]
              [--since 2026-01] [--until 2026-08] [--dur 5s..2m]
              [--shoot lake] [--json] [--paths] [--play N]
```

- `<text>`: case-insensitive substring against `review.desc` and shoot title;
  multiple words = AND. No fuzzy matching in v1.
- `--paths`: original paths only (pipe-friendly). `--json`: full records.
  `--play N`: launch the configured player on result N's **proxy**.

Mechanism: walk `searchRoots` for `manifest.json` (skip dot-dirs, depth cap 6);
parse with the shared loader; unreadable/old-schema manifests → one-line
warning, skipped. No persistent index in v1 — measure first; add a cache
(`~/.cache/studio/index.json`, keyed by manifest mtime) only if a real library
exceeds ~200 ms. Filter in memory; sort rating desc, then date desc. Output
table `#  rating  duration  date  cam  shoot  name`, truncated to terminal
width, count footer.

---

## 13. `qc` — pre-upload render checks

Runs against `video.yaml:render` (or `--render`). All checks always run;
pass/fail table; exit 1 on any FAIL. Thresholds in one tunable struct.

| Check | Method | Fail condition |
|---|---|---|
| Loudness | `ffmpeg -i f -af loudnorm=print_format=json -f null -` (parse the JSON block from stderr) | integrated LUFS outside −14 ±2, or true peak > −1 dBTP |
| Clipping | `astats` max level | ≥ 0 dBFS samples |
| Head/tail silence | `silencedetect=n=-50dB:d=2` | silence overlapping first or last 3 s |
| Head/tail black | `blackdetect=d=1:pix_th=0.10` | overlap with first/last 2 s (WARN — intros exist) |
| Stream sanity | ffprobe vs `--profile` | wrong res/fps, codec ∉ {h264, hevc}, no audio |
| Duration sanity | ffprobe vs `script.md:targetLength` | > 2× target (WARN) |
| A/V sync drift | compare stream durations | delta > 200 ms (WARN) |

Each FAIL prints offending timestamps and the exact ffmpeg line to reproduce.
On pass, write `qc-report.json` (checks, values, timestamp) — consumed by
`upload`.

---

## 14. `thumbs` — thumbnail candidates

1. Source frames: `--from render` samples the final render every N seconds
   (N = duration/60, min 2 s); `--from clips` samples rating-5 originals via
   the manifest, 5 frames each. Extraction: one ffmpeg run per source
   (`-vf fps=…`) to `thumbs/raw/%04d.png` at 1280×720.
2. Score in Go (`image/png` + own code, no OpenCV): sharpness = variance of a
   3×3 Laplacian on the luma plane; discard bottom 50%; then greedily pick
   `--count` frames maximizing pairwise timestamp spread (avoids near-dupes).
3. Output: winners as `thumbs/candidate-NN-<ts>.png`; delete `raw/`; write
   `thumbs/contact-sheet.png` (`image/draw` grid, 4 columns, timestamp
   captions). Final art is a human job: save/edit the pick as
   `thumbs/thumbnail.png`.

Performance target: < 1 min for a 10-min 4K render (measure; drop extraction to
960×540 if missed).

---

## 15. `upload` — YouTube upload

- **API:** YouTube Data API v3 — `videos.insert` (resumable), then
  `thumbnails.set`, then `playlistItems.insert` per playlist. Scopes
  `youtube.upload` + `youtube`.
- **Auth:** installed-app OAuth with local redirect on `127.0.0.1`; user
  creates their own Google Cloud project + OAuth client (README walkthrough;
  verify current console steps against Google's docs when writing it — the
  console UI moves). Token cached at `~/.config/studio/yt-token.json` (0600),
  auto-refresh.
- **Flow:** validate `video.yaml` hard limits (title ≤ 100 chars, tags total
  ≤ 500 chars, thumbnail ≤ 2 MB at 1280×720, render exists) and require a
  passing `qc-report.json` unless `--skip-qc`. `--dry-run` prints the full
  request payload. Upload with progress bar; chunked, resumable, 3 retries with
  backoff on transient failure. On success write `youtube.videoId` /
  `uploadedAt` back to `video.yaml`.
- **Idempotency:** refuse if `youtube.videoId` is set. `--update` performs a
  metadata-only `videos.update` (+ thumbnail re-set) — in scope for v1; you'll
  want it the first time you fix a typo.
- **Quota:** an upload costs ~1600 units of the default 10k/day — fine for
  personal volume; surface quota errors verbatim.

---

## 16. `archive` — verify and cold-store

Preconditions: all kept clips applied; warn if `youtube.videoId` is null
(archiving unpublished needs `--force`).

1. Re-hash every `originals/` file against manifest `xxh64`; any mismatch
   aborts the run (bit-rot or tampering — investigate, don't archive).
2. If `archiveRoot` set: `rsync -a --checksum` project → archive, then verify
   by re-hashing the copy. (Shelling out to rsync is deliberate; remote
   archiveRoots work free via rsync-over-ssh. Redundancy beyond one archive
   copy is restic/homelab territory, out of scope.)
3. Prune: delete `proxy/` and `thumbs/raw/` unless `--keep-proxies`
   (regenerable; `proxyInfo` in the manifest allows a future `studio proxies`
   rebuild). The suite's only deletions, and only of files it created.
4. Stamp the manifest with `archivedAt` + `archivePath`.

---

## 17. Global error-handling rules

- External commands: capture stderr; include the last ~10 lines in errors.
- All manifest / `video.yaml` writes: marshal → temp file in same dir → fsync
  → rename.
- Ctrl-C during ingest proxy generation: finish/kill current ffmpeg jobs,
  write the manifest with completed groups only, print resume instructions
  (`--append`).

---

## 18. Milestones & acceptance

| # | Deliverable | Acceptance |
|---|---|---|
| 1 | `config`, `manifest`, `probe` | Unit tests: schema round-trip; ffprobe JSON fixtures (DJI HEVC, GoPro, no-audio) parse |
| 2 | `ingest` | Fixture dump (DJI orig+LRF+SRT, GoPro GX+LRV, orphan SRT, corrupt MP4): correct grouping, LRF adopted, corrupt file reported+skipped, manifest valid, re-run refuses without `--append` |
| 3 | `serve` | Instant scrubbing on 480p proxies; full keyboard flow rates+names 20 clips mouse-free; SIGINT loses no edits |
| 4 | `apply` + `undo` | Dry-run table correct; duplicate-desc plan rejected; injected mid-run failure (read-only dir) leaves recoverable state; `undo` restores a byte-identical tree (find + xxh64) |
| 5 | `scaffold` | Opens clean in the installed Kdenlive; proxies attach; refuses existing `--out` |
| 6 | `new` | Layout + skeletons created; refuses existing dir |
| 7 | `search` | Fixture tree with 3 shoots incl. one corrupt manifest: AND-matching, every filter, `--paths` works in `xargs`, corrupt manifest warns without failing |
| 8 | `chapters` | Fixtures (0 guides / 2 guides / no guide at 0 / 9-s gap): loud parse-error path, synthesized intro, min-count error, short-chapter error; `--write` round-trips `video.yaml` byte-stable outside sentinels |
| 9 | `prompt` | Parse errors show line numbers; iPad Safari renders, tap zones work; two clients sync ≤ 1 s; log replays cleanly |
| 10 | `qc` | Fixture renders engineered per failure (ffmpeg `sine`/`color` sources): each check trips correctly; `qc-report.json` written on pass |
| 11 | `thumbs` | Candidates visibly sharper than average and time-spread; meets the < 1 min target |
| 12 | `upload` | Dry-run payload correct; real unlisted upload lands with all metadata + thumbnail; second run refuses; `--update` fixes a description |
| 13 | `archive` | One corrupted byte → abort; archived copy re-hash matches; prune deletes only tool-created dirs |
| 14 | Docs | README: Kdenlive external-proxy setup, template + guide diff procedures, OAuth walkthrough, flake devShell, one real end-to-end video |

**Build order:** 1→5 first (the footage core — this is the tool you asked for
originally), then 6, 7, 8 (instant payoffs), 9 before your next recording
session, then 10→12 (the publish path, in dependency order), 13 once the first
video ships. Rough calendar: core in two weekends; the rest are mostly
one-to-two-evening tools, with `prompt` and `upload` each a weekend.

## 19. Deferred (do not build in v1)

Hover-scrub thumbnail strips · whisper transcripts for sync-sound naming ·
DNxHR editorial proxies · shot-list/pre-shoot manifest · prompt-log →
Kdenlive guide pre-placement · search cache & fuzzy matching ·
`studio proxies` regeneration · `studio clean` · analytics pulls.
Each slots in behind existing files without schema breaks
(bump `schemaVersion` when needed).
