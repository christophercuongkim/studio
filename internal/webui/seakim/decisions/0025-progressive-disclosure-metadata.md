# 0025 — Metadata on demand is allowed; the disclosure must have a non-pointer path

- **Status** Accepted
- **Date** 2026-08-22
- **Affects** `guidelines/accessibility`, `spec/`; every binding

## Context

Dense, high-volume surfaces hide secondary metadata until asked. A chat that stamped
every bubble with a visible timestamp would be unreadable; Messenger, iMessage, and
Slack all hide the per-message time behind a tap or a hover and show a coarse separator
only on a gap. The chat build did the same — tap a bubble to reveal its time, long-press
to open the reaction picker — and hit two unwritten questions.

1. **Is hiding metadata behind an interaction allowed at all?** SeaKim has no rule on
   progressive disclosure of secondary information. The instinct to reduce noise is
   right, but "the timestamp is one tap away" quietly assumes a pointer and sighted use.
2. **What about the interaction itself?** A tap-to-reveal and a long-press-to-react are
   *pointer* gestures. The accessibility rules already say that anything reachable only
   by pointer "does not exist" for a keyboard or assistive-technology user. A timestamp
   that only a tap reveals is invisible to a screen reader; a reaction picker that only a
   long-press opens is unreachable from a keyboard.

The tension is real: the density argument for hiding metadata is sound, and the
accessibility argument against pointer-only disclosure is also sound. They are
reconciled by separating *what is shown* from *what is announced*.

## Decision

**Metadata may be disclosed on demand — visually — provided it is always reachable
non-visually as the element's accessible description**, and **any pointer-only affordance
carries a documented non-pointer trigger.**

- **Time-on-demand.** A bubble may hide its timestamp from view and reveal it on tap. The
  timestamp must nonetheless be reachable non-visually at all times — but as the element's
  **accessible description**, not folded into its accessible name. The distinction
  matters: a name is announced on every traversal, so putting the time there stamps
  "3:42 PM" onto all fifty messages a screen-reader user arrows past — trading sighted
  density for AT verbosity, which is not an improvement, only a relocation of the noise. A
  description is available on demand (the AT user asks for detail on the message they care
  about), which preserves the real principle — the value is never pointer-only — without
  making the AT pass the loud one. The visual reveal stays a sighted-density convenience,
  not the source of truth.
- **Non-pointer trigger for gestures.** Any hover- or long-press-driven action (the
  reaction picker) must also be reachable without a pointer: the host element is
  focusable, an activation key opens the same affordance, or a visible control appears in
  the focus/hover state. "Long-press only" is not a complete implementation.
- **No animated reveal of a value being read.** The disclosed metadata cuts in; it does
  not count up or slide, per the existing "never animate a value the user is reading"
  rule.

## Consequences

- Chat keeps its clean, timestamp-free column and its long-press reactions, and gains a
  keyboard/AT path to both — the timestamp in the accessible description, the reaction
  picker behind focus-plus-key or a visible affordance.
- Any surface tempted to hide metadata behind a gesture now has a rule to meet rather
  than an omission to exploit.
- Per 0012, the automatable half is genuinely weak and the ADR should say so plainly. A
  static check can, at most, assert that a component declaring a hidden-then-revealed
  value also carries that value in its accessible description — but nothing in source
  *declares* "this has a hidden timestamp," so even that is closer to a per-component test
  someone must remember to write than a rule the gate enforces. The half that actually
  protects users — "the gesture has a non-pointer path" — is human review, full stop.
  This ADR is therefore mostly a guideline with a manual review, not a gated rule, and is
  honest about it.
- Versioning (0019): **Minor**. It adds an accessibility allowance; no Tier 0 rule
  changes.

## Rejected alternatives

- **Require all metadata to be permanently visible.** Kills the density that makes a
  chat or a log readable, and pushes teams to invent the hiding anyway, unspecified.
- **Allow pointer-only disclosure.** Directly contradicts the rule that pointer-only
  affordances do not exist for keyboard and AT users; a timestamp no screen reader can
  reach is not "disclosed", it is missing.
- **Make the reveal a persistent per-message toggle in state.** Solves nothing for AT
  (the value is already reachable via the description) and adds state to every message for
  a sighted convenience the description already delivers for free.
