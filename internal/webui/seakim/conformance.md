# Conformance

What a binding must do to legitimately call itself SeaKim. Tiers defined in
[0008](decisions/0008-conformance-tiers.md); this is the working checklist.

**Rules version 4.1** — a binding claims conformance *to a version*, because this document
changes. Declare it alongside your own version, per
[0011](decisions/0011-versioning.md) and [0019](decisions/0019-versioning-second-pass.md):

```yaml
version: 1.2.0          # your binding
seakim_rules: "4.1"     # the rules version you were reviewed against (may lag; never lead)
```

A binding may lag. That is a legitimate, visible state — far better than lag nobody can see.

The distinction that matters is not which components you ship — it is **which rules you
are allowed to adapt.** A rule that may bend per platform is a guideline. A rule that may
not is the identity. Conflating them is how a system dissolves, one reasonable exception
at a time.

Building for a new platform? Read
[`CONTRIBUTING-A-BINDING.md`](CONTRIBUTING-A-BINDING.md) first — this document is the
contract, that one is the route through it. Per
[0010](decisions/0010-bindings-are-contributed-not-owned.md) the team that needs a platform
owns its binding; nobody here is queued to write it.

**A check asserts an outcome; the rule that produced it usually stays judgement.**
[0012](decisions/0012-conformance-checks-ship-with-rules.md),
[0017](decisions/0017-distribution-interval-glyph.md),
[0018](decisions/0018-raised-shadow-direction.md),
[0019](decisions/0019-versioning-second-pass.md),
[0020](decisions/0020-preview-surfaces-are-gated.md), and
[0021](decisions/0021-loading-states.md) are all this shape — a machine-checkable
gate catches a *symptom* (a stale version claim, a blank preview card, an indefinite
rotation used as a loader) while the rule it serves (bump when you should; show every
component; share a domain; *is this skeleton shaped like the content it stands in for?*)
remains a discipline review enforces. Read a green gate as "the outcome held", not "the
rule was obeyed".

**Not every repo gate is a binding obligation.** `tool/preview-check.mjs`
([0020](decisions/0020-preview-surfaces-are-gated.md)) renders *this* repo's gallery,
`/next`, and `/flutter` — it is a gate on this repo, not something a contributed binding owes.
A SwiftUI binding owes SeaKim its Tier 0 conformance, not three preview surfaces (per 0010).

---

## Tier 0 — non-negotiable

No platform exception, ever. Break one and the binding is not SeaKim.

- [ ] **`0px` corners** on everything that contains content. Round only where the shape
      is conceptually round: avatars, status dots, switch tracks, count pills. No 4px
      radius anywhere, at any size.
- [ ] **Borders define, shadows lift.** In-flow surfaces get a 1px hairline and no
      shadow. A shadow promises the thing floats above the page. The only concession is
      the raised shadow on bars that scroll over content.
- [ ] **One accent hue live at a time**, bound per app, and the shared layer is
      achromatic. If two things on a screen compete for a primary *action*, one of them is
      wrong — but a systematic identity fill (own-message bubbles, selected rows) may
      repeat, because it marks a category, not a call to action. (See
      [0026](decisions/0026-accent-as-ownership-fill.md).)
- [ ] **Semantic tokens only.** No component reads a stone step, a ramp step, or a
      literal colour. This is the rule everything else depends on — theming and per-app
      hue rotation both break the moment it is violated.
- [ ] **Press is a scale, not a ripple.** 0.97 at 80ms, on every clickable thing.
- [ ] **Focus is always visible.** 2px accent ring, 2px gap. Never suppressed, never
      replaced by a colour swap alone. Full rules in
      [`guidelines/accessibility.md`](guidelines/accessibility.md).
- [ ] **44px minimum touch target** on coarse (touch) pointers, whatever the density
      says. The 44px floor is a touch-surface requirement; a precise pointer (desktop
      mouse) may be denser, which is how SeaKim keeps its compact 34px controls. On a
      coarse pointer a visible mark may still render smaller *only* when its hit area —
      pointer and assistive-technology bounds — reaches 44px; the mark may shrink, the
      target may not. (See [0023](decisions/0023-sub-floor-chrome-hit-area.md).)
