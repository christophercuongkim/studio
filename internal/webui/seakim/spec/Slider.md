# Slider

Coarse adjustment along a continuous or stepped range. Per
[0006](../decisions/0006-slider.md).

**Not for:** exact values that matter — a price cap, a budget, a passenger count. Pair
with a mono `Input` or use the input alone. Not for choosing between named options; that
is `SegmentedControl` or `Select`.

## A fader, not a dial

```
  Price range                        $380-$460   <- value, --type-data, in the label row

  +----------###########################-------+
            |||                       |||        <- thumbs: 12 wide x 20 tall
            ^^^                       ^^^
  track 4px tall, --surface-inset + hairline
  filled span --fill-accent
```

| Part | Treatment |
| --- | --- |
| Track | 4px tall, `--surface-inset`, 1px `--border-subtle`. `--radius-none` — the ends are square. |
| Filled portion | `--fill-accent`, no border. From track start to thumb, or between thumbs. |
| Thumb | 12 wide x 20 tall, `--fill-accent`, 1px `--border-strong`. A vertical bar. |
| Hit area | Invisible, `--control-h-touch` tall, centred on the track |
| Ticks | 1px `--border-default` marks, only when the scale is discrete with fewer than 12 steps |
| Value | `--type-data`, in the `Field` label row. Always present. |

The thumb is **taller than the track and narrower than it is tall**. That is the whole
idea: it reads as a mechanical fader position, never as a dot or a handle, and it stays
legible at 12px wide without a shadow to separate it from the track. A round thumb needs
either a shadow or bulk to read, and SeaKim gives it neither.

## States

| State | Treatment |
| --- | --- |
| Hover | Thumb border goes to `--border-focus`. No growth, no halo. |
| Press / drag | Whole control scales `--press-scale`. The thumb does not grow. |
| Focus | Standard `--focus-ring` around the whole control, not the thumb |
| Disabled | `opacity: 0.4` |

Nothing in the layout moves on hover, and that includes the thumb.

## The value is always visible as text

A slider without a readable number is a guess. One figure for a single slider, an en-dash
range with no spaces for two (`$380-$460` renders with an en dash), in the label row.

**No value tooltip on the thumb.** It would be an overlay following the finger, hidden
under it on touch, and it breaks "never animate a value the user is reading" at exactly
the moment the number matters most. A static figure in the label row stays legible
throughout the drag.

## Range slider

- Two identical thumbs; the span between them is the filled portion.
- Thumbs may meet but never cross. At equal values, dragging further pushes rather than
  swaps.
- Clicking the track moves the nearer thumb.
- Each thumb is separately focusable and announces which end it controls.

## Keyboard

Named here so no binding invents its own:

| Key | Effect |
| --- | --- |
| Arrow left / right | One step, or 1% of the range if continuous |
| Page Up / Down | One tenth of the range |
| Home / End | Minimum / maximum |

## Responsive

Identical at every breakpoint. The hit area is already `--control-h-touch`, so no
adaptation is needed — this is one of the few components that does not change shape.

On touch, the invisible hit area matters more than anywhere else: a 12px thumb is not a
target, the 44px band around it is.

## Accessibility

- Platform slider role, with min, max, current value, and a text description of the value
  (`$412`, not `412`).
- The visible value text is associated with the control, not read as separate content.
- Range sliders expose two controls, each labelled by which end it moves ("minimum
  price", "maximum price").
