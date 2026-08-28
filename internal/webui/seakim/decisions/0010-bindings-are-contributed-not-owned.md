# 0010 — The system is the rules; bindings are contributed, not owned

- **Status** Accepted
- **Date** 2026-08-04
- **Affects** every future binding; `conformance.md`; how this repo describes itself

## Context

The repo currently reads as though SeaKim *is* the React implementation, with Flutter as
a port of it and Next.js as an adapter for it. `conformance.md` even lists SwiftUI as
"not started", which implies somebody here owes you a SwiftUI binding.

That framing is wrong in a way that gets more expensive with every product added. It
makes the design system a **library** — and a library has to be written once per platform
by whoever maintains it, which is a queue with one server. A team that needs Kotlin
Compose either waits, or builds something off-system because waiting is worse.

It also mislocates the value. The genuinely hard, genuinely reusable work in this repo is
not the React code. It is the decisions with their rationale, the platform-neutral specs,
the conformance tiers, and the token source that generates to any language. That layer
took the arguments; the React components are one worked example of obeying it.

## Decision

**The product of this repo is the rules. Bindings are examples and contributions.**

Three consequences, stated so a team can act on them without asking:

**1. React and Flutter are reference bindings, not the deliverable.**

They exist to prove the rules are implementable and to give a new author something to
read. React is the most complete and is the place to look when a spec is ambiguous —
but "React does it this way" is not itself an argument. If React and a spec disagree,
**the spec wins and React is the bug.**

**2. A team that needs a platform owns that binding.**

Nobody here is queued to write SwiftUI, Kotlin Compose, or anything else. The team that
needs it builds it, against [`conformance.md`](../conformance.md), and contributes it
back into this repo so the next team inherits it. Reviewing a contributed binding is a
checklist, not a negotiation — that is what 0008 was for.

**3. Everything a binding author needs is platform-free, and stays that way.**

| Layer | Platform-free? | Notes |
| --- | --- | --- |
| `decisions/` | Yes | Why, with the rejected alternatives |
| `spec/` | Yes | Anatomy, states, responsive, a11y. No code, per 0001 |
| `tokens/src/` | Yes | DTCG JSON. Emits to any language, per 0007 |
| `conformance.md` | Yes | Tier 0 / 1 / 2 and the inventory |
| `guidelines/`, `readme.md` | Yes | Foundations, voice, iconography |
| `components/`, `flutter/`, `next/` | No | Reference bindings |

A rule that only makes sense as React or as Flutter belongs in that binding's readme,
not in the shared layer. If you catch one leaking upward, that is a bug in the docs.

## Consequences

- **New platforms stop being blocked on us.** The cost of adding one drops to a token
  emitter plus the widget layer, both scoped by documents that already exist.
- **The specs have to be good enough to build from without reading React.** They are not
  yet: only three exist ([Table](../spec/Table.md), [Slider](../spec/Slider.md),
  [DatePicker](../spec/DatePicker.md)), so a new author still learns most of the library
  from source. Per 0001 specs are written on demand, and **a new binding is exactly the
  demand.** Expect the first contributed binding to force a run of them, and treat the
  questions its author asks as the spec backlog.
- **Contributed bindings vary in completeness**, and that is fine — Tier 2 already splits
  the inventory into mandatory, expected, and on demand precisely so a narrow binding can
  be legitimate.
- **We give up uniformity of authorship.** Different hands will write different bindings.
  Tier 0 is what keeps them the same product; Tier 1 is where they are allowed to differ,
  as long as they say so.

## Rejected alternatives

- **Keep owning every binding centrally.** Highest consistency, and a queue with one
  server. The first team that cannot wait builds off-system, and then there are two
  systems.
- **Publish the rules and accept no contributions back.** Every team solves the same
  problems privately, and nothing compounds.
- **Declare React canonical and require others to match it exactly.** Re-privileges one
  platform, which is the same mistake 0007 corrected in the token layer. It also bakes
  React's accidents — its prop names, its state model — into platforms where they are
  wrong.
