# 0022 — Anchored contextual popover is a third overlay species

- **Status** Accepted
- **Date** 2026-08-22
- **Affects** the overlay-species rule (`conformance.md`), `spec/`; every binding — `flutter/` in particular (`SkSelect`, `SkDatePicker`, `SkHoverLabel` today, a new `SkPopover` tomorrow)

## Context

The overlay-species rule fixes exactly two shapes: a centred panel at `md`+ and a
bottom sheet at `sm`. Both are *modal and screen-anchored* — they ignore where the
thing that opened them lives.

A contextual overlay that is **anchored to its trigger** is neither. A reaction bar
above a long-pressed message, a menu under a "more" button, an autocomplete list under
a field — these must sit next to the element that spawned them, because their whole
meaning is "this, here". Forcing them into a centred panel or a bottom sheet severs the
spatial link that is the point.

The system already builds this shape repeatedly and pretends it hasn't. `SkSelect` and
`SkHoverLabel` each hand-roll an `OverlayPortal` + `CompositedTransformFollower`, neither
exported nor shared; `SkDatePicker` dodges the overlay entirely and renders its calendar
inline in a `Column` below the field, which is its own bug (an anchored picker that
cannot escape its parent's clip or width). So the anchored shape appears across three
components with no primitive and **no name** — two ad-hoc overlays and one inline
work-around — and **none of the three** meets the overlay obligations (focus does not
move in, Tab is not trapped, focus is not restored, there is no Escape). That produces
two failures:

1. **Consumers re-derive it badly.** A binding consumer that needs an anchored menu
   (chat reactions were the trigger for this ADR) drops to a raw `Overlay` and, with no
   primitive to lean on, skips the overlay obligations — focus does not move in, Tab is
   not trapped, focus is not restored on close, and there is no Escape-to-dismiss. The
   accessibility rules require all four for any overlay; a hand-rolled one gets none.
2. **The surface reads wrong.** With no species definition, the ad-hoc popover picks
   the wrong tokens — a raised card surface and no lift shadow — when every *defined*
   floating surface in the system (dialog, select, sheet, tooltip) uses the overlay
   surface plus `--shadow-popover`. "Borders define, shadows lift": an anchored popover
   genuinely floats and must say so.

## Decision

**Add the anchored popover as a third overlay species**, and grow a primitive that owns
it so bindings stop re-deriving it.

The species, in the Tier-1 overlay table. The anchored popover has **two modes**, and
the split is not cosmetic — scrim presence *is* modality, and modality determines the
focus rule. A modal popover (a reaction bar) takes focus; a non-modal one (a tooltip, an
autocomplete list) must not, because it is a description or a suggestion attached to a
trigger the user is still driving, not a container they have entered.

| Species | Trigger | Surface | Dismissal | Focus |
| --- | --- | --- | --- | --- |
| Centred panel | modal, `md`+ | overlay + `--shadow-dialog` | scrim tap, Escape | trap + restore |
| Bottom sheet | modal, `sm` | overlay + `--shadow-sheet` | scrim tap, Escape, drag | trap + restore |
| **Anchored popover — modal** | **contextual + scrim** (reaction bar) | **overlay + `--shadow-popover`, hairline border** | **scrim tap, Escape, outside-press** | **moves in, restores on close** |
| **Anchored popover — non-modal** | **contextual, no scrim** (tooltip, autocomplete) | **overlay + `--shadow-popover`, hairline border** | **Escape, outside-press, trigger blur** | **stays on the trigger** |

Both modes position against the trigger's rect and flip to stay on-screen. The modal mode
moves focus in and restores it on close; the non-modal mode leaves focus on the trigger
(a tooltip that steals focus is broken for every AT user, and today `SkHoverLabel` wraps
its content in `IgnorePointer` precisely so it cannot).

**Primitive.** Add `SkPopover` (React and Flutter): given an anchor rect/target and a
child, it produces the positioned surface with the popover shadow, on-screen flipping,
outside-press and Escape dismissal, and a `modal` flag selecting the focus behaviour
above (scrim + focus-trap when set, focus-inert when not). `SkSelect` refactors onto the
modal mode, `SkHoverLabel` onto the non-modal mode, and `SkDatePicker` moves off its
inline calendar onto the primitive — none carries its own `OverlayPortal` plumbing.

## Consequences

- Consumers get a sanctioned, accessible anchored overlay instead of a raw `Overlay`
  that fails the focus and Escape rules.
- The three existing ad-hoc implementations (two hand-rolled overlays, one inline picker)
  converge on one primitive; their a11y and shadow behaviour stop drifting apart.
- Per 0012, the overlay-species conformance check must grow the two anchored rows so a
  floating surface with no lift shadow, or an anchored overlay with no Escape handler, is
  caught. The focus-mode distinction stays a manual pass — "a tooltip must not trap
  focus" is a judgement a static check cannot make.
- `SkPopover` is a new component: per 0001 it owes a `spec/` file, and per 0020 a
  preview-surface entry and manifest row before it can ship.
- Versioning (0019): **Minor**. The overlay-species rule is Tier 1, and this adds a
  species rather than changing a Tier 0 rule.

## Rejected alternatives

- **Keep two species; push contextual actions into a bottom sheet on `sm`.** A bottom
  sheet for "react to *this* message" throws away the anchoring that carries the
  meaning, and on `md`+ there is no answer at all.
- **Leave it unspecified and let bindings hand-roll.** That is the status quo, and it
  already produced three divergent implementations and an inaccessible consumer copy.
  An un-named pattern that ships three times is a spec gap, not a non-decision.
- **A primitive with no species rule.** The check has nothing to enforce, so the wrong
  surface and missing Escape reappear the first time someone bypasses the primitive.
