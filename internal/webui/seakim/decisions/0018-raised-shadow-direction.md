# 0018 — Raised shadow casts away from the anchored edge

- **Status** Accepted
- **Date** 2026-08-14
- **Affects** every binding; the token depth layer; `flutter/SkDepth` and web
  `--shadow-raised*`; `conformance.md`; relates to [0002]

## Context

ADR [0002] sends the primary action, at `sm`, to a **sticky footer bar** and names
`--shadow-raised` as the sanctioned exception to "borders define, shadows lift": a
bar that scrolls over content earns a shadow.

But `raised` is currently a single, direction-fixed value. In Flutter
`SkDepth.raised` is a `BoxShadow` with `offset: Offset(0, 1)`, and web
`--shadow-raised` is likewise a downward `0 1px 2px`. That is correct for a bar
anchored at the **top** of the content it covers — a top bar or header — where the
shadow falls downward, onto the content below.

A **footer** is anchored at the bottom. The content it floats over is *above* it,
so a downward shadow falls off the bottom edge of the viewport and does nothing:
the footer reads as flat, and the one place 0002 promised a lift doesn't get one.
The token that 0002 leans on can't actually satisfy 0002. This surfaced building a
Material adopter's `sm` footer, where `SkDepth.raised` produced no visible lift.

**Every current caller is already a footer.** The three voyage screens that use
`--shadow-raised` (`SearchScreen`, `TripDetailScreen`, `CheckoutScreen`) are all
`position: sticky; bottom: 0` bars, and Flutter's `SkDepth.raised` has no callers
at all. So this is not a latent risk in future code — the shadow is falling off the
viewport in shipped screens today. There is no top-bar caller to protect.

**The system already casts upward elsewhere, and that is the strongest argument
for this change.** Direction is not new to the depth layer: `sheet` casts up
(`0 -8px 32px`) because a bottom sheet floats over content *above* it; `popover`,
`dialog`, and `toast` cast down. `raised` is the anomaly for holding one direction
while serving two roles. 0018 does not introduce an upward shadow to the system —
it makes `raised` consistent with the precedent `sheet` already set.

**Why this stays inside 0002's exception rather than reopening it.** 0002 banned
the FAB partly because a shadowed object floating near the bottom of the screen
reads as "a dialog that lost its dialog." An upward-casting footer is, structurally,
a shadowed thing at the bottom casting onto content — closer to that banned object
than a flat footer is. It is still fine, for a reason worth stating so the next
person does not read 0018 as licence to shadow bottom-anchored things generally:
the footer is **full-width and edge-anchored**, so its shadow reads as *an edge
lifting*, not an object hovering. That distinction is what keeps it inside 0002's
bar exception.

## Decision

**A raised bar's shadow casts toward the content it floats over — away from the
edge the bar is anchored to.** A top-anchored bar casts downward; a bottom-anchored
(footer) bar casts upward. Blur and colour are unchanged; only the sign of the
vertical offset flips with the anchor.

**The accessor is role-named and the role is required — there is no
direction-defaulted `raised`.** A caller states which bar it is building and gets
the correctly-oriented shadow:

- Flutter: `SkDepth.raisedTopBar(brightness)` / `SkDepth.raisedFooter(brightness)`.
- Web: `--shadow-raised-topbar` / `--shadow-raised-footer`. `--shadow-raised` is
  retained as a back-compat alias of the top-bar orientation.

Role names, not an `edge` axis parameter, on purpose. An `edge` argument is
ambiguous — a reader cannot tell whether they pass the edge the bar is *anchored to*
or the edge it *casts toward*, and either read wrong silently produces the exact bug
this ADR exists to prevent, invisibly on a top bar. Two named surfaces remove the
axis entirely: there is nothing to get backwards. Direction belongs in the depth
layer, not composed inline at the call site — an inline `Offset(0, -1)` is invisible
to the token layer and drifts between bindings the moment one changes (this is the
shape [0013] rejected).

**A footer wearing the top-bar orientation is the bug.** Because there is no
defaulted `raised`, the Flutter compiler forces the caller to state a role; a footer
author cannot silently inherit the downward value.

## Consequences