- [ ] **Both themes work**, neither derived from the other. (See
      [0005](decisions/0005-light-mode-is-first-class.md).)
- [ ] **Reduced motion collapses all durations to zero** and press scale to 1.
- [ ] **Sentence case, verbs on buttons, no emoji in product UI.**
- [ ] **No gradients.** The dialog scrim is the only exception.
- [ ] **No floating action button.** (See
      [0002](decisions/0002-no-floating-action-button.md).) A Material-based binding must
      actively neutralise it, not merely avoid it. A labelled, square, secondary
      scroll-position utility (jump-to-latest, back-to-top) that floats transiently with
      `--shadow-popover` is *not* a FAB and is permitted. (See
      [0027](decisions/0027-non-primary-floating-affordance.md).)
- [ ] **No spinner as a loading affordance.** No indefinite rotation; show a skeleton
      (when you know what is arriving) or the labeled loading state (when you know only
      that something is). A Material-based binding must actively neutralise
      `CircularProgressIndicator`, not merely avoid it. (See
      [0021](decisions/0021-loading-states.md).)

## Tier 1 — adapt if you must, and write down why

The intent is fixed; the mechanism is the platform's business. **An undocumented
adaptation is a Tier 0 violation in practice** — the point of this tier is the paper
trail, not the latitude.

| Concern | Fixed intent | Free to differ |
| --- | --- | --- |
| Breakpoints | Measured **container** width; sm/md/lg at 640/1024 | The measuring mechanism |
| Text input internals | SeaKim chrome, hairline border, inset focus ring | Selection handles, context menu, autofill may come from the platform |
| Icon delivery | Phosphor, with the weight rules below | Webfont, bundled font, or SVG set |
| Overlay species | Centred panel at `md`+, bottom sheet at `sm`, and a trigger-anchored popover (modal or non-modal) per [0022](decisions/0022-anchored-popover-species.md) | Route, portal, or overlay mechanism |
| Springy curves | Enters overshoot, exits never | Exact felt overshoot — compositors differ |
| Accessibility API | Every control labelled, focusable, announced | ARIA, Semantics, accessibilityLabel |
| Pointer affordances | Present on pointer devices | Native cursor and scrollbar styling is fine |

### Icon weights

- Regular at 20px is the default. 16px in dense rows, 24px in tab bars, 32px+ only in
  empty states.
- `fill` marks **active** — current tab or nav item, saved, locked. Nothing else.
- `bold` at 14px and below, and inside solid accent fills.
- `duotone` only in empty states and slides.
- Never mix weights within a row or a nav.

## Tier 2 — component inventory

### Mandatory (14)

Every screen in both kits needs these. A binding without them is incomplete.

`Icon` · `Button` · `IconButton` · `Card` · `Badge` · `Field` · `Input` · `Checkbox` ·
`Switch` · `Select` · `Dialog` · `Toast` · `EmptyState` · one navigation shell

A responsive binding needs both navigation shells (side nav and tab bar), since the
swap between them is Tier 0 responsive behaviour.

### Expected (9)

Ship unless the platform genuinely has no use case.

`Avatar` · `Stat` · `Tag` · `Textarea` · `Radio` · `SegmentedControl` · `Tabs` ·
`Tooltip` · `Table`

### On demand (4)

Specced so they are consistent whenever they arrive.

`Slider` · `DatePicker` · `Range` (the interval/distribution glyph, [0017](decisions/0017-distribution-interval-glyph.md)) · `Sheet` as a component distinct from a `Dialog` mode

### Forbidden (1)

`FloatingActionButton`

### Naming

Component **names** are fixed across bindings. **Prefixes and API shape** are not.
`SkButton`, `Button`, and `SKButton` are all fine; `PrimaryButton` is not. A reader should
move between two bindings and the specs without a translation table. Where a binding
renames for platform correctness, the mapping goes in its readme —
`Radio` becoming `SkRadioGroup` in Flutter is the pattern.

---

## Current state

Honest rather than flattering.

Rules layer: **SeaKim 1.0**. Each binding declares the rules version it conforms to, and a
binding that lags is behind rather than broken — see
[0011](decisions/0011-versioning.md).

Bindings are **claimed** or **unclaimed**, not a roadmap — an unclaimed platform is one
nobody has needed yet, not one we owe you.

