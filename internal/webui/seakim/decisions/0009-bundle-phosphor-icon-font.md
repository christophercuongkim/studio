# 0009 — Bundle the Phosphor icon font; drop phosphor_flutter

- **Status** Accepted
- **Date** 2026-08-04
- **Affects** Flutter binding; iconography rules in `readme.md` unchanged

## Context

The Flutter binding does not build on any single stock SDK.

`phosphor_flutter` 2.1.0 — including git main — declares
`class PhosphorIconData extends IconData`. Flutter sealed `IconData` as a `final class`
in 3.27. The SeaKim token layer meanwhile uses `Color.withValues`, which only exists from
3.27. One dependency needs an old SDK, the other needs a new one.

This is not a bug to patch around. It is a question about whether a shared design system
should have a third-party package on its critical path at all.

## Decision

**Drop the `phosphor_flutter` dependency. Bundle the Phosphor icon fonts as assets and
address glyphs through ordinary `IconData` codepoints.**

Phosphor publishes plain `.ttf` files per weight under MIT. Four are declared in
`pubspec.yaml` alongside the three text families:

| Asset | Family | Job |
| --- | --- | --- |
| `Phosphor.ttf` | `Phosphor` | regular — all UI |
| `Phosphor-Bold.ttf` | `PhosphorBold` | 14px and under, and inside solid fills |
| `Phosphor-Fill.ttf` | `PhosphorFill` | active state only |
| `Phosphor-Duotone.ttf` | `PhosphorDuotone` | empty states and slides |

Nothing subclasses `IconData`, so modern Flutter is satisfied and `withValues` stays.

### `SkGlyph` changes meaning

It was `PhosphorIconData Function([PhosphorIconsStyle style])` — a phosphor function.
It becomes a SeaKim value type holding one codepoint per weight, so `SkIcon` still owns
the weight decision and call sites are unchanged in shape:

```dart
SkIcon(SkIcons.mapPin)                            // was PhosphorIcons.mapPin
SkIcon(SkIcons.mapPin, weight: SkIconWeight.fill)
```

Blast radius is four files: `sk_glyph.dart` (new), `sk_icon.dart`, and the icon field
types on `SkButton` and `SkIconButton`. Every other widget only names the type.

### Codepoints are generated, never typed

`tool/gen_icons.dart` reads `tool/phosphor_codepoints.json` — extracted once from the
bundled fonts and committed — and emits
`lib/src/tokens/sk_icons.g.dart` — the same pattern as `palette.g.dart`, and for the same
reason: roughly forty glyphs × four weights is where a system quietly breaks.

**`sk_icons.g.dart` is deliberately not committed.** A missing file is a compile error you
cannot miss; wrong codepoints compile cleanly and render tofu. Generating it is a
documented build step, listed in `flutter/README.md` beside dropping in the fonts.

## Consequences

- **A build step before first compile.** Fetch the Phosphor font release, run the
  generator. Previously `pub get` covered it. This is the real cost.
- Four more font assets, on top of the ten text ones. All MIT or OFL; none committed here,
  for the same licensing reason as the text families.
- **The system stops depending on one package’s release cadence** — the durable win, and
  it matters more once three apps share the binding. The same deadlock cannot recur from a
  third party.
- A binding author must keep the four weight families in step with the generator. The
  generator fails loudly on a missing weight rather than silently omitting it.
- Duotone needs two codepoints per glyph in Phosphor’s font, and the generator emits both.
  Duotone is only used in empty states and slides, so a partial duotone set is acceptable
  where a partial regular set would not be.

## Rejected alternatives

- **Pin Flutter below 3.27, revert five `withValues(alpha:)` to `withOpacity`.** Ten
  minutes of work. Locks *every* SeaKim app to an old SDK to satisfy one icon library, and
  the problem returns the moment anything else needs a modern API.
- **Wait for a compatible `phosphor_flutter`.** No date, and the binding is unbuildable
  until then. Also leaves the structural exposure in place.
- **Switch icon sets to one with a maintained Flutter package.** Would change the look of
  every screen on one platform to solve a packaging problem, and break icon parity with
  the web bindings.
- **Ship SVG assets instead of a font.** More flexible, and it loses `currentColor`
  inheritance, needs per-icon asset declarations, and would diverge from how the web
  bindings deliver icons.