- Both bindings change, **additively**. Web gains two custom properties and keeps
  `--shadow-raised` working. Flutter exports `SkDepth` (previously internal) with the
  two role-named accessors; the removed `raised` was never exported and had no
  callers, so nothing public breaks.
- One more thing a binding author must state — but a footer given the wrong role is
  visibly flat, so it fails in review rather than shipping subtly off.
- **Versioning, both levels of [0011]:**
  - **Rules: MINOR.** Two tokens added, none removed or renamed, no Tier 0 rule
    changed. `--shadow-raised` stays as an alias precisely so this is not a rename.
  - **Web binding: MINOR.** Additive custom properties; existing `--shadow-raised`
    callers keep working.
  - **Flutter binding: MINOR, not major.** `SkDepth.raised` was binding-internal
    Dart (not a rules-layer token, and not exported), with zero callers — removing it
    is not a public break, and the new exported accessors are purely additive. An
    earlier reading called this major on the assumption that callers existed; they do
    not.
  - Binding-version bookkeeping (`pubspec.yaml` `version` / `seakim_rules`) is now
    brought in line in the same change (Flutter → 1.1.0, `seakim_rules: "3.3"`) —
    [0017] had added `SkRange` to Flutter without moving either, so the declared
    conformance had gone stale invisibly. That was the second near-miss in 0011's
    scheme (the first being the brick revalue at 2.1.0); two is the point at which
    the bump table wants a successor ADR, not a third round of case-by-case
    judgement. Flagged for a follow-up, not blocking 0018.

## Enforceability

Per [0012], a rule that needs a check names it. Three distinct things here, only one
genuinely unenforceable:

- **"A raised surface states its anchor" — enforced for free.** With no
  direction-defaulted `raised`, this is a compile error in Dart (a missing method is
  not a lint), which is the decisive reason the role is required rather than
  defaulted. This is the half that matters and it costs no new check.
- **"No inline shadow offset in component code" — checkable, deferred.** A
  `BoxShadow(...)` or `box-shadow` literal with a numeric offset in component source,
  outside the depth-token files, is the same family as the existing literal-colour
  rule and would catch the "hardcode an upward value inline" drift path below. It is
  **not added now**: no component ships such a literal today (all reference tokens),
  and a new Tier 0 check is itself a stricter-conformance change that would argue for
  a major bump against this ADR's minor. If added later its home is `conformance.md`,
  per 0012.
- **"This footer wears the correct edge" — not checkable.** Knowing which edge a bar
  is anchored to means knowing the layout. This is the same class as [0017]'s
  shared-domain rule and 0012's "is this shadow justified?": judgement, and 0012 is
  right to refuse a check for it.

Per [0014]'s discriminator, this is a Tier-1-shaped mechanism decision constraining a
shared token, so it is correctly an ADR rather than a binding-readme line.

## Scope: vertical bars only

`raised` applies to the **vertical pair** — a top bar or a footer. A rail or side
nav that scrolls over content is *not* a raised surface: it is in-flow and gets a
hairline per Tier 0 #2, not a shadow. The role-named accessors deliberately offer no
`raisedLeft` / `raisedRight`, so the question "what does a left-anchored bar cast?"
has one answer — it does not take `raised` at all.

## Rejected alternatives

- **Leave `raised` downward, exempt footers.** Keeps the token simple but makes
  0002's sanctioned footer shadow a fiction — the footer looks flat, or each
  binding re-invents an upward shadow inline, which is exactly how drift starts.
- **A defaulted anchor (caller may omit the role).** Rejected. The default is safe
  for callers that already work (there are none but top bars in theory) and unsafe
  for exactly the footer case this ADR is about — a footer author who does not know
  the parameter exists gets today's broken flat footer, from a token that claims to
  have fixed it. Required is what converts the failure from silent to a compile
  error.
- **An `edge` axis parameter.** One argument, two readings ("anchored to" vs "casts
  toward"), and the wrong read is invisible on a top bar. Role names have exactly two
  valid values and no axis to reverse.
- **A separate `footer` elevation token unrelated to `raised`.** Duplicates `raised`
  for the sole difference of an offset sign; two tokens to keep in step instead of
  one role-named pair that knows its edge.
- **Hardcode an upward value on bottom bars in each binding.** The same offset
  flip, but inline and invisible to the token layer — unenforceable, and it drifts
  between bindings the moment one changes.
