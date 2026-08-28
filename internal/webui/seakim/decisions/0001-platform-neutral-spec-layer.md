# 0001 — Component opinions live in spec/, bindings hold only usage docs

- **Status** Accepted
- **Date** 2026-08-04
- **Affects** every binding; `components/*.prompt.md`; all future `spec/` files

## Context

Component opinions were written into `components/**/*.prompt.md` — a folder whose name
says React. The content was never React-specific: "one primary button per view",
"selected segment gets a wash, no sliding pill", "the cancel button says what keeping
means" are product decisions that a SwiftUI binding needs just as much.

The Flutter widgets already obey those opinions. Nothing records that they must, and
nothing told their author what they were — they were reconstructed by reading React
source. That works for one port by one author on one afternoon. It does not survive a
third binding or a second person.

Two shapes were proposed: `decisions/` for cross-cutting calls with rationale, and
`spec/` for one platform-neutral contract per component.

## Decision

Adopt both folders, with three constraints that were **not** in the original proposal.

**1. Specs contain no code. None.**

The proposal had each spec carry "a small snippet per binding". That reintroduces the
exact problem it is meant to solve, one layer up: three snippets per spec across
twenty-odd specs is sixty code fragments living outside any compiler, drifting
silently. Specs describe anatomy, variants, states, and behaviour in prose and tables.
How you *call* the thing is the binding's business.

**2. `.prompt.md` files stay, and get smaller.**

They become what their name implies: how to use this component *in this binding*.
Signature, an example, binding-specific gotchas, and a link to the spec. Every
sentence of cross-platform opinion moves to the spec and is deleted from the prompt.
A binding without a usage doc is unusable; a spec cannot replace it.

**3. Specs are written on demand, never in a batch.**

Do not write twenty specs this week. A spec for a component that already works
identically in two bindings documents the present and then rots. Write a spec when:

- the component does not exist yet (Table, Slider, DatePicker — see 0003, 0006, 0004), or
- a second binding is about to implement it, or
- two bindings disagree and someone has to be wrong.

**Numbers are referenced, never restated.** A spec says `--control-h-md`, not `34px`.
A spec containing a literal pixel value that also exists in a token is a future
contradiction with a date on it.

## Consequences

- Three places to look instead of one. Mitigated by the rule above: specs exist only
  where there is real cross-binding risk, so the folder stays small and every file in
  it is load-bearing.
- A spec and a binding *can* still drift, and nothing here detects it. Accepted for
  now — see 0008, which makes conformance a reviewable checklist rather than a
  compile-time guarantee. Automated spec-to-code checking is not worth building at
  three bindings.
- `.prompt.md` files must be edited to remove the opinions that moved. Until that
  happens the opinion exists twice, which is the failure mode this ADR exists to fix.
  Tracked as debt in `spec/README.md`.

## Rejected alternatives

- **One folder, `docs/`.** Blurs two genuinely different documents: a spec is current
  and gets edited, an ADR is historical and never does. Mixing them means either
  editing history or freezing the spec.
- **Specs replace `.prompt.md`.** Leaves every binding with no usage documentation and
  pushes API details into a document that is supposed to be API-free.
- **Opinions stay in the readme.** The readme is already 300 lines and covers
  foundations. Per-component anatomy would double it and bury the foundations.
- **Specs as JSON/YAML schema.** Tempting, and wrong at this size: the valuable content
  is judgement ("if two things are accent-coloured, one is wrong"), which does not
  survive being turned into a field.
