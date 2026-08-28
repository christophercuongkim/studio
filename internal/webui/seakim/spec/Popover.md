# Popover

A contextual overlay anchored to the trigger that opened it — it sits next to that
element because its whole meaning is "this, here". Per
[0022](../decisions/0022-anchored-popover-species.md).

**Not for:** a modal that owns the whole screen — that is a `Dialog`, a centred panel
under a scrim that ignores where it was opened from. Not for a large surface pinned to
the bottom edge on a phone — that is a `Sheet`. A popover severs neither the spatial link
to its trigger nor the flow the user is in; when either of those is the point, it is the
wrong species.

## Two modes

The mode is not cosmetic: the scrim *is* the modality, and the modality fixes the focus
rule. A modal popover is a container the user has entered; a non-modal one is a
description or a suggestion attached to a trigger they are still driving.

| Mode | Scrim | Focus | Dismissal | Example |
| --- | --- | --- | --- | --- |
| Modal | `--surface-scrim` | moves into the content, trapped, restored to the trigger on close | scrim tap, Escape, outside-press | a reaction bar over a message |
| Non-modal | none | **stays on the trigger** — it never steals focus | Escape, outside-press, trigger blur | a tooltip, an autocomplete list |

A tooltip that steals focus is broken for every assistive-technology user, so the
non-modal mode must not move focus in. Where a non-modal popover carries nothing operable
(a pure tooltip) it is also pointer-inert; an autocomplete list is not, because its
suggestions must be clickable.

## Positioning and flipping

Both modes position against the trigger's rect and open below it by default, offset by a
hairline gap. The side is resolved on **both axes** so the surface stays on-screen:
vertically it flips to open upward when below would overflow the viewport bottom and there
is more room above; horizontally it anchors the surface's left edge to the trigger when the
trigger sits in the left half of the viewport, and its right edge when the trigger sits in
the right half — so a trigger hugging the right edge opens leftward rather than off-screen.
Resolution is **approximate** — the decision is made from the trigger's rect against a
fixed space budget and the viewport centre, not from the surface's measured size, so a
popover larger than the budget can still clip. A caller that pins a side passes `placement`
and takes over both axes (no resolution).

## Dismissal

- **Escape** dismisses in both modes.
- **Outside-press** dismisses in both modes — the scrim in the modal mode, a transparent
  tap-away catcher in the non-modal one.
- **Trigger blur** additionally dismisses the non-modal mode: when focus leaves both the
  trigger and the popover, it closes.

## Surface

The overlay surface — `--surface-overlay`, a 1px `--border-default` hairline on every
edge, `--radius-none` corners, and `--shadow-popover`. Borders define, shadows lift: the
popover genuinely floats, so it carries the lift shadow every other floating surface in
the system (dialog, select, sheet, tooltip) carries. A floating popover with no shadow is
off-spec.

## Responsive

At `sm` the anchored shape still holds — a popover is inherently small and stays next to
its trigger. A modal popover whose content grows past what fits beside the trigger belongs
in a `Sheet` at `sm` instead; the anchoring is worth keeping only while the content is
small enough to sit next to the thing that opened it.

## Accessibility

- Meets the overlay obligations 0022 requires: the modal mode moves focus in, keeps Tab
  within the content, and restores focus to the trigger on close; both modes dismiss on
  Escape.
- The non-modal mode leaves focus on the trigger — a description or suggestion never
  becomes a place the user has to escape from.
- The modal mode marks its surface as a modal dialog; the non-modal mode marks its surface
  as a description of the trigger, not a container.
