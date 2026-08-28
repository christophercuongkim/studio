# 0007 — DTCG JSON becomes the token source; CSS becomes an output

- **Status** Accepted
- **Date** 2026-08-04
- **Affects** `tokens/*.css`, `flutter/tool/gen_tokens.dart`, every future binding

## Context

**The primary problem is shape, not correctness: CSS as source privileges the web.**

Every non-web binding faces the same choice — parse CSS, or keep a hand copy. Parsing
means writing a correct reader for `oklch()` inside nested `[data-app]` selectors, once
per binding. Neither is a reasonable standing requirement for joining the system, and the
cost falls entirely on whoever arrives next: today Flutter, tomorrow SwiftUI.

There is also a live symptom of that pressure. `tokens/*.css` is documented as the single
source of truth, and **it is not, and has not been since the Flutter port landed.**

`flutter/tool/gen_tokens.dart` does not parse the CSS. It carries its own hardcoded
copies of the ramp steps and the app hues:

```dart
const List<(String, double, double)> _ramp = [('s050', 0.96, 0.022), …];
const Map<String, (double, String)> _apps = {'clay': (55, …), 'sea': (245, …), …};
```

Those eleven L/C pairs and four hues exist twice — once in `tokens/colors.css`, once in
Dart. Nothing checks them against each other. Change a lightness step in the CSS and the
Flutter build stays silently, confidently wrong.

That duplication is **evidence, not the argument.** It could be fixed without changing
anything architectural — make the generator actually parse the CSS. What it demonstrates
is what happens under the current shape when a second platform arrives and CSS parsing is
inconvenient: the author copies the values instead, and nobody notices for months. A third
binding would face the same fork.

## Decision

**Promote a neutral source. Adopt [DTCG](https://tr.designtokens.org) JSON under
`tokens/src/`, and generate CSS, Dart, and TS from it.** `tokens/*.css` becomes a
build output and is marked as such.

```
tokens/
  src/                    ← source of truth, hand-edited
    color.tokens.json     ← oklch values, stone hex, status ramps
    dimension.tokens.json ← space, control heights, chrome, radius
    typography.tokens.json
    motion.tokens.json
  *.css                   ← GENERATED. do not edit.
tool/
  build-tokens.mjs        ← one script, three emitters
```

### Colour stays in oklch in the source

The JSON stores `oklch(0.72 0.130 245)`, not a hex. Each emitter resolves it for its
platform: CSS emits the `oklch()` function untouched and lets the browser do it; Dart and
Swift get sRGB baked with **chroma-reduction gamut mapping**, which is the logic already
in `gen_tokens.dart` and moves into the shared build. Clipping is not acceptable — it
shifts hue and breaks the promise that every app's ramp differs only in H.

This is the part that makes the neutral source worth it: the *ramp maths* becomes shared
code instead of a rule each binding reimplements.

### Sequence it — colour first

Do not migrate all five token files at once.

| Phase | Scope | Why |
| --- | --- | --- |
| 1 | Colour + app hues | The only place duplication exists today. Fixes a real bug. |
| 2 | Dimension (space, control heights, chrome) | Cheap, and the next thing a binding retypes. |
| 3 | Typography, motion | Stable and rarely touched. Lowest value, do last. |

A half-migrated token layer is its own confusion, so each phase deletes the hand-written
CSS it replaces in the same commit. No file is a source and an output at once.

### Guardrails

- Generated CSS carries a `/* GENERATED — do not edit. Run npm run tokens. */` banner,
  matching `palette.g.dart`.
- The build is checked in as generated files, not run at install time. A binding author
  can read the CSS without a toolchain, and diffs are reviewable.
- CI (when it exists) re-runs the build and fails if output differs from what is
  committed. Until then, this is a review checklist item.

## Consequences

- A Node script becomes a dependency of changing a colour. Real cost — editing CSS
  directly is currently a one-line change with instant feedback. Mitigated by keeping the
  emitters small and the output committed.
- One more indirection between intent and result. Genuinely worse for a designer poking
  at values; genuinely better for three bindings staying in agreement.
- DTCG is a young spec and tool support is uneven. Only its file format is being adopted
  — no Style Dictionary, no third-party pipeline — so the risk is a hand-written emitter,
  not a vendor.
- The readme's "source of truth" claim becomes true, which it currently is not.

## Rejected alternatives

- **Keep CSS as source, make the Dart generator actually parse it.** Fixes the duplication
  without new infrastructure, and it is the cheapest honest option — worth taking if 0007
  is rejected. Rejected here because it makes CSS parsing a standing entry requirement for
  every future binding, and a correct reader for `oklch()` inside nested `[data-app]`
  selectors is not a small thing to get right three times.
- **Keep CSS as source, accept the duplication, add a test comparing the two.** Papers
  over the shape problem and only works for the one binding you wrote the test for.
- **Style Dictionary.** More capable than needed and adds a dependency with its own
  opinions about naming.
- **TypeScript as the source.** Neutral in practice for web bindings, absurd for a
  SwiftUI binding to consume.
