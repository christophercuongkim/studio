# Range

A single value shown with its interval — floor, expected, ceiling — on a scale shared
with its siblings. Per [0017](../decisions/0017-distribution-interval-glyph.md).

**Not for:** an adjustable value — that is a `Slider`, which is an input with a hit area
and reads as operable. Not for magnitude-in-a-grid (more vs less) — that is the
`--chart-seq` heatmap per [0015](../decisions/0015-sequential-chart-ramp.md). Not for a
single exact number with no range — that is a `Stat`.

## A band and a marker on a shared track

```
  Chase    5.8 ██████████▉█████ 28.2      <- floor / ceiling numbers in adjacent cells
                    ^ mid marker
  domain 0 ─────────────────────── 30      shared across every row

  +--------##########|##########------+
  ^track            ^marker           ^end
  low───────────────mid──────────────high
```

| Part | Treatment |
| --- | --- |
| Track | Full domain width, `--surface-inset`, 1px `--border-subtle`. `--radius-none` — ends square. |
| Band | From `low` to `high`. `--text-tertiary`. No border, square ends. Carries a `min-width` of `--border-emphasis` so a near-zero-spread interval still reads as a tight band, not a smudge. |
| Marker | At `mid`. `--border-emphasis` wide, `--text-primary`, spans just past the track. Square. |
| Promoted band | `--fill-accent` — **one instance at a time**, never a whole column (see States). |

Values are tokens, not literals: marker and band-floor are `--border-emphasis`, corners are
`--radius-none`. Track heights are the one exception (`sm` 5px, `md` 7px) — no depth token
carries a track height, so they are stated as a ratio to keep coherent: the marker spans
`~top:-1 / bottom:-1` past the track, so the glyph's bounding box is the track height **plus
2px**. State that against the two densities — a `comfortable` 44px / `compact` 34px row
(per [0003](../decisions/0003-tables.md)) has ample vertical budget for either track, but
the marker overshoot means the glyph is not flush to the track box.

Colour never encodes the value: floor/expected/ceiling read by **position**, not hue. Two
guards, and the second is the one that gets missed: no two `Range`s in a column are accent
at once, **and** a `Range` in a selected row stays grey — the row's own `--border-accent`
leading edge (0003) already owns that row's one accent.

## The domain is shared, or the bars lie

Every `Range` in a comparison takes the **same `domain`** `[min, max]`. A band's width is
read as spread; two bands drawn on different scales misrepresent it the way a truncated bar
axis does. A lone `Range` may self-scale to `[0, high]`; a column that does not share a
domain is off-spec.

## States

| State | Treatment |
| --- | --- |
| Default | Achromatic — grey band, ink marker. The whole column reads as calm structure. |
| Promoted | The **hovered or focused** row's band goes `--fill-accent`. Exactly one at a time. |
| In a selected row | **Stays grey.** Selection's accent is already spent on the row's leading border; the band does not also promote. |

There is no hover growth and no shadow — a `Range` is an output glyph, not a control.

## Responsive

The glyph fills its container width, so within a table at `lg`/`md` it simply narrows with
its column. The case that matters is the restack: per [0003](../decisions/0003-tables.md) a
record table at `sm` **changes species** — there are no cells, only a primary line, a
secondary line, and one surviving figure. "Narrows with its column" describes a model
SeaKim does not use, so the real question is what happens to a `Range` when its row
restacks.

Default: **drop the glyph at `sm`, keep the floor/ceiling figures.** A 44px-tall list row
has no vertical budget for a track plus marker overshoot, and this matches
`data-visualisation.md`'s existing rule — below roughly 240px, replace the chart with the
table it summarises. A binding that wants to keep it may attach the glyph to the surviving
figure on the primary line, but the numbers, not the bar, are what must survive.

## Accessibility

- Carries a text equivalent: an `aria-label` (defaulting to "{low} to {high}, {mid}"), and
  the exact figures live in adjacent numeric cells — never the bar alone.
- Passes the colour-alone check: the interval is legible in greyscale because position, not
  colour, carries it.
- Not focusable — it encodes nothing interactive. A promoted row's selection is announced
  by the row, not the glyph.
