# studio

One local Go binary covering the full life of a personal YouTube video:

```
studio new → script/prompt → shoot → ingest → serve (review) → apply (rename)
  → scaffold (.kdenlive) → edit → chapters → qc → thumbs → upload → archive
```

plus `studio search` across everything you've ever ingested.

It's a single static binary, stdlib-first, with the review and prompter web UIs
embedded. `ffmpeg`/`ffprobe` do the media work (shelled out, never linked);
per-video state lives in plain files in the project folder — no database.

See [`docs/studio-unified-implementation-plan.md`](docs/studio-unified-implementation-plan.md)
for the full design spec and [`docs/lessons.md`](docs/lessons.md) for accumulated
gotchas.

---

## Install / dev shell

The dev shell pins the whole toolchain (Go, ffmpeg, gopls, rsync, mpv):

```sh
nix develop            # drops you into a shell with everything on PATH
go build -o studio ./cmd/studio
./studio --help
```

External binaries expected on PATH (all provided by the dev shell): `ffmpeg`,
`ffprobe` (≥ 6), `rsync` (archive), `mpv` (optional, `search --play`). Kdenlive
itself is only needed for editing and for the one-time template/guide setup below.

---

## Configuration

On first run studio writes `~/.config/studio/config.yaml` with commented
defaults (honors `$XDG_CONFIG_HOME`):

```yaml
projectsRoot: ~/videos            # where `studio new` creates projects
externalRoot: ""                  # preferred root when connected (e.g. an external
                                  # SSD: /run/media/you/SSD/videos). Used automatically
                                  # when its drive is mounted, else falls back to
                                  # projectsRoot. One-off override: `new --root <dir>`.
searchRoots: [~/videos]           # dirs scanned for manifest.json (recursive)
kdenliveTemplate: ~/videos/templates/empty-25.12.kdenlive
camCode: DJI                      # default; per-run --cam overrides
player: mpv                       # used by `studio search --play`
archiveRoot: ""                   # empty = `studio archive` disabled
uploadDefaults:
  privacy: private                # private | unlisted | public
  categoryId: "27"                # 27 = Education
```

---

## Commands

| Command | What it does |
|---|---|
| `studio new <slug> [--title] [--date]` | Create a project folder + `script.md` / `video.yaml` skeletons |
| `studio ingest <dump> --project <dir> [--cam] [--copy] [--jobs N] [--append]` | Import a card dump: group, probe, checksum, move, resolve proxies, write `manifest.json` |
| `studio serve <project> [--addr 127.0.0.1:7723]` | Browser review UI: rate, name, keep/reject (localhost) |
| `studio apply <project> [--dry-run]` | Rename kept clips to final names, journaling for undo |
| `studio undo <project> [--log …]` | Reverse the most recent apply |
| `studio scaffold <project> [--template] [--out]` | Generate a `.kdenlive` with applied clips in the bin |
| `studio status <project>` | Review/apply counts |
| `studio search <text> [filters]` | Search clips across every shoot (`--rating 4+`, `--status`, `--cam`, `--since/--until`, `--dur 5s..2m`, `--shoot`, `--paths`, `--json`, `--play N`) |
| `studio prompt <project> [--addr 0.0.0.0:7724] [--log]` | Teleprompter from `script.md` (binds all interfaces for iPad use) |
| `studio chapters <project> [--project-file] [--write] [--offset 0s]` | Kdenlive guides → YouTube chapters |
| `studio qc <project> [--render] [--profile 4k30\|1080p30\|1080p60]` | Pre-upload render checks; exits non-zero on any FAIL |
| `studio thumbs <project> [--from render\|clips] [--count 12]` | Extract + rank thumbnail candidates + contact sheet |
| `studio upload <project> [--dry-run] [--privacy] [--update] [--skip-qc]` | Upload the render to YouTube |
| `studio archive <project> [--dry-run] [--keep-proxies] [--force]` | Verify, rsync to cold storage, prune regenerable files |

Every command validates what it needs up front, exits non-zero on error, and the
destructive ones (`apply`, `archive`, `upload`) support `--dry-run`.

---

## End-to-end walkthrough

