# 0026 — The single-accent rule governs competing actions, not identity washes

- **Status** Accepted
- **Date** 2026-08-22
- **Affects** the single-accent gloss in `conformance.md`; `spec/`; every binding

## Context

A Tier-0 identity rule is stated, in `decisions/0008`, as: "one accent hue live at a time,
bound per app, and the shared layer is achromatic." That is the normative content, and it
is about hue *rotation* — which single accent a product is allowed to be — not about how
many times that accent may appear on a screen.

`conformance.md` restates it and appends a gloss: "if two things on a screen are
accent-coloured, one of them is wrong." The gloss is illustrative, and for its intended
subject — competing calls to action — it is right: two primary buttons, three "featured"
cards, and one loses. But read literally, as a counting rule, it misfires the moment a
surface uses accent as *identity* rather than *emphasis*.

A chat thread is where it misfires hardest. The convention for "this message is mine" is
an accent-filled bubble with on-accent text, and a normal thread shows a dozen at once. By
the gloss's literal reading, eleven of them are "wrong." They are not; the thread reads
perfectly, because those bubbles are not a dozen competing calls to action — they are one
**identity wash** repeated, every instance meaning the same thing, "mine." The same is
true of selected rows in a multi-select list.

The gloss, in other words, was written about emphasis and is being read as if it governed
every accent pixel. The fix is to say what it always meant, not to invent a second kind of
accent with its own rules. (An earlier draft of this ADR created an "identity-accent"
category gated by conditions like "the primary action must stay the most prominent accent
element" — unenforceable, and the wrong shape: a rule that needs a new category and a
judgement call every time is the failure, not the fix.)

## Decision

**Clarify the gloss: the single-accent rule governs competing accent *actions*, not the
repeated use of accent as an identity or ownership fill.** No new category, no conditions —
one sentence of scope on an existing rule.

- The `conformance.md` gloss changes from "if two things on a screen are accent-coloured,
  one of them is wrong" to, in substance: *if two things on a screen compete for a primary
  **action**, one of them is wrong; a systematic identity fill (own-message bubbles,
  selected rows) may repeat, because it marks a category rather than a call to action.*
- Own-message bubbles keep the accent fill. Selected rows are already covered — `spec/`'s
  table (`spec/Table.md`) specifies `--surface-selected` plus a `--border-accent` leading
  edge — so this clarification simply stops the gloss from appearing to contradict a spec
  that already ships; it does not re-permit anything.
- The scarcity rule keeps its full force where it belongs: competing accent *actions*.

## Consequences

- Own-message bubbles and selected rows read as conformant because the rule text now says
  what it meant, not because a reviewer waived it.
- The scarcity rule keeps its teeth for action hierarchy — two primary buttons still lose
  one — because the clarification narrows scope to actions, it does not weaken the rule.
- Per 0012, this needs **no** conformance-check change, because there is no accent check
  to change: `tool/conformance-check.mjs` deliberately excludes "one accent per screen" as
  a judgement call (it says so at the top of the file), and it stays a manual pass. The
  earlier draft's claim that "the check distinguishes accent actions from identity
  surfaces" described a gate that does not exist.
- Versioning (0019): **Minor**. This clarifies a gloss to match the unchanged normative
  rule in 0008 — it does not change the Tier 0 rule itself, and adds no stricter
  obligation. (Had it created a new accent category, that would have been a Tier 0 change
  and Major; the reframe is deliberately the lighter one.)

## Rejected alternatives

- **Create an exempt "identity accent" category with conditions.** The original draft.
  Manufactures a second kind of accent gated by an unenforceable "most prominent element"
  test, and re-permits selected rows the spec already sanctions. A category that needs
  re-arguing at each surface is the problem this was meant to solve.
- **Paint own bubbles a neutral fill.** Throws away the clearest ownership signal in
  messaging to satisfy a gloss that was written about buttons. The chat reads worse to
  obey a rule that never meant to govern it.
- **Drop the single-accent rule.** It is load-bearing for action hierarchy. The problem
  was the gloss's scope, not the rule; narrowing the gloss keeps the rule's value intact.
- **Leave it as a per-app waiver.** A waiver every chat surface needs is a missing
  clarification, and it fails audits until someone re-argues it each time.
