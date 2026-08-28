# Accessibility

The rules were always here — focus in `tokens/depth.css`, contrast in the colour docs,
targets in `tokens/spacing.css`, reduced motion in `tokens/motion.css`. They were never
in one place, so a binding author had to reconstruct them from four files and hope.
This is that page.

Everything below is **Tier 0** unless marked otherwise — see
[`conformance.md`](../conformance.md).

---

## Colour and contrast

| Target | Ratio | Against |
| --- | --- | --- |
| Body and UI text | 4.5:1 | Its own surface, in **both** themes |
| Text 20px+ or 17px semibold | 3:1 | Its own surface |
| Icons carrying meaning | 3:1 | Its own surface |
| Control borders, focus ring | 3:1 | Adjacent surface |
| Decorative rules, disabled | no minimum | — |

**A new accent hue** must clear 4.5:1 at `oklch(0.72 0.13 H)` on `--stone-950` *and* at
`oklch(0.56 0.14 H)` on `--stone-50` before it enters the system. That is the check in
[`tokens/src/README.md`](../tokens/src/README.md), and it is why light mode is not
optional: an accent that only passes in dark is not an accent, it is a dark-mode value.

**Colour is never the only signal.** Every status that uses colour also carries an icon,
a label, or a shape:

- `Badge` with `tone="danger"` still reads "Cancelled".
- A `Stat` delta shows an up or down arrow, not just green or red.
- A selected table row gets a 2px leading border as well as a wash.
- An invalid `Input` gets an error line in `Field`, not just a red border.

This is not a red/green colour-blindness footnote. It is also how the UI stays legible on
a projector, in sunlight, and in a screenshot pasted into chat.

## Focus

`--focus-ring` is a 2px accent ring with a 2px canvas gap. Three rules:

- **Visible-only.** Bound to `:focus-visible` or the platform equivalent, so a pointer
  click does not leave a ring behind. Never `outline: none` with nothing in its place.
- **On the control, not the child.** A slider rings the whole control, not the thumb. A
  table row action rings the button.
- **Never suppressed for aesthetics.** If the ring looks wrong somewhere, the layout is
  too tight — give it room rather than removing it.

Focus **order** follows reading order. Two places that need attention:

- **Overlays trap focus.** A `Dialog` or bottom sheet moves focus in on open, keeps Tab
  inside, restores focus to the trigger on close, and closes on Escape.
- **Hover-revealed controls are focusable.** Table row actions appear on hover from `md`
  up — they must also appear on focus, and be in the tab order regardless. `Table.jsx`
  does this with `onFocusCapture`. Anything reachable only by hover does not exist for a
  keyboard user, and does not exist at all on touch.

## Targets and pointers

- **44px minimum** for anything tappable, at every density and on every platform. A 12px
  slider thumb is legal because its hit area is 44px tall; the visible mark and the target
  are different things.
- **Density never shrinks a target.** Density 7/10 shows up as tight vertical padding and
  generous section gaps, not as small controls.
- **Hover is an enhancement.** Every hover affordance has a non-hover equivalent. This is
  why table rows change species at `sm` rather than scrolling.

## Motion

`prefers-reduced-motion` collapses every duration to 0 and `--press-scale` to 1. Already
handled in `tokens/motion.css`; Flutter reads `MediaQuery.disableAnimations`.

**Never animate a value the user is reading.** Prices, scores, times, and counts cut
instantly; their containers may animate. A number in motion at the moment it matters is
the reason there is no value tooltip on the slider thumb.

## Naming things

- **Every icon-only control carries a label.** `IconButton` requires it — it becomes the
  accessible name and the tooltip. An unlabelled icon control is a bug, not a style.
- **Tooltips are not labels.** They do not exist on touch. Anything essential lives in the
  accessible name or on screen.
- **Labels are the visible text**, matched exactly. If a button reads `Add leg`, its
  accessible name is `Add leg`, not `Add itinerary segment`.
- **Numbers announce their unit.** `24.6` alone is not information; `24.6 points` is.

## Structure

- **One `h1` per screen**, then no skipped levels. Visual size comes from `--type-*`
  roles, never from picking a bigger heading tag.
- **Real semantics, not divs with roles.** A table is a table; the `sm` list row is a
  list and announces as one rather than pretending to still be a table.
- **Landmarks** on the shell: navigation, main, and a labelled banner. Both kits do this.
- **Live regions** for anything that appears without user action. `Toast` is a status
  region; a form error is announced when it appears. A **loading state** announces itself
  **busy** and then announces **completion** when content replaces it — `role="status"` +
  `aria-busy` on the web, `Semantics(liveRegion: true)` in Flutter (per
  [0021](../decisions/0021-loading-states.md)). A **skeleton** is decorative and stays
  `aria-hidden` — the region around it owns the announcement, not each block.

## Forms

- Every control sits in a `Field`, which associates the label, hint, and error.
- **The error replaces the hint** rather than stacking, and follows the three-facts rule:
  cause, consequence, next step.
- **Errors are announced, not just coloured.** The red border is the second signal.
- **Required is marked in text**, not only with an asterisk colour.
- Placeholders are examples, never a restatement of the label — a placeholder disappears
  on focus and cannot carry meaning.

## What each binding owes

Tier 1: the API differs, the outcome does not.

| Concern | React | Flutter |
| --- | --- | --- |
| Names and roles | ARIA attributes | `Semantics` |
| Focus | `:focus-visible`, `tabIndex` | `FocusableActionDetector` |
| Live regions | `role="status"` | `Semantics(liveRegion: true)` |
| Reduced motion | media query in `tokens/motion.css` | `MediaQuery.disableAnimations` |
| Focus trap | portal + key handling | route-based, `Navigator` |

## Reviewing

Five minutes, and it catches most of what matters. Steps 3 to 5 are the same ones in
[`conformance.md`](../conformance.md).

1. **Tab through the screen.** Ring visible at every stop, order matches reading order,
   never trapped outside an overlay, always trapped inside one.
2. **Unplug the mouse and complete the main task.** Not a metaphor — actually do it.
3. **Turn on reduced motion.** Everything works, nothing moves.
4. **Check both themes.** Contrast fails asymmetrically; light is where it usually breaks.
5. **Screenshot in greyscale.** Anything that becomes ambiguous was relying on colour alone.
6. **Run a screen reader over one flow.** Names match visible text, numbers carry units,
   nothing announces as "button button".

## Known gaps

Honest rather than flattering.

- **No automated checks.** No axe, no contrast linting in CI, no a11y tests in either
  binding. The review above is manual and therefore skippable.
- **No screen-reader pass has been done** on either UI kit. The semantics are written to
  be correct; nobody has listened to them.
- **Contrast has been verified by eye and by construction** — the ramps are built to clear
  their targets — but not measured token-by-token across both themes.
