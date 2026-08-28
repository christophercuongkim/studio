# 0014 — Text selection is Tier 1, and when a Tier 1 divergence needs an ADR

- **Status** Accepted
- **Date** 2026-08-05
- **Affects** `--fill-accent-selection` / `SkColors.fillAccentSelection`; every binding's
  Tier 1 table; `decisions/README.md`

## Context

[0013](0013-alpha-variants-are-tokens.md) promoted the text-selection wash to a token in
both bindings and left one thing unsettled: React inverts glyph colour on selection, and
Flutter paints a 32% accent rect behind glyphs and leaves their style alone. The two
bindings render selected text differently.

The obvious reading is that one of them is a bug. It is not. Flutter's `selectionColor`
is a fill drawn behind the run; restyling the selected range is not something
`EditableText` does, so matching React would mean rebuilding text rendering rather than
changing a value. That is the definition of a mechanism the platform determines —
**Tier 1**, not a Tier 0 unification.

But Tier 1 latitude comes with an obligation that had not been discharged here, and it is
a contrast one. A wash that leaves glyph colour untouched is only legible if any text over
it still clears 4.5:1. Nothing in the token guarantees that. Per
[0005](0005-light-mode-is-first-class.md), a value whose safety is unguaranteed by
construction is not yet semantic.

What makes it safe today is scope, and only by accident: `fillAccentSelection` is read in
four places, all inside `SkInput` and `SkTextarea`, and both only ever render
`textPrimary`. Nobody decided that. It is true because nothing else has needed a
selection wash yet.

## Decision

**Two things, and the second is the more general one.**

### 1. The divergence stands, scoped

Text selection appearance is Tier 1. React inverts glyph colour; Flutter tints behind
unchanged glyphs. Both are conformant.

The token is narrowed from "selection wash" to **"text selection inside a form control,
over `--text-primary`"**, stated on the token itself in both bindings. It is not a general
selection treatment. The moment something needs selection over mixed content — a
selectable table cell, a code block with syntax colour, a chart label — the tint stops
being provably legible and that case needs the solid-accent treatment with an inverted
foreground, or its own token with its own contrast argument.

### 2. A Tier 1 divergence needs an ADR when it constrains the shared layer

[0008](0008-conformance-tiers.md) requires Tier 1 adaptations to be documented, and names
the binding's Tier 1 table as where. That is right for most of them and wrong for some,
and the line was never drawn:

| Divergence | Where it goes |
| --- | --- |
| Mechanically obvious, constrains nothing outside the binding | Tier 1 table only |
| Adds a constraint to a shared token or spec, **or** a reader comparing bindings would reasonably call it a bug | Tier 1 table **and** an ADR |

`LayoutBuilder` instead of `ResizeObserver` is the first kind: nobody is surprised and no
shared rule moves. This is the second kind — it puts a usage limit on a shared token, and
a SwiftUI author reading only `flutter/README.md` would never encounter it.

That is also the answer to why this is an ADR rather than a doc comment: **a constraint on
the shared layer cannot live inside one binding.** A Dart comment is the right place to
repeat it and the wrong place to keep it.

## Consequences

- Selected text looks different in the two bindings, permanently and on purpose. Anyone
  comparing them now finds this record instead of filing a bug.
- The token carries a usage limit, which means it can be misused — there is no mechanism
  stopping someone reaching for it over mixed content.
  [0012](0012-conformance-checker.md) cannot catch that; it compares values, not contexts.
  This one relies on the doc comment being read.
- Some Tier 1 divergences now cost an ADR. The discriminator above should keep that to a
  handful — it is meant to catch the ones that constrain other people, not to document
  every platform difference.
- If a real need for selection over mixed content arrives, this ADR is superseded rather
  than amended, and the successor decides the general case.

## Rejected alternatives

- **Make Flutter match React.** Requires restyling a selected text range, which
  `EditableText` does not support — so it means owning text rendering to unify an
  appearance detail. Wrong trade, and it would break
  [0009](0009-bundle-phosphor-icon-font.md)-style dependence-on-platform-internals again.
- **Make React match Flutter.** Cheap — drop the inversion — and it discards a treatment
  that is strictly more legible on the platform that can do it, to buy uniformity nobody
  asked for.
- **Leave it as a Tier 1 table line, no ADR.** What I first proposed. It records *that*
  they differ and loses the contrast reasoning and the scope limit — the two parts a
  future author actually needs.
- **Ban the tint until a general answer exists.** Blocks working form controls on an
  unbuilt hypothetical.
