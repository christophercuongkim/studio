# Data visualisation

Both products are data products. Bench shows a scoring chart across a matchup; Voyage
shows fare movement over time. Neither could be built, because nothing said what a chart
looks like — which is why Bench's chart tab is currently an empty state.

This is the shortest useful answer, not a charting library. Anatomy, colour, and the
rules that keep a chart recognisably SeaKim.

---

## The problem this has to solve

Every rule in the system pushes one direction: **one accent hue live at a time**, and the
shared layer is achromatic. A chart wants the opposite — a series needs to be told apart
from another series, and that usually means several colours at once.

The resolution is to treat a series count as a design decision rather than a data
property.

## Three jobs, three marks

Before the series rules, the shape of the whole vocabulary. A chart answers one of three
questions, and each has its own mark and its own relationship to colour:

| The question | Mark | Colour |
| --- | --- | --- |
| Which one? (identity) | Series line / bar | Accent + grey, or `--chart-1..6` |
| More or less? (magnitude) | Heatmap cell | `--chart-seq-1..4` |
| How sure? (interval) | `Range` | Achromatic — position carries it |

**Magnitude is the only job that encodes by hue.** Identity uses hue to *distinguish*;
interval uses none at all. That line is what keeps the three coherent — reach for a
sequential ramp only when the answer is more-vs-less, never to tell series apart or to
show a spread. Each mark is detailed below; `Range` is the third job, not an appendix.

## One or two series: no new colour

This covers almost everything either app needs.

| Role | Treatment |
| --- | --- |
| Primary series | `--fill-accent` |
| Comparison series | `--text-tertiary` |
| Area fill under a line | The line colour at 12% |
| Reference or projection | `--border-strong`, 1px dashed |

A head-to-head matchup is you in accent, them in grey. A fare over time is the fare in
accent and its 30-day average dashed. **Nothing new enters the palette**, and the accent
still means "the thing this screen is about".

This is the default. Reach for the categorical ramp only when the data genuinely has more
than two peers.

## Three or more series: the categorical ramp

Six hues, evenly spaced, at fixed lightness and chroma so no series shouts louder than
another. Generated from `tokens/src/color.tokens.json` alongside everything else, and
deliberately **not** derived from the app accent — a chart must read the same whichever
product it is in.

| Step | oklch | Reads as |
| --- | --- | --- |
| `--chart-1` | `oklch(0.70 0.13 245)` | blue |
| `--chart-2` | `oklch(0.70 0.13 145)` | green |
| `--chart-3` | `oklch(0.70 0.13 55)` | amber |
| `--chart-4` | `oklch(0.70 0.13 320)` | plum |
| `--chart-5` | `oklch(0.70 0.13 195)` | teal |
| `--chart-6` | `oklch(0.70 0.13 25)` | red-orange |

Assign **in order, and stably**. Series 1 is always `--chart-1`, in every view, on every
refresh — a colour that moves between renders is worse than no colour. Never sort the
palette by value.

**Six is the ceiling.** A seventh series is not a colour problem, it is a chart problem:
group the tail into "other", split into small multiples, or use a table.

Status colours are **not** available as series colours. Green and red mean success and
danger everywhere else in the system, and a chart is not exempt from that.

## Anatomy

```
  120 |                                    <- axis label, --type-data --text-tertiary
      |            ...........
   80 |- - - - -.-´- - - - - - -           <- gridline, 1px --border-subtle
      |     ..-´
   40 |..-´                                <- series, 2px --fill-accent
      |
    0 +----+----+----+----+----+
       Q1   Q2   Q3   Q4                   <- axis, 1px --border-default
```

| Part | Treatment |
| --- | --- |
| Plot area | No fill, no border. It sits in the page, so it gets neither. |
| Axis line | 1px `--border-default`. Baseline only — no box around the plot. |
| Gridlines | 1px `--border-subtle`, horizontal only, 3–5 of them. Never both directions. |
| Axis labels | `--type-data`, `--text-tertiary`, tabular figures |
| Line series | 2px, square caps and joins, no smoothing |
| Bar series | Square corners. Gap is 25% of bar width. |
| Point marker | 6px square, shown at data points only when there are fewer than 20 |
| Legend | Only with 3+ series. Otherwise label the series inline at its end. |

