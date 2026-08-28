# 0013 — Alpha variants are tokens, not per-component constants

- **Status** Accepted
- **Date** 2026-08-05
- **Affects** `tool/conformance-check.mjs`; every binding

## Context

The `literal-colour` check flagged six shadow declarations in `tokens/depth.css`. That was
a scoping bug, not a finding: [0012](0012-conformance-checks-ship-with-rules.md) scopes the
rule to *component code*, and `depth.css` defines tokens exactly as `colors.css` does. It
was simply missing from the exemption list. A literal has to bottom out somewhere, and the
token layer is where.

Fixing that exposed the real question underneath. When a component needs a colour no token
provides — accent at 32%, for a text selection — may it compose one?

The system was answering both ways at once:

```dart
c.fillAccent.withValues(alpha: 0.32)   // five sites, and no rule looked at them
```
```css
rgb(var(--fill-accent) / 0.32)          /* would have been flagged */
```

Flutter had been doing it since the binding was written. CSS could not. Neither position
was chosen; the difference was an artefact of which patterns the checker happened to match.
That is precisely the drift 0012 exists to prevent — a Tier 0 rule that binds one binding
and not another is how two interpretations become three.

The composed values were also undesigned. `0.32` meant "selection" in four places, `0.22`
was one badge border, and nothing said what either was or stopped the next component
picking `0.28`.

## Decision

**An alpha variant is a token.** A component never composes alpha onto a colour; it reads a
token that already carries it. The token layer may compose freely — that is its job.

This is the same call [0005](0005-light-mode-is-first-class.md) already made for disabled
states. Blanket opacity was rejected there for a reason that generalises: a value invented
at the point of use is invisible to the people who own the system, and it drifts.

Enforced by the `composed-alpha` rule, which covers **both** shapes — Dart's
`.withValues(alpha:)` and `.withOpacity()`, and CSS's `rgb(var(--…) / …)`. It skips the
same palette files every colour rule skips.

`tokens/depth.css` joins that exemption list.

## Consequences

- `SkColors.fillAccentSelection` is minted — accent at the selection weight, derived in
  `sk_colors.dart` where the alpha is legitimate. Four widgets read it instead of computing it.
- **A cross-binding divergence closed itself.** React's subtle badge drew
  `var(--border-subtle)`; Flutter tinted the same border with the tone's foreground at 22%.
  They had rendered differently for as long as both existed, and nobody had noticed. Flutter
  now matches React — the token was already there.
- Widening the checker to see Dart immediately caught three more violations that had been
  invisible: `SkInput`, `SkTextarea`, and `SkSelect` faded disabled controls with
  `opacity: widget.enabled ? 1 : 0.5`. The `disabled-opacity` rule had only matched lines
  containing "disabled" or "off", so the inverse phrasing read straight past it. All three
  now resolve to the disabled tokens.
- The cost lands on whoever needs the next wash: mint the token first. That is the point.

## Not settled here

CSS `::selection` paints a **solid** accent and flips the text to `--on-accent`; Flutter
tints the background and leaves the glyphs alone. Both are conventional for their platform,
but "what does selected text look like" should be one decision, and it currently is not.
Worth its own ADR if a third binding has to guess.
