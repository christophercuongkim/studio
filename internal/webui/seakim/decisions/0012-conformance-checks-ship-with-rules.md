# 0012 — Conformance checks ship with the rules; consuming repos run them

- **Status** Accepted
- **Date** 2026-08-04
- **Affects** `tool/`; every consuming app and contributed binding

## Context

Tier 0 rules are currently enforced by someone remembering them. That works at one
engineer and fails at two.

The obvious objection to putting CI here is right as far as it goes: this repo does not
build an app, so it cannot know whether *your* screen has a stray hex value. Enforcement
has to run where the code is.

But letting each consumer write its own checker produces a worse failure. Voyage writes a
literals linter, Bench writes another, a contributed SwiftUI binding writes a third — now
there are three interpretations of Tier 0 and none of them is `conformance.md`. When a
Tier 0 rule changes, three scripts need finding and editing, and the ones nobody
remembers silently keep enforcing the old rule.

A rule and its test belong in the same place. Running the test does not.

## Decision

**This repo ships the checks. Consuming repos run them.** Same split as the token
emitters: the system owns the shared logic, the consumer calls it.

| Check | Runs where | Asks |
| --- | --- | --- |
| `tool/build-tokens.mjs --check` | **Here** | Are the generated outputs stale against `tokens/src/`? |
| `tool/conformance-check.mjs` | **Consuming repo CI** | Does this code obey the Tier 0 rules that are machine-checkable? |

The first is about the rules layer being self-consistent and has no consumer dimension.
The second is invoked from each app or binding, pointed at its own source.

### What is machine-checkable, and what is not

The checker only covers rules where a false positive is rare and a violation is
unambiguous:

| Checked | Rule |
| --- | --- |
| Literal colours | Semantic tokens only — no hex, `rgb()`, or raw ramp step in component code |
| Non-zero radius | 0px except the two conceptual-round tokens |
| Hardcoded spacing | No px value that duplicates a spacing or control token |
| Blanket opacity on disabled | Disabled is a token, not an `opacity: 0.4` pass |
| Unlabelled icon controls | Every icon-only control carries a label |
| Suppressed focus | No `outline: none` without a replacement ring |
| Stale token outputs | Generated files match their source |

Deliberately **not** checked, because judgement is the point and a false positive would
train people to ignore the tool:

- **One accent per screen.** Requires knowing what the screen is about.
- **Is this shadow justified?** Requires knowing whether the thing floats.
- **Copy voice.** A linter that flags exclamation marks would also flag legitimate ones.
- **Contrast in context.** Computable in principle; needs the resolved surface behind a
  token, which means rendering. Left to a future visual check.

`conformance.md` keeps its manual review list. The script narrows what a human has to
look for; it does not replace looking.

## Consequences

- One script to maintain, and when Tier 0 changes it changes once. That is the whole
  point.
- **Consumers must opt in.** Nothing forces an app to run it, so this is a convention
  rather than a guarantee. Acceptable: the alternative is this repo owning other
  people's CI, which it cannot.
- The script will produce false positives on legitimate exceptions — the palette
  specimen cards are *supposed* to contain hex values. It needs an ignore mechanism, and
  every ignore is a small admission that a rule has an edge.
- A check that is bypassed constantly is worse than no check, because it teaches people
  the rules are optional. If an ignore list grows past a handful of entries, the rule is
  wrong, not the code.

## Rejected alternatives

- **Each consumer writes its own.** Divergent interpretations of Tier 0, and rule changes
  that do not propagate.
- **No automated checks at all.** Survivable at one engineer, and this decision exists
  because that is ending.
- **A published npm package for the checker.** Right eventually. Premature while the
  consuming repos are all in reach.
- **Enforce everything, including the judgement calls.** Produces false positives on the
  rules that matter most, which is how a team learns to pass `--no-verify`.