```sh
# 1. Start a project and rough out the script.
studio new lake-trip --title "A weekend at the lake"
$EDITOR ~/videos/2026-08-27_lake-trip/script.md

# 2. (Optional) run the teleprompter while recording; open it on an iPad too.
studio prompt ~/videos/2026-08-27_lake-trip --log

# 3. Ingest the card dump (moves files; --copy to keep the card).
studio ingest /run/media/you/DJI_CARD --project ~/videos/2026-08-27_lake-trip --cam DJI

# 4. Review: rate, name, keep/reject — entirely from the keyboard.
studio serve ~/videos/2026-08-27_lake-trip           # http://127.0.0.1:7723

# 5. Rename kept clips to their final names (dry-run first).
studio apply ~/videos/2026-08-27_lake-trip --dry-run
studio apply ~/videos/2026-08-27_lake-trip

# 6. Generate the Kdenlive project, then edit in Kdenlive.
studio scaffold ~/videos/2026-08-27_lake-trip
kdenlive ~/videos/2026-08-27_lake-trip/*.kdenlive

# 7. After editing: chapters from your timeline guides, QC the render, thumbs.
studio chapters ~/videos/2026-08-27_lake-trip --write
studio qc ~/videos/2026-08-27_lake-trip --profile 1080p30
studio thumbs ~/videos/2026-08-27_lake-trip --from render
# pick a candidate, save it as thumbs/thumbnail.png

# 8. Upload (dry-run first to see the payload).
studio upload ~/videos/2026-08-27_lake-trip --dry-run
studio upload ~/videos/2026-08-27_lake-trip

# 9. Once it's live, archive the project to cold storage.
studio archive ~/videos/2026-08-27_lake-trip
```

---

## Kdenlive setup (one-time)

studio never edits an existing `.kdenlive`; it only reads guides (`chapters`)
and creates new projects from a template (`scaffold`). Two one-time setup steps,
both done in the Kdenlive GUI.

### External proxy profile

studio keeps each clip's original in `originals/` and its proxy at
`../proxy/<stem>.mp4` (same stem). Point Kdenlive at that layout:

**Settings → Configure Kdenlive → Proxy Clips → External Proxy**, and set a
profile whose original lives in `originals/` and whose proxy is
`../proxy/%1.mp4` (verify the exact pattern token against your installed
version — the UI moves). Then proxies attach automatically when you open a
scaffolded project.

### Empty template

Create the template `scaffold` injects clips into:

1. New Kdenlive project with your target profile (e.g. 1080p60).
2. Add bin folders: `A-Cam`, `B-Roll`, `Audio`, `Selects`.
3. Save it as `~/videos/templates/empty-<ver>.kdenlive` and set
   `kdenliveTemplate` in the config.

`scaffold` routes rating-5 clips into `Selects` and the rest into `A-Cam`.

### If Kdenlive's format changes on upgrade

studio's Kdenlive parser is pinned to the format it was tested against
(7.38.0). If a future version stores producers or guides differently, studio
**fails loudly** rather than silently mangling — regenerate the template, and if
`chapters` errors on the guides property, update the constants in
`internal/kdenlive/guides.go` (they're commented with the tested version).

---

## YouTube upload (one-time OAuth setup)

`studio upload` uses the YouTube Data API v3 with your own Google OAuth client.

1. In the [Google Cloud Console](https://console.cloud.google.com/): create a
   project, enable the **YouTube Data API v3**, and create an **OAuth client ID**
   of type **Desktop app**. (The console UI changes — follow Google's current
   walkthrough.)
2. Download the client JSON and save it as `~/.config/studio/yt-client.json`.
3. First `studio upload` prints an authorization URL. Open it, grant access, and
   the loopback redirect captures the token. It's cached at
   `~/.config/studio/yt-token.json` (mode 0600) and auto-refreshed thereafter.

`studio upload --dry-run` prints the exact request payload and never touches the
network — use it to check title/tags/privacy/description before going live.
`--update` changes metadata on an already-uploaded video; a second plain
`upload` refuses (idempotent).

---

## Remote access over Tailscale

`studio prompt` binds `0.0.0.0` by default and prints a URL for every
non-loopback interface, including your Tailscale IP — so from any device on the
tailnet you can open `http://<tailscale-ip>:7724` (or the MagicDNS name) and the
prompter loads and stays in sync. Two things to check if a second device can't
reach it:

- **Host firewall.** Open the port, e.g. on NixOS:
  `networking.firewall.allowedTCPPorts = [ 7724 ];`
- **Tailscale ACLs.** The default policy allows device→device; only an issue if
  you've tightened it.

Every other server (`serve`) binds `127.0.0.1` on purpose. To expose one
temporarily, pass `--addr 0.0.0.0:<port>`.

---

## Project layout

```
<projectsRoot>/<yyyy-mm-dd>_<slug>/
  video.yaml           # publish metadata — single source of truth
  script.md            # outline / prompter script
  <slug>.kdenlive      # created by `studio scaffold`
  manifest.json        # footage manifest
  undo/                # apply + ingest logs
  originals/           # camera files + sidecars
  proxy/               # edit/scrub proxies
  render/              # Kdenlive exports land here
  thumbs/              # thumbnail candidates + chosen thumbnail.png
```
