# 0016 — Many-series trajectories: the >6 change-over-time case

- **Status** Accepted
- **Date** 2026-08-05
- **Affects** every binding; `guidelines/data-visualisation.md`
- **Depends on** [0015](0015-sequential-chart-ramp.md) — the mandatory fallback is a
  sequential grid

## Context

`data-visualisation.md` caps categorical colour at six and rules that a seventh series
"is not a colour problem, it is a chart problem: group the tail into 'other', split into
small multiples, or use a table."

Right for identity-by-category. Wrong for **change-over-time with many entities** — a
finish-rank bump chart, a fare-rank spaghetti — where the *trajectories themselves are the
subject*: who rose, who collapsed, who crossed whom. "Other" erases the entities; small
multiples break the crossings that carry the meaning; a table loses the shape. None of the
three sanctioned options covers the case, so a binding invents a pattern or ships an empty
state.

fantasy-hub hit this first: 8–12 managers over 12 seasons. It shows every trajectory in
achromatic grey with one promoted on hover — a fourth pattern the spec neither blesses nor
forbids, and one that fails the touch rule unless a non-pointer fallback exists.

## Decision

Name a **fourth** >6 treatment, scoped to change-over-time: the **achromatic trajectory
set**.

- **No per-series colour.** Every line is `--text-tertiary`. The six-colour ceiling is
  honoured *because nothing is coloured* — identity does not come from hue here.
- **One accent at a time, on demand.** Pointer hover or keyboard focus promotes one line
  to `--fill-accent` and dims the rest: "one accent hue live at a time", applied
  temporally.
- **Direct end-labels, always rendered**, so identity is never colour-alone and survives
  with no interaction at all.
- **A non-pointer equivalent is mandatory**, satisfying both the touch rule and "every
  chart has a text equivalent": a **sequential grid per [0015](0015-sequential-chart-ramp.md)**
  over the same entity×period→value data, adjacent to the chart. Not a loose "heatmap or
  table" — the grid is specified, including its cell borders and text-flip rule, so the
  fallback is as defined as the chart it backs.
- Straight segments, square caps, no smoothing — unchanged.

This reopens nothing about the palette. It moves identity off colour and onto labels plus
interaction, which is what the six-ceiling was protecting.

### The range: 8 to 15

Stated rather than left to judgement, because "judgement" means every author re-derives it
and the failure modes differ at each end:

| Count | Treatment |
| --- | --- |
| 1–6 | Categorical colour. Distinct hues are genuinely better. |
| 7 | Categorical is exhausted; prefer grouping the tail or small multiples. |
| **8–15** | **Achromatic trajectory set.** This decision. |
| 16+ | Grey lines hairball regardless. Small multiples, or the sequential grid alone. |

15 is a ceiling on legibility, not on data: above it, overlapping grey strokes stop
resolving as separate paths and end-labels start colliding. A chart with 20 entities
should show the grid and let the user pick.

## Consequences

- Every such chart ships **two** views, and the grid is the one that has to be correct —
  it is the only path for touch and screen readers.
- 0015 must land first. It has.
- fantasy-hub's pairing already matches this; its grid should move to the 0015 tokens.
- Keyboard focus promotion is per binding and easy to skip. A pointer-only implementation
  is non-conformant even with the grid present, because the chart itself must be
  navigable.
- A hard 15 will eventually be argued with. That is preferable to each author picking
  their own and none of them writing it down.

## Rejected alternatives

- **Force "other" or small multiples.** Erases the entities, or breaks the crossings that
  are the whole point.
- **Extend the categorical ramp past six.** Reopens the ceiling and yields hues nobody can
  tell apart.
- **Hover-only promotion, no fallback.** What fantasy-hub shipped. Fails on touch, which
  is most readers.
- **Leave the range to judgement.** The original open question. Costs every author the
  same derivation and produces inconsistent answers.
