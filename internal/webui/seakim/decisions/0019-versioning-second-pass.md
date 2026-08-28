# 0019 — Versioning, second pass: token revalues and honest binding versions

- **Status** Accepted
- **Date** 2026-08-16
- **Affects** `VERSION`; `CHANGELOG.md`; every binding's version file; `tool/version-check.mjs`;
  extends [0011] (does not replace its two-level model)

## Context

[0011] set up two-level versioning — one number for the rules, an independent number per
binding, each declaring the rules version it targets. The model is right. The **bump table**
and the **binding-bump discipline** have each now missed once, quietly, and 0011 itself says
the whole point is that lag should be *visible*:

- **A token revalue had no row.** At 2.1.0, `--hue-brick` was revalued 15 → 8 — a value
  change that alters rendered output everywhere the accent appears. 0011's table has Major
  (removed/renamed/Tier 0), Minor (added), and Patch (regenerated with *identical* values).
  A value that changes is none of these, so it was slotted Minor by judgement with the
  reasoning parked in the changelog. Defensible, but a table you have to argue around is a
  table with a hole.
- **A binding version drifted invisibly.** [0017] added the whole `SkRange` widget to the
  Flutter binding without moving `pubspec.yaml` `version` or `seakim_rules` — both stayed at
  `1.0` through a release that gave the binding new public API. Nothing surfaced it: no gate
  reads `pubspec.yaml`, and the binding compiled and tested green. The one field whose job is
  to make lag visible had gone stale invisibly — the exact failure 0011 was written to
  prevent. This is recorded as lesson 16, and was corrected in passing by [0018].

Two near-misses is the point at which the table wants a successor, not a third round of
case-by-case judgement.

## Decision

### 1. A token revalue within the rules is Minor; a revalue that breaks a guarantee is Major

Add the missing row, on the axis that actually draws a line. A token whose value changes
**within the existing rules** — a *retune*, like the brick shift — is a **Minor** bump: no
rule moved, nothing was removed or renamed, and no consumer's code stops working.
Regeneration that produces **identical** values stays **Patch**.

"Rendered output changes" is *not* the line — that is true of every revalue, so it draws no
line at all. The line is whether the revalue stays inside the rules or leaves them. Two ways
it leaves them, and both are **Major**, because each invalidates a guarantee a consumer relies
on:

- The revalue is **itself a Tier 0 change** — radius stops being `0`, a border becomes a
  shadow. Already Major under 0011; unchanged.
- The revalue **fails a documented contrast gate.** A stone step retuned until
  `--text-secondary` on `--surface-card` drops under 4.5:1 breaks no Tier 0 rule *by name*,
  but [0005] makes contrast structural — the retune has left the rules even though no rule
  names that pairing. This is the honest edge of "retune", and it is Major.

| Bump | When |
| --- | --- |
| **Major** | A Tier 0 rule changes; a token is removed or renamed; a component moves to a stricter tier; a token revalue fails a documented contrast gate |
| **Minor** | A token, spec, component, or Tier 1 allowance is **added**; **a token's value is retuned within the existing rules**; a rule is clarified without changing behaviour |
| **Patch** | Wording, examples, a generated output regenerated with **identical** values |

### 2. A binding's version moves with the change that alters its surface

When a binding **gains, changes, or removes public API** in a release, its own `version` and
its `seakim_rules` move **in the same commit** — never left for later, never only the rules
number. The changelog's per-binding subsection is the trigger: a release that names a binding
in a subsection is a release in which that binding's declared version moved.

`seakim_rules` **may lag** the rules version — 0011 is explicit that a binding conformant to
an older rules version is a legitimate, visible state. It **must never lead**: a binding
cannot claim conformance to a rules version that does not exist yet, and must only name one it
has actually been reviewed against.

### 3. Make lag visible, and fail the one lag that is never honest

The failure in near-miss 2 was not that the binding lagged; 0011 permits lag. It was that the
lag was **invisible**. So the fix surfaces it — and fails only the states that cannot be
honest.

`tool/version-check.mjs` gains a binding audit: it reads each binding's `version` and
`seakim_rules` and prints them against the rules `VERSION` on every run. It **fails** on two
states:

- `seakim_rules` **ahead** of `VERSION` — an impossible claim (conformance to rules that do
  not exist).