| | React | Flutter | Next.js | SwiftUI · Compose · other |
| --- | --- | --- | --- | --- |
| Status | Reference binding | Written; unbuilt (see 3) | Adapter over React | Unclaimed |
| Version | 1.0.0 | 1.0.0 | 1.0.0 | — |
| Targets rules | 1.0 | 1.0 | 1.0 | — |
| Conforms to | SeaKim 1.0 | SeaKim 1.0 | SeaKim 1.0 | — |
| Tier 0 | Pass, except light unverified | Pass, except light unverified | Inherits React | — |
| Tier 1 documented | Partial | Yes, in `flutter/README.md` | Yes, in `next/README.md` | — |
| Mandatory 14 | 14 / 14 | 14 / 14 | Inherits React | — |
| Expected 9 | 9 / 9 | 9 / 9 | Inherits React | — |
| On demand | 2 / 3 | 2 / 3 | — | — |
| Forbidden | Clean | Clean | Clean | — |

### Open non-conformance

**1. ~~Light mode is unverified everywhere.~~ Audited 2026-08-04.**

Every specimen card and guideline page now carries a theme toggle (`theme-toggle.js`), so
"reviewed in both themes" is possible rather than aspirational. Slides remain exempt —
projected, dark by design.

One real Tier 0 failure was found and fixed: **disabled state was `opacity: 0.4`**, which
survives dark (everything fades toward the near-black page, and the fill/text relationship
holds) and collapses in light (everything fades toward white, so a disabled accent button
becomes pale-on-pale, well under 4.5:1). Replaced with `--fill-disabled`,
`--text-disabled`, and `--border-disabled` in both themes, applied in `Button`,
`IconButton`, and `Tag`.

That is exactly the class of bug 0005 predicted: a value that only works in one theme was
never semantic. Remaining: the same opacity pattern still appears in `Checkbox`, `Radio`,
`Switch`, `Input`, `Select`, and `Textarea` — lower risk, since those fade a neutral
surface rather than an accent fill, but they should move to the tokens too. The Flutter
binding has the same pattern and the same fix pending.

**2. ~~Flutter does not build.~~ Resolved.** The `phosphor_flutter` dependency is gone;
the icon font is bundled and code points are generated into `sk_icons.g.dart`. See
[0009](decisions/0009-bundle-phosphor-icon-font.md). `flutter analyze` is clean on 3.44.x.

Invariants to preserve when touching those widgets:

- `SkGlyph` is our own type, carrying every weight a glyph can be drawn in. Icon fields on
  `SkButton` and `SkIconButton` are typed `SkGlyph`, never a raw `IconData`.
- Code points are **generated**, never hand-typed. Run `dart run tool/gen_icons.dart`.
- Duotone draws two layers in one colour, backdrop knocked back to 0.20 — that is what
  keeps it reading as one mark.

**3. `sk_icons.g.dart` cannot be generated — the Flutter binding does not compile.**

`tool/gen_icons.dart` reads `tool/phosphor_codepoints.json`, and that file is **not in the
repo**. Without it the generated `sk_icons.g.dart` cannot exist, and every widget that
references `SkIcons` — `SkCheckbox`, `SkToast`, `SkDialog`, `SkSelect`, `SkStat`,
`SkEmptyState`, `SkTable` — has an unresolved import.

This is a consequence of [0009](decisions/0009-bundle-phosphor-icon-font.md) that was not
called out when it was accepted: dropping the package moved icon delivery in-house, which
means the code point table is now **our** checked-in artefact. It has to be extracted once
from the bundled `.ttf` files and committed, exactly as `palette.g.dart`'s inputs are.

Two things are needed, and neither can be produced from here:

- `flutter/assets/icons/Phosphor-{Regular,Bold,Fill,Duotone}.ttf` — MIT, from the Phosphor
  release.
- `flutter/tool/phosphor_codepoints.json` — extracted from those fonts, then
  `dart run tool/gen_icons.dart`.

Until both land, treat the Flutter binding as **written and reviewed but unbuilt**.

