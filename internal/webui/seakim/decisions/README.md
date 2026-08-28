# Decisions

Numbered, dated records of calls that were **contested, expensive, or reversible**.
Not every decision in SeaKim needs one — the readme already states dozens of rules
that nobody argued about. An ADR earns its place when a future reader would
reasonably ask "why on earth is it like this?" or would otherwise re-litigate it.

## Status values

| Status | Meaning |
| --- | --- |
| `Proposed` | Written up, awaiting sign-off. Do not build against it yet. |
| `Accepted` | In force. Bindings must comply. |
| `Superseded by NNNN` | Kept for the record. Never deleted, never edited to agree with the new answer. |

Fourteen are in force as of 2026-08-05; 0015 and 0016 are proposed and must not be built against yet. ADRs are **append-only**. Correct one by writing its successor, not by rewriting it —
the value is in the trail, and a silently edited ADR is worse than none.

## Index

| # | Decision | Status |
| --- | --- | --- |
| [0001](0001-platform-neutral-spec-layer.md) | Component opinions live in `spec/`, bindings hold only usage docs | Accepted |
| [0002](0002-no-floating-action-button.md) | No floating action button | Accepted |
| [0003](0003-tables.md) | Tables: anatomy, sort, density, and the `sm` species swap | Accepted |
| [0004](0004-date-and-time-selection.md) | Date and time selection | Accepted |
| [0005](0005-light-mode-is-first-class.md) | Light mode is first-class, and enforced | Accepted |
| [0006](0006-slider.md) | Slider anatomy | Accepted |
| [0007](0007-token-source-format.md) | DTCG JSON becomes the token source; CSS becomes an output | Accepted |
| [0008](0008-conformance-tiers.md) | What a binding must do to call itself SeaKim | Accepted |
| [0009](0009-bundle-phosphor-icon-font.md) | Bundle the Phosphor icon font; drop `phosphor_flutter` | Accepted |
| [0010](0010-bindings-are-contributed-not-owned.md) | Bindings are contributed, not owned | Accepted |
| [0011](0011-versioning.md) | Versioning | Accepted |
| [0012](0012-conformance-checks-ship-with-rules.md) | Conformance checks ship with the rules; consuming repos run them | Accepted |
| [0013](0013-alpha-variants-are-tokens.md) | Alpha variants are tokens, not per-component constants | Accepted |
| [0014](0014-text-selection-tier-1.md) | Text selection is Tier 1, and when a Tier 1 divergence needs an ADR | Accepted |
| [0015](0015-sequential-chart-ramp.md) | A sequential (magnitude) chart ramp | Accepted |
| [0016](0016-many-series-trajectories.md) | Many-series trajectories: the >6 change-over-time case | Accepted |
| [0017](0017-distribution-interval-glyph.md) | A distribution/interval glyph (`Range`) | Accepted |
| [0018](0018-raised-shadow-direction.md) | Raised shadow casts away from the anchored edge | Accepted |
| [0019](0019-versioning-second-pass.md) | Versioning, second pass: token revalues and honest binding versions | Accepted |
| [0020](0020-preview-surfaces-are-gated.md) | Preview surfaces are deliverables, and a render gate proves it | Accepted |
| [0021](0021-loading-states.md) | Loading is a skeleton or a labeled fallback, never a bare spinner | Accepted |
| [0022](0022-anchored-popover-species.md) | Anchored contextual popover is a third overlay species | Accepted |
| [0023](0023-sub-floor-chrome-hit-area.md) | A control may render below the touch floor only if its hit area is not | Accepted |
| [0024](0024-avatar-receipts-and-stack.md) | Avatar as read-receipt, and an avatar stack primitive | Accepted |
| [0025](0025-progressive-disclosure-metadata.md) | Metadata on demand, with a non-pointer path | Accepted |
| [0026](0026-accent-as-ownership-fill.md) | The single-accent rule governs competing actions, not identity washes | Accepted |
| [0027](0027-non-primary-floating-affordance.md) | A non-primary floating affordance is allowed, and is not a FAB | Accepted |
| [0028](0028-searchable-select.md) | A searchable long-list picker is its own component (SkCombobox), not a hand-rolled trigger | Accepted |
| [0029](0029-error-state.md) | The error state is a first-class treatment, not an SkEmptyState variant | Accepted |

## Writing one

Five headings, and stop. Context is the part people skip and the part that matters in
two years — write down the constraint you were under, not just the answer.

```md
# NNNN — Title in plain words

- **Status** Proposed | Accepted | Superseded by NNNN
- **Date** YYYY-MM-DD
- **Affects** which bindings, which specs

## Context
What was true, and what forced a call.

## Decision
The rule, stated so a binding author can obey it without asking.

## Consequences
What this costs. Name the thing it makes harder.

## Rejected alternatives
What else was on the table and why it lost. One line each.
```
