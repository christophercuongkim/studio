# 0015 — A sequential (magnitude) chart ramp

- **Status** Accepted
- **Date** 2026-08-05
- **Affects** every binding; `guidelines/data-visualisation.md`; `tokens/src/color.tokens.json`

## Context

`guidelines/data-visualisation.md` decides colour for **identity** (1–2 series → accent +
grey; 3–6 → the fixed `--chart-1..6` categorical ramp) and leaves **magnitude** undefined.
Magnitude is a real, recurring job — a finish-rank grid, a head-to-head win-rate matrix,
any heatmap or choropleth — and it is exactly the case the categorical ramp is wrong for:
six discrete hues cannot show "more vs less".

With nothing specified a binding improvises, and both obvious improvisations break a rule:

- **App-accent ramp** (`--brand-100..900` as light→dark) reads raw ramp steps, which
  `conformance.md` forbids, and makes the same chart look different in Bench vs Voyage.
- **Accent-at-opacity** keeps to one accent hue and extends the sanctioned "area fill =
  line colour at 12%", but it is per-app and undocumented, so every consumer picks its own
  floor and stop count.

The first real consumer hit this already: fantasy-hub's Hall of Records ships
accent-at-opacity (14%→100%) as an interim, explicitly pending this decision.

## Decision

Add a **fixed, product-independent, single-hue sequential ramp** to
`tokens/src/color.tokens.json` under `chart.seq`, generated to every binding like
`--chart-1..6`.

### Fixed, not app-accent

Same rationale as the categorical ramp, and stronger here: a sequential ramp is **read as
a scale**. Someone who learns "darker means higher" in Bench carries that to Voyage, so
cross-product consistency matters more for magnitude than for identity, not less.

### Four steps, not five

`--chart-seq-1` through `--chart-seq-4`, monotonic in lightness so order survives
greyscale and colour-vision deficiency by construction — single-hue sequential needs no
adjacent-pair hue check.

Four rather than the proposed five because of contrast arithmetic. Across four steps
adjacent pairs land 1.42–1.93:1 apart in light and 1.72:1 in dark; at five steps that
compresses to roughly 1.3:1, below where a cell boundary stays legible without a rule.
A fifth step is possible only if every cell gets a border — a different decision, and one
that fights the hairline grid the table spec already provides.

### Hue 265, chosen to stay out of the way

Indigo. Clear of every app accent (brick 8, sea 245, turf 145, plum 320) and every status
hue (150, 85, 25, 240), so a magnitude ramp is never mistaken for an accent or a state.

### Its own dark steps, validated

Not an automatic flip. Light ascends into darkness from near-white; dark ascends into
lightness from just above `--surface-card`. Per
[0005](0005-light-mode-is-first-class.md), each theme is validated on its own surfaces.

| | 1 | 2 | 3 | 4 |
| --- | --- | --- | --- | --- |
| Light | `#dde8ff` | `#a8c4ff` | `#6f95eb` | `#3d63be` |
| Dark | `#26365c` | `#3c5898` | `#567cd3` | `#82a8fd` |

### Text-on-cell rule

Label ink is `--text-primary` on the pale end and flips to `--text-inverse` once the cell
is dark enough — **at step 4 in light, at step 3 in dark**, both verified above 4.5:1 at
every stop. The flip point differs per theme because the ramps run in opposite directions;
that is a consequence of validating them separately rather than an inconsistency.

### Accent-at-opacity survives as the one exception

`conformance.md` names it the **only** sanctioned alternative, for a one-off tint
deliberately tied to its screen's accent. Not for anything a reader will compare across
products.

## Consequences

- fantasy-hub's interim becomes non-standard and should move to the tokens.
- Four steps means coarser buckets than five. For a 12-way finish rank that is bands of
  three, which the direct labels carry anyway.
- `--chart-seq-1` sits 1.23:1 from a white card in light — nearly invisible alone. The
  hairline grid from the table spec is what separates an empty cell from a floor-value
  cell, so **a sequential grid without cell borders is off-spec.**
- Two more tokens per theme to maintain, and a flip point a binding can get wrong.

## Rejected alternatives

- **App-accent ramp.** Reads raw ramp steps and makes the same chart mean different things
  in different products.
- **Accent-at-opacity everywhere.** No fixed floor or stop count, so two consumers
  disagree while both believing they are compliant.
- **Five steps.** Adjacent contrast drops below legibility without cell borders.
- **A diverging ramp as well.** No consumer needs one yet. Add it when one does — per
  [0001](0001-platform-neutral-spec-layer.md), on demand.
