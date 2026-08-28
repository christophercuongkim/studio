# 0006 — Slider anatomy

- **Status** Accepted
- **Date** 2026-08-04
- **Affects** every binding; `spec/Slider.md`

## Context

Never specified. Voyage needs a price-range filter and a departure-time window; Bench
needs nothing today but will want threshold settings. Every platform default is a round
thumb on a rounded track, which is two radius violations in the smallest component in
the system.

## Decision

**A fader, not a dial.** Thin square track, rectangular thumb taller than the track,
and the value always readable as text.

| Part | Treatment |
| --- | --- |
| Track | 4px tall, `--surface-inset` fill, 1px `--border-subtle`. `0px` corners — the ends are square. |
| Filled portion | `--fill-accent`, from the track start to the thumb. No border. |
| Thumb | **12 wide × 20 tall**, `--fill-accent`, 1px `--border-strong`. A vertical bar, not a square and not a circle. |
| Hit area | Invisible, `--control-h-touch` tall, centred on the track. The thumb is small; the target is not. |
| Hover | Thumb border goes to `--border-focus`. No growth, no halo. |
| Press | The whole control scales `--press-scale`, per the standard press rule. The thumb does not grow. |
| Focus | Standard `--focus-ring` around the whole control, not the thumb. |
| Disabled | `opacity: 0.4`. |
| Ticks | 1px `--border-default` marks, **only** if the scale is discrete and has fewer than 12 steps. Never on a continuous range. |
| Range (two thumbs) | Both thumbs identical; the span between them is the filled portion. Thumbs may meet but not cross. |

The thumb being **taller than the track and narrower than it is tall** is the whole
idea: it reads as a mechanical fader position, it never gets confused with a dot or a
handle, and it works at 12px wide without a shadow to separate it from the track. A
round thumb needs either a shadow or a large size to stay legible, and SeaKim will give
it neither.

### The value is always visible as text

A slider without a readable number is a guess. Every slider is paired with its value in
`--type-data` — one figure for a single slider, an en-dash range for two thumbs
(`$380–$460`), positioned in the `Field` label row rather than floating above the thumb.

**No value tooltip on the thumb.** It would be an overlay following the finger, hidden
under it on touch, and it violates "never animate a value the user is reading" — the
number would be in motion at exactly the moment it matters most. A static figure in the
label row is legible throughout the drag.

### Not a substitute for typing

A slider is for coarse, exploratory adjustment where the exact number does not matter
much. Where it does — a price cap, a budget — pair it with a mono `Input`, or use the
input alone. Same reasoning as the date field in 0004.

## Consequences

- The value display means a slider is never a bare control; it needs a `Field` or an
  equivalent label row. Slightly more markup, considerably more usable.
- Keyboard support is per binding: arrows step, Home/End jump to the ends, Page
  Up/Down move by a tenth. Named here so no binding invents its own.
- Two-thumb range sliders have real edge cases (crossing, equal values, which thumb has
  focus). Specified in `spec/Slider.md` rather than left to each binding.

## Rejected alternatives

- **Round thumb.** Needs a shadow or bulk to read against the track; both are off-system.
- **Square thumb, equal sides.** Reads as a checkbox that escaped.
- **Thumb grows on hover or press.** Material's move. Contradicts "nothing in the layout
  moves on hover" and makes the press feel unrelated to every other control.
- **Value in a tooltip above the thumb.** Animated, obscured by the finger on touch,
  and gone the moment you release.