**4. Font binaries are not committed.** `flutter/pubspec.yaml` declares 10 text `.ttf`
assets plus 4 Phosphor icon fonts, none of which are in the repo. Outfit, Plus Jakarta
Sans, and IBM Plex Mono are open-licence Google Fonts (SIL OFL 1.1); Phosphor is MIT. A
real build needs them dropped into `flutter/assets/fonts/` and `flutter/assets/icons/`.
Licence notices are already committed and wired to `showLicensePage()`. The web bindings
fetch their fonts at runtime, so only Flutter is blocked on this.

**5. ~~Tokens have two sources.~~ Phase 1 done.**

`tokens/src/color.tokens.json` is now the single source of truth for colour, and
`tool/build-tokens.mjs` emits `tokens/colors.css`, `tokens/theme-light.css`,
`tokens/apps.css`, `flutter/lib/src/tokens/palette.g.dart`, and
`tokens/generated/colors.ts` from it. Verified name-for-name against the previous
hand-written files: **zero tokens lost or added.**

The oklch chroma-reduction gamut mapping now lives in one place instead of being a rule
each binding reimplements — the biggest win of [0007](decisions/0007-token-source-format.md),
and the thing that would have drifted first.

`flutter/tool/gen_tokens.dart` is now redundant and should be deleted once the team is
comfortable with the new pipeline. Phases 2 and 3 (dimension, then typography and motion)
remain, and are lower value — those files are stable and rarely touched.

`node tool/build-tokens.mjs --check` fails on stale output; wire it into CI when CI exists.

**6. ~~`Slider` and `DatePicker` unbuilt.~~ Built in both bindings.**
`Table` exists in **both** bindings per [0003](decisions/0003-tables.md) —
`components/data/Table.jsx` and `flutter/lib/src/widgets/sk_table.dart`. Sortable, two
densities, and the `sm` species swap driven by three per-column declarations
(`identifying`, `survives`, `secondary`). Disabled tokens landed in both bindings at the
same time. Still to do: replace Bench's three hand-rolled precursors with it.
`Slider` ([0006](decisions/0006-slider.md)) and `DatePicker`
([0004](decisions/0004-date-and-time-selection.md)) remain specced and unbuilt, with
anatomy visualised in
[guidelines/proposed-components.html](guidelines/proposed-components.html).

**7. ~~No binding neutralises the FAB.~~ Done.** `SkMaterialTheme` sets
`floatingActionButtonTheme` to zero radius, zero elevation, and disabled colours, so a FAB
that survives migration is visibly wrong in review rather than quietly almost-right, per
[0002](decisions/0002-no-floating-action-button.md).

**8. `Sheet` is not a distinct component.** The last on-demand item from
[0008](decisions/0008-conformance-tiers.md). Today a sheet is a mode of `Dialog` —
`showSkSheet` in Flutter picks bottom sheet at `sm` and centred panel from `md` up, and
React's `PlayerSheet` does the same by hand. That works, so this is not urgent; promote it
when a second product needs a sheet that is not a dialog.

**9. ~~`Range` is built in both bindings, pending 0017.~~ Done, [0017](decisions/0017-distribution-interval-glyph.md) accepted.**
The interval/distribution glyph — `components/data/Range.jsx` and
`flutter/lib/src/widgets/sk_range.dart`, against `spec/Range.md`. Its two load-bearing
rules (siblings share a domain; exactly one promoted instance) are deliberately not
machine-checkable and live in review, not lint.

---

## Reviewing a new binding

In order. Each step is cheap and catches a different class of error.

1. **Grep for literals.** Any hex colour, any `px` value that duplicates a token, any
   radius above zero. This one check catches most Tier 0 violations.
2. **Screenshot the same screen in all four combinations** — dark and light, two apps.
   Contrast and accent rotation both fail visibly here.
3. **Resize through the breakpoints.** Navigation must swap, not shrink. Overlays must
   change species. Nothing may be reachable only by hover.
4. **Tab through every screen.** Focus ring visible at every stop, in order, never
   trapped.
5. **Turn on reduced motion.** Everything still works, nothing animates.
6. **Read the copy against the voice guide.** Sentence case, verbs on buttons, no
   exclamation marks, no emoji, errors as three facts.
7. **Check the Tier 1 table is documented** in the binding readme. Undocumented
   adaptation is the failure this whole tier exists to prevent.
