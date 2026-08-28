# Vendored: SeaKim Design System

A pinned snapshot of the SeaKim design system, vendored so studio stays a single
static binary with no build step and no external assets (plan §2). studio's
embedded web UIs (`internal/webui/serve`, `internal/webui/prompt`) are a SeaKim
**web consumer** and conform to **Tier 0** (the identity rules).

## Provenance

- **Source:** https://github.com/christophercuongkim/seakim-design-system
- **Commit:** `6f7de408be6a3172676e6c198f42132f63203885`
- **System version:** `4.3.0`
- **Conformance rules:** `4.3` (the `VERSION` file is authoritative; the prose in
  `conformance.md` still says "4.1"/"1.0" upstream — stale, tracked by `VERSION`).
- **Vendored:** 2026-08-28

## What's here

- `styles.css` + `tokens/` — the runtime token layer (the load-bearing part).
  Studio's UIs build only from the **semantic** tokens (`--surface-*`,
  `--text-*`, `--fill-*`, `--border-*`, `--space-*`, `--radius-*`, `--type-*`,
  `--dur-*`, `--focus-ring`, …), never raw ramps (`--stone-*`/`--brand-*`) or
  hex literals.
- `fonts/` + `tokens/fonts.css` — **local adaptation.** Upstream `fonts.css`
  `@import`s Outfit / Plus Jakarta Sans / IBM Plex Mono from
  `fonts.googleapis.com`. Studio can't use a CDN, so the OFL latin webfonts are
  vendored as `.woff2` and referenced via `@font-face`. The font *families* are
  SeaKim identity; *delivery* is explicitly platform-adaptable (Tier 1) — this
  is a sanctioned adaptation, not a rule break.
- `conformance.md`, `decisions/` (ADRs), `spec/`, `guidelines/` — reference, for
  auditing the binding against the rules.
- `tool/conformance-check.mjs` — the machine checker (ADR 0012: checks ship with
  the rules, the consumer runs them). Run it against the frontends:
  `node internal/webui/seakim/tool/conformance-check.mjs internal/webui`.

## Local adaptations (documented, per Tier 1)

1. **Fonts self-hosted** instead of the Google Fonts `@import` (offline / no-CDN
   invariant). `tokens/fonts.css` replaced; families and weight ranges preserved.
2. **`data-app="studio"` accent binding** appended to `tokens/apps.css`. studio
   isn't one of the upstream apps, so it binds its own accent (turf hue) by
   mirroring the `bench`/`fantasy` ramp — apps.css is the intended extension
   point for per-app accents. Re-apply this block on re-sync.

No Tier 0 (identity) rule is bent. Icon delivery, if icons are added later, must
likewise be self-hosted (Phosphor bundled font or inline SVG) — never a CDN.

## Conformance declaration

```yaml
binding: studio web UI (serve review + prompt teleprompter)
role: consumer          # per ADR 0010: not a reusable binding — no Tier 2 inventory owed
tier: 0                 # the identity rules; met in full
seakim_rules: "4.3"     # reviewed against; from VERSION (never lead)
checker: tool/conformance-check.mjs — clean on internal/webui
adaptations: fonts self-hosted; data-app="studio" turf accent (both documented above)
```

Machine check runs in `go test ./internal/webui` (skips without node; the
devShell provides it). The judgement half of `conformance.md` (one accent per
screen — turf; justified shadows — overlays only; sentence-case verb-labelled
copy; both themes reviewed) was checked by hand.

## Re-syncing

To update: re-clone the source at a newer commit, re-copy `styles.css` +
`tokens/` (re-apply the `fonts.css` local adaptation), refresh the reference
docs + checker, bump the commit/version above, and re-run the checker + the
manual review in `conformance.md`.