**No smoothing on lines.** A monotone curve invents values between the points you
measured. Straight segments are honest about sampling.

**Square corners on bars, square caps on lines** — the radius rule does not stop at the
plot area.

**No shadows, no gradients, no 3D, no pie charts.** A pie is a bar chart that is harder to
read; if the parts sum to a whole and there are more than three, use a stacked bar.

## Axes and honesty

- **Bar charts start at zero.** Always. A truncated bar axis misrepresents the ratio,
  which is the only thing a bar chart is for.
- **Line charts may truncate**, because they show change rather than magnitude — but the
  axis must be labelled so the truncation is visible.
- **Label the unit once**, on the axis, not on every tick.
- **State the source and cadence** near any live chart, the same way the matchup screen
  says scores update every 30 seconds. A number without provenance is decoration.

## Interaction

| State | Treatment |
| --- | --- |
| Hover a point | 1px `--border-strong` vertical crosshair, plus a readout |
| Readout | A popover, `--shadow-popover`, positioned near the cursor but never under it |
| Selected range | `--surface-selected` band behind the plot |
| Touch | No hover. Values live in a caption or a table below the chart. |

**Never animate a value the user is reading** — the readout number cuts, it does not
count up. The line may draw in on first render at `--dur-slow`; it does not re-animate on
data change.

## At `sm`

A chart is one of the few things that genuinely cannot restack, so it adapts instead:

- Drop to 2–3 gridlines.
- Label the first and last tick only.
- Move the legend below the plot, or delete it and label series inline.
- Below roughly 240px wide, **replace the chart with the table it summarises.** A chart
  too small to read is worse than the numbers it came from.

## Accessibility

A chart is the easiest place to fail the colour-alone rule, and the most common.

- **Never colour alone.** Lines get distinct dash patterns or end labels; bars get direct
  labels or a pattern. Check it in greyscale.
