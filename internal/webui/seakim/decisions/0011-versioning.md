# 0011 — Versioning: one number for the rules, independent numbers for bindings

- **Status** Accepted
- **Date** 2026-08-04
- **Affects** every binding; `CHANGELOG.md`; how a consuming app pins SeaKim

## Context

Nothing in the repo carries a version. That has been survivable because one person is
building and consuming it, but it fails in three specific ways as soon as a second app or
a second author appears:

- **A consuming app cannot pin.** Voyage and Bench both track whatever is on the branch,
  so a token change lands in both at once whether or not either is ready for it.
- **A binding cannot say what it conforms to.** `conformance.md` changes over time. "This
  binding is conformant" is meaningless without naming the version it was checked against.
- **There is no record of what changed.** Ten ADRs describe *why* decisions were made and
  nothing describes *what shipped when*. Those are different documents.

The obvious answer — one semver number for the whole repo — is wrong here, and the reason
is [0010](0010-bindings-are-contributed-not-owned.md). Bindings are contributed by
different teams on their own schedules. A single repo version would either force every
binding to release together, or lie about which ones actually changed.

## Decision

**Two levels. The rules are versioned once; each binding versions itself and declares
which rules version it targets.**

### The rules version

One semver number in `VERSION`, covering the platform-free layer: `tokens/src/`,
`spec/`, `decisions/`, `conformance.md`, and the foundation guidelines.

| Bump | When | Example |
| --- | --- | --- |
| **Major** | A Tier 0 rule changes, a token is removed or renamed, or a component moves to a stricter tier | Radius stops being 0px; `--surface-card` renamed |
| **Minor** | A token, spec, component, or Tier 1 allowance is added; a rule is clarified without changing behaviour | `--chart-*` added; `Table` specced |
| **Patch** | Wording, examples, a generated output regenerated with identical values | Typo in a spec; a guideline gains a diagram |

**Removing or renaming a token is always major**, even if nothing appears to use it.
A contributed binding you cannot see may use it, which is the whole point of the
arrangement.

### Binding versions

Each binding carries its own semver, in whatever file its ecosystem expects
(`pubspec.yaml`, `package.json`), and declares the rules version it targets:

```yaml
# flutter/pubspec.yaml
version: 1.1.0
seakim_rules: "1.1"     # the rules version this binding conforms to
```

A binding may lag. `seakim_rules: "1.0"` on a repo at 1.2 is a legitimate, *visible*
state — it says the binding is conformant to 1.0 and has not caught up, which is far
better than the current situation where lag is invisible.

A binding **must not** claim a rules version it has not been reviewed against.

### The changelog

One `CHANGELOG.md` at the root, Keep a Changelog format, with a section per release and a
subsection per binding when a binding changed. ADRs say why; the changelog says what and
when. Both, not one.

Every entry that stems from a decision links it. A change with no ADR and no obvious
cause is a change nobody will be able to explain in six months.

### Starting version

**1.0.0.** Not 0.x. Two products ship on this, eleven decisions are accepted, and the
conformance contract is written — the qualities 0.x signals (unstable, expect breakage,
do not depend on this) are not true here. Calling it 0.x to feel humble would be
inaccurate in the direction that costs people time.

## Consequences

- Two numbers to maintain instead of none. The rules number moves rarely; binding numbers
  move on their own schedules, which is the point.
- **Token renames get expensive**, deliberately. A major bump forces every binding author
  to look. That friction is the feature — it is what stops the semantic layer churning.
- The changelog only stays useful if it is written at the time. A retrospective changelog
  is an archaeology project.
- `conformance.md` needs a rules-version line, so "conformant" has a referent.

## Rejected alternatives

- **One semver for the whole repo.** Forces every binding to release together, or lies
  about which changed. Contradicts 0010.
- **Date-based versioning (2026.08).** Reads fine, communicates nothing about breakage —
  and breakage is the only thing a consumer needs the version to tell them.
- **Version each token file separately.** Maximum precision, unusable in practice: no
  consumer wants to track five numbers.
- **Start at 0.x.** Signals instability that is not true, and 0.x semver has no agreed
  meaning for breaking changes, so it communicates less than 1.0 does.
