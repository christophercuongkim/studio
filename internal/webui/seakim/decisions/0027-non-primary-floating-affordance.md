# 0027 — A non-primary floating affordance is allowed, and is not a FAB

- **Status** Accepted
- **Date** 2026-08-22
- **Affects** decision 0002 (extends it), the shadow rule (`conformance.md`), `spec/`; every binding

## Context

Decision 0002 bans the floating action button. Its argument is precise: a FAB is three
rejected things at once — a circle, a shadow on something that lives in the layout, and
an unlabelled *primary* action. That reasoning is sound and stays.

A scroll utility looks superficially like a FAB and shares none of its sins. A
"jump to latest" in a chat, a "back to top" in a long feed — these float bottom-trailing
over scrolling content, but they are:

- a **square** `SkIconButton` at `0px`, not a circle;
- **labelled** ("Jump to latest"), with the tooltip/`Semantics` the icon rule requires;
- a **secondary utility**, not the screen's primary action — the primary action still
  lives in its sanctioned home (footer bar or top-bar trailing) per 0002;
- **transient**, appearing only when a scroll state calls for it (you have scrolled up
  from the bottom) and gone otherwise.

So 0002's ban does not reach it — but 0002 also doesn't *sanction* it, and the shadow
rule ("borders define, shadows lift") leaves it in a gap: it genuinely floats over
content, yet there is no named lift treatment for a floating utility that isn't one of the
anchored bars. 0018 (Accepted) defines the *raised* shadow only for edge-anchored bars —
a top bar casts down, a footer casts up — and exposes it as `raisedTopBar`/`raisedFooter`
with, deliberately, no general `raised()`. A jump-to-latest button is anchored to no edge;
it floats with margins on every side, so it has no anchor edge to cast away from and 0018's
raised role simply does not describe it. The chat build shipped one flat, which reads as an
object that forgot to say it floats.

## Decision

**A non-primary floating affordance is permitted**, distinct from the banned FAB, under
fixed constraints:

1. **Square, `0px`** — an `SkIconButton`, never a circle.
2. **Labelled** — tooltip and accessible name, always.
3. **A scroll-position utility only** — the sanctioned set is narrow and named:
   jump-to-latest and back-to-top. Not "any navigation utility," which stretches to
   swallow the ban — a "new message" or "add" is not a scroll utility because it happens
   to move the viewport, and holding a compose/add here is a FAB that 0002 forbids. If it
   is not moving the user within content they are already scrolling, it does not qualify.
4. **Transient** — driven by a scroll or view state, not permanent chrome.
5. **Lifted with `--shadow-popover`** — it floats over content, but it is not edge-anchored,
   so it takes the popover lift, not `raised`. 0018's raised role is defined only for
   edge-anchored bars (and Flutter exposes no general `raised()` to call); `--shadow-popover`
   is the honest token for a thing that floats free of any edge. A floating thing must
   promise it floats.

## Consequences

- Chat's jump-to-latest, and back-to-top on long feeds, are sanctioned and get a defined
  shadow (`--shadow-popover`) instead of sitting flat or being waved through case by case.
- 0002 keeps its full force: the test is still "is this the primary action wearing a
  popover's costume?" — and a labelled, secondary, transient scroll utility is not.
- Per 0012, the FAB check narrows from "no floating control" to "no floating *primary*
  action and no circle"; a labelled square scroll utility with a popover shadow passes.
  ("Primary" stays a manual judgement; the circle and the shadow are checkable.)
- Versioning (0019): **Major**. It changes two Tier 0 rules — it extends 0002's FAB ban
  and the shadow rule — even though the net effect is to permit a narrow, well-fenced case.

## Rejected alternatives

- **Extend the 0002 ban to cover it.** Conflates a labelled square utility with the
  unlabelled circular primary action 0002 actually rejects; it would forbid a pattern
  every long-scroll surface legitimately needs.
- **Anchor it into a bar permanently.** A permanent bar for a control that is only useful
  when scrolled-up wastes the space the rest of the time; the transient float is the
  correct affordance for a scroll-state utility.
- **Allow it but with no shadow.** Leaves it in the same unspecified gap that produced a
  flat floating object; "shadows lift" applies precisely because it floats.
