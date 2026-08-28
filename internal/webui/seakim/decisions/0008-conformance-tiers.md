# 0008 — What a binding must do to call itself SeaKim

- **Status** Accepted
- **Date** 2026-08-04
- **Affects** every binding, current and future

## Context

Two bindings exist and a third is scaffolded. Nothing defines what "a SeaKim binding" is,
so the Flutter port's fidelity rests entirely on one author having read the React source
carefully. A SwiftUI author would have no idea which rules are the brand and which are
incidental.

The distinction that matters is not "which components" — it is **which rules a platform
is allowed to adapt.** A rule that may bend per platform is a guideline. A rule that may
not is the identity. Conflating the two is how systems dissolve one reasonable exception
at a time.

## Decision

Three tiers. Full checklist in [`conformance.md`](../conformance.md).

### Tier 0 — non-negotiable

Break one of these and the binding is not SeaKim. No platform exception, ever:

1. **`0px` corners** on everything that contains content. Round only where the shape is
   conceptually round — avatars, status dots, switch tracks, count pills. No 4px anywhere.
2. **Borders define, shadows lift.** In-flow surfaces get a hairline and no shadow.
   Shadow means the thing floats above the page and can be dismissed.
3. **One accent hue live at a time**, bound per app, and the shared layer is achromatic.
4. **Semantic tokens only.** No component reads a stone step, a ramp step, or a literal
   colour. This is what keeps theming and per-app hue rotation working at all.
5. **Press is a scale, not a ripple.** `--press-scale` 0.97 at 80ms on every clickable
   thing.
6. **Focus is always visible** — 2px accent ring with a 2px gap. Never suppressed,
   never replaced by a colour swap.
7. **44px minimum touch target**, whatever the density.
8. **Both themes work.** Neither derived from the other. (See 0005.)
9. **`prefers-reduced-motion` collapses all durations to zero** and press scale to 1.
10. **Sentence case, verbs on buttons, no emoji in product UI.**
11. **No gradients.** The dialog scrim is the only exception.

### Tier 1 — adapted per platform, with the reason written down

The intent is fixed; the mechanism is the platform's business. A binding that adapts one
of these must say so in its readme — Flutter's does, for text input.

| Concern | Fixed intent | May differ |
| --- | --- | --- |
| Breakpoints | Measured **container** width; sm/md/lg at 640/1024 | `ResizeObserver`, `LayoutBuilder`, `GeometryReader` |
| Text input internals | SeaKim chrome, hairline border, inset focus ring | Selection handles, context menu, autofill may come from the platform |
| Icon delivery | Phosphor, weight rules per 0008 §icons | Webfont, bundled font, or SVG set |
| Overlay species | Centred panel at `md`+, bottom sheet at `sm` | Route, portal, or overlay mechanism |
| Springy curves | Enters overshoot, exits never | Exact felt overshoot; compositors differ |
| Accessibility API | Every control labelled, focusable, and announced | ARIA, `Semantics`, or `accessibilityLabel` |
| Scroll and cursor affordances | Present on pointer devices | Platform-native styling is fine |

### Tier 2 — component inventory

**Mandatory (14).** A binding is incomplete without these; every screen in both kits
needs them:

`Icon` · `Button` · `IconButton` · `Card` · `Badge` · `Field` · `Input` · `Checkbox` ·
`Switch` · `Select` · `Dialog` · `Toast` · `EmptyState` · one navigation shell
(`SideNav` **or** `TabBar` — whichever the platform's surfaces need; responsive bindings
need both)

**Expected (9).** Ship unless the platform has no use case:

`Avatar` · `Stat` · `Tag` · `Textarea` · `Radio` · `SegmentedControl` · `Tabs` ·
`Tooltip` · `Table`

**On demand (3).** Build when a product needs one; specced so they are consistent when
they arrive:

`Slider` (0006) · `DatePicker` (0004) · `Sheet` as a distinct component rather than a
`Dialog` mode

**Forbidden (1).** `FloatingActionButton` (0002). A binding that ships one is
non-conformant, and a Material-based binding must actively neutralise it.

### Naming

Component **names** are fixed across bindings; **prefixes and API shape** are not.
`SkButton`, `Button`, `SKButton` are all fine. `PrimaryButton` is not — a reader should
be able to move between bindings and specs without a translation table.

Where a binding renames something for platform correctness, the mapping goes in its
readme. Flutter's `Radio` → `SkRadioGroup` is the pattern.

## Consequences

- A new binding has a definition of done, and reviewing one becomes a checklist rather
  than an argument.
- Tier 1 requires bindings to *document* their adaptations, which is where the honesty
  lives. An undocumented adaptation is a Tier 0 violation in practice.
- Flutter is currently non-conformant on two counts, both known and recorded: the
  phosphor / sealed `IconData` blocker means it does not build on modern Flutter, and
  light mode has never been visually verified. Neither is a design failure; both are
  tracked in `conformance.md`.
- Fixing the tier list is now a decision with a number, not a chat message.

## Rejected alternatives

- **Everything is mandatory.** Blocks a narrow binding — a SwiftUI watch app has no use
  for `SideNav` — and would make partial adoption non-conformant, which discourages
  adoption.
- **Everything is a guideline.** Then nothing is the brand and the system is a mood board.
- **Conformance as an automated test suite.** Right eventually, wrong at three bindings.
  The rules that matter most (one accent per screen, is this shadow justified) are
  judgement calls a test cannot make.