- `seakim_rules` **behind by a full Major step** — a binding claiming rules `2.x` while the
  rules are at `3.x`. A Major bump means a Tier 0 rule moved or a token vanished, so such a
  binding is either broken against the current rules or has never been re-reviewed. That is
  not the honest lag [0010] protects.

Minor and Patch lag still flow freely and are **reported, not failed** — a binding
legitimately a release or two behind stays green, with its lag printed.

What the audit **cannot** do is tell an intentional lag from a forgotten bump, or know whether
a diff touched a binding's public API — that is reading intent from code, the same class as
[0018]'s "does this footer wear the correct edge?" and [0017]'s shared-domain rule. So rule 2
stays a discipline the changelog records and review enforces; the audit is the mechanical
backstop that makes the *outcome* visible and fails only the impossible and the dangerous.

## Consequences

- **`version-check.mjs`'s inputs widen; its job does not.** Its job was always *are the
  declared version numbers mutually consistent* — and a binding's `version` and `seakim_rules`
  are declared version numbers, so auditing them is the same question over more inputs, not a
  new concern. This is **not** a licence to graft unrelated checks onto the tool on the
  precedent that it grew once.
- **A token revalue now has an unambiguous home**, on the retune-vs-breaks-a-guarantee axis, so
  the next palette retune is not re-argued from scratch.
- **Binding lag is legible instead of silent** — printed every run, failed only when ahead or a
  Major step behind. A printed line on a green run is still easily skimmed past; that is why the
  Major-step failure exists for the one case that is never intentional. The audit surfaces the
  outcome; it does not, on its own, distinguish an intentional lag from a forgotten bump, and
  rule 2 remains unenforced by machine.
- **Two cases stay uncovered, and the version numbers cannot see either.** A token added to
  *one* binding and not the other is invisible from the numbers — `--fill-accent-selection`
  exists today as a Dart getter with a documented scope limit while CSS treats it differently
  ([0013], [0014]); that divergence was deliberate and recorded, but an accidental one would
  read identically. And a **spec change with no token** — `spec/Table.md` gaining a rule — moves
  the rules version as a Minor clarification while leaving every binding non-conformant until it
  implements the rule; a binding can sit at `seakim_rules: "3.1"` without 3.1's new spec rule,
  and the audit cannot tell.
- **`seakim_rules` is a claim, not a measurement** — the deeper limit under all of this, and
  the most important admission in this ADR. It records what a binding was *reviewed* to conform
  to; no check reads the binding's behaviour to verify the claim is still true. The audit
  polices the claim's *shape* (not ahead, not a Major step behind); it cannot police its
  *truth*.
- This supersedes 0011's bump *table* (it gains rows) and adds discipline around binding bumps.
  It does **not** touch 0011's two-level model, its starting-version choice, or its changelog
  format.

## Rejected alternatives

- **Make a token revalue Major.** Too heavy. A retune within the existing rules is a visible
  change, not a break — no consumer's code stops working. Major is for removal, rename, Tier 0,
  and the contrast-gate case, where a consumer *must* look. Spending it on every retune would
  blunt the signal that makes Major mean something.
- **Forbid binding lag outright.** Contradicts [0010] and 0011 — bindings are contributed on
  their own schedules, and a binding honestly conformant to older rules is a feature, not a
  fault. The problem was never lag; it was invisible lag, and lag a full Major step deep.
- **A separate binding-version linter.** Another tool to run and keep in step. `version-check`
  already owns the version numbers; the audit belongs there.
- **Parse the changelog to enforce "a named binding's version moved."** Mechanically possible,
  but it couples the check to changelog prose formatting and still cannot tell an intentional
  no-bump from a forgotten one. The audit catches the outcome that actually matters — a stale or
  dangerously-behind `seakim_rules` — without that fragility.
- **A pre-commit hook that forces a version bump.** Rejected: a hook cannot know whether a given
  commit *should* bump — "did this diff change a token, rule, or component in a releasable way?"
  is intent, the same unknowable this ADR keeps running into, and a heuristic ("touched
  `tokens/src/` but not `CHANGELOG`? warn") fires on every work-in-progress commit in a
  multi-commit feature until the warning is trained out. Enforce the *outcome* at the gate — the
  three-way consistency check and this binding audit — not a should-have at commit time. The
  version-consistency checks are fast and false-positive-free, so they belong pre-push and in
  CI; only the checks with no false positives and no intent-guessing earn a place pre-commit.
