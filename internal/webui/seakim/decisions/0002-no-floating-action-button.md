# 0002 — No floating action button

- **Status** Accepted
- **Date** 2026-08-04
- **Affects** every binding; `flutter/SkMaterialTheme` in particular

## Context

Material assumes a FAB and gives it a prominent slot in `Scaffold`. SeaKim has no
concept of one. Now that `SkMaterialTheme` lets an existing Material app adopt the
SeaKim look without rewriting widgets, that app will arrive with a FAB already in it
and the system has no answer.

A FAB is, structurally, three things SeaKim rejects at once:

1. **A circle.** Corners are `0px`; the only round shapes are ones that are
   *conceptually* round — avatars, status dots, switch tracks, count pills. A button is
   not conceptually round.
2. **A shadow on something that lives in the layout.** "Borders define, shadows lift"
   means a shadow promises the thing is above the page and dismissible. A FAB is
   permanent chrome wearing a popover's costume.
3. **An unlabelled primary action.** Icon-only controls must carry a label and a
   tooltip — and tooltips do not exist on touch, which is the only place a FAB lives.

Squaring it off does not rescue it. A 56px square with a shadow, floating over content,
is a worse object than the circle: it reads as a dialog that lost its dialog.

Meanwhile the primary action already has a documented home at every breakpoint:

- `sm` — sticky footer bar above the tab bar. Already used by Voyage checkout, trip
  detail, and Bench lineup.
- `md` / `lg` — trailing side of the top bar, or the section header for a section-scoped
  action.

## Decision

**No floating action button, on any platform.** There is no `SkFab`, and no
`SkButton` variant that produces one.

The primary action of a screen lives in exactly one of:

| Breakpoint | Home |
| --- | --- |
| `sm` | Sticky footer bar, full width or leading-aligned, `--shadow-raised` only |
| `md`, `lg` | Top-bar trailing slot, or the owning section's header row |

The sticky footer is the sanctioned exception to the shadow rule and already is one:
`--shadow-raised` exists for "bars that scroll over content". It is a full-width bar
anchored to an edge, not a floating object, and it can hold a labelled button.

**Material adopters:** `SkMaterialTheme` must neutralise the FAB rather than restyle
it. Set `floatingActionButtonTheme` to a shape of `RoundedRectangleBorder` with zero
radius and `elevation: 0`, so a FAB that survives migration is visibly wrong in review
instead of quietly looking almost-right. Migration guidance: move the action to a
`bottomNavigationBar`-adjacent footer or the `AppBar` actions.

## Consequences

- A Material app cannot adopt SeaKim without touching its FAB. This is intentional and
  should be the first item in any migration checklist.
- Screens with a genuinely dominant single action lose a bit of visual punch on mobile.
  Acceptable: a full-width footer button is a *larger* target than a FAB and is labelled.
- Patterns that lean on a FAB's speed-dial expansion have no equivalent. If one is
  ever needed, it is a bottom sheet of labelled actions — which is already specified.

## Rejected alternatives

- **Accept a squared FAB with rules.** Loses the radius rule and the shadow rule to buy
  a component the system does not need, on the one breakpoint where a better
  alternative already ships.
- **Accept it only in Bench.** Per-app component divergence is how a shared system
  becomes two systems. The readme already forbids app-specific components in the shared
  library.
- **Allow it as a documented escape hatch.** An escape hatch in a system this young
  becomes the default within two sprints.