- **Every chart has a text equivalent.** Either a caption stating the takeaway ("Bench
  Warmers lead by 1.0 with two players still to play") or a linked data table. This is the
  one that gets skipped.
- Chart colours are only guaranteed against `--surface-page` and `--surface-card`.
- The chart element carries a label and a description; individual points are not
  separately focusable unless they are interactive.

## What exists today

**The tokens, and nothing else.** `--chart-1` through `--chart-6` are emitted to CSS, and
to Dart as `SkChartPalette.series1`–`series6` with `SkChartPalette.ordered` for assignment
order. There is still no chart *component* in either binding, and nothing consumes the
tokens yet.

This page is the spec that unblocks writing one, and the next data screen either app needs
is the demand that should trigger it — per
[0001](../decisions/0001-platform-neutral-spec-layer.md), specs are written on demand and
so is the code.

When it is built: the tokens belong in `tokens/src/color.tokens.json` under a `chart` key,
so they generate to every binding like everything else.

## Magnitude

Per [decision 0015](../decisions/0015-sequential-chart-ramp.md). Heatmaps, rank grids,
choropleths — anything where the question is *more or less*, not *which one*.

Use `--chart-seq-1` through `--chart-seq-4`. **Never the categorical ramp**: six discrete
hues cannot express order. **Never the app accent ramp**: a sequential ramp is read as a
scale, so it has to mean the same thing in every product — someone who learns "darker means
higher" in Bench should carry that to Voyage.

- **Four steps, fixed hue 265.** Clear of every app accent and every status hue, so a
  magnitude cell is never mistaken for an accent or a state.
- **Each theme has its own steps.** Light ascends into darkness, dark ascends into
  lightness. Validated separately, never flipped.
- **Cell borders are required.** `--chart-seq-1` sits 1.23:1 from a white card, so the
  hairline grid is what separates a floor-value cell from an empty one. A sequential grid
  without borders is off-spec.
- **Label ink flips at `--chart-seq-ink-flip`** — step 4 in light, step 3 in dark. Below it
  `--text-primary`, at or above it `--text-inverse`. Every stop verified above 4.5:1.

The only sanctioned alternative is **accent-at-opacity** — `--fill-accent` at a varying
alpha — and only for a one-off tint deliberately tied to its own screen. Not for anything a
reader will compare across products.

## More than six series over time

Per [decision 0016](../decisions/0016-many-series-trajectories.md). The six-colour ceiling
still holds for identity-by-category, but a bump chart or rank spaghetti is a different job:
the *trajectories* are the subject, so "other" erases the entities and small multiples break
the crossings that carry the meaning.

| Count | Treatment |
| --- | --- |
| 1–6 | Categorical colour |
| 7 | Group the tail, or small multiples |
| **8–15** | **Achromatic trajectory set** |
| 16+ | Small multiples, or the sequential grid alone |

The achromatic trajectory set:

- **No per-series colour.** Every line `--text-tertiary`. The ceiling is honoured because
  nothing is coloured — identity comes from labels, not hue.
- **One accent at a time**, on pointer hover *or keyboard focus*: the hovered line goes
  `--fill-accent`, the rest dim. Pointer-only is non-conformant.
- **Direct end-labels always rendered**, so identity survives with no interaction.
- **An adjacent sequential grid is mandatory** — the same entity×period data as a magnitude
  grid per 0015. This is the only path for touch and screen readers, so it is the view that
  has to be right.

15 is a legibility ceiling, not a data one: above it grey strokes stop resolving as separate
paths and end-labels collide. Show the grid and let the reader pick.

## Range and interval

Per [decision 0017](../decisions/0017-distribution-interval-glyph.md). A single value shown
with its uncertainty — a floor, an expected value, a ceiling — for many rows on one shared
scale. Identity is *which*, magnitude is *more or less*; this is *a value and its spread*.
The `Range` glyph is the mark.

| Part | Treatment |
| --- | --- |
| Track | Full domain width, `--surface-inset`, 1px `--border-subtle`. Square ends. |
| Band (`low`→`high`) | `--text-tertiary`. No border, square ends. |
| Marker (`mid`) | `--border-emphasis`, `--text-primary`. Square. |
| Promoted band | `--fill-accent`, one instance at a time. |

- **Achromatic, because it repeats.** A column of intervals is the shared layer, so it is
  grey — the same resolution as the many-series case above. Promote **one** row to
  `--fill-accent` on hover or focus, never the whole column and never a *selected* row (its
  accent is already spent on the row's leading border).
- **Colour never encodes the value.** Floor/expected/ceiling read by position. Magnitude by
  colour is `--chart-seq`'s job; a `Range` does not borrow it.
- **One shared domain, or the bars lie.** Every row takes the same `[min, max]`; a band's
  width is read as spread, so two scales misrepresent it like a truncated bar axis.
- **Text equivalent, as ever.** An `aria-label`, and the exact numbers in adjacent cells —
  never the bar alone. It stays legible in greyscale because position carries it.

`Range` and the many-series trajectory set are both grey-with-accent-promotion. Different
shapes, so not confusable — but a screen showing both must promote in only **one** of them
at a time, or "one accent live" breaks *across* marks instead of within one.

> **Known smell — a mark-grey token, not yet minted.** Three marks now reach for
> `--text-tertiary` to mean "the repeated, unpromoted layer" (comparison series, 0016
> trajectories, the `Range` band) and `--text-primary` for mark ink. These are *text*
> tokens carrying graphical meaning — the same drift [0013](../decisions/0013-alpha-variants-are-tokens.md)
> named, one level up. If dataviz ever needs to retune mark greys without moving body text,
> it can't. Not worth minting `--mark-quiet` / `--mark-ink` for three uses; name them when a
> fourth mark arrives or someone actually needs the retune. Recorded here so the next person
> doesn't rediscover it.
