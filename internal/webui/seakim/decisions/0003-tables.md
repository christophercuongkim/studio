# 0003 — Tables: anatomy, sort, density, and the sm species swap

- **Status** Accepted
- **Date** 2026-08-04
- **Affects** every binding; `spec/Table.md`; Bench roster and standings

## Context

The system has no table. Bench already needs one in three places (roster, free agents,
standings) and Voyage needs one for fare comparison, so the pattern exists in the kits
as hand-rolled markup with no shared contract. That is the last stage before three
slightly different tables ship.

A table in SeaKim is unusually well-served by the existing rules: hairline borders,
`0px` corners, tabular mono figures, and density 7/10 are what a data grid wants
anyway. The genuinely open calls are the sort affordance and what happens at `sm`.

## Decision

### Anatomy

| Part | Treatment |
| --- | --- |
| Container | 1px `--border-subtle` on the outside. No shadow, no radius. |
| Header cell | `--type-eyebrow`, uppercase, `--tracking-caps`, `--text-tertiary`. Hairline bottom border. Sticky when the table scrolls vertically. |
| Body row | Hairline top border. Hover steps the background to `--surface-hover`; the border never changes. |
| Numeric cell | `--type-data` with tabular figures, right-aligned. Always. |
| Text cell | `--type-body-sm`, `--text-primary` for the identifying column, `--text-secondary` for supporting ones. |
| Row actions | Trailing cell, fixed width, revealed on row hover at `md`+. |
| Selected row | `--surface-selected` fill plus a 2px leading accent border. |
| Zebra striping | **Never.** Hairlines already separate rows; stripes plus borders is noise. |
| Vertical rules | Only in a stat *matrix* where columns are peers. Never in a record list. |

### Sort

The header cell is the control — no separate icon button. It carries an arrow that is:

- **absent** when the column is not sorted and not hovered,
- **`--text-tertiary`** on hover, pointing at the direction a click would apply,
- **`--text-accent` and `fill` weight** when the column is the active sort.

One sorted column at a time. Multi-column sort is a power feature that needs a UI of
its own; a table header cannot carry it legibly and should not try.

The active sort is the only accent-coloured thing in the header — consistent with "one
accent per screen" treating the table as its own region.

### Density

Two densities, tied to existing control heights so a control can sit in a row without
the row growing:

| Density | Row height | Use |
| --- | --- | --- |
| `comfortable` | `--control-h-touch` (44) | Default. Rows with avatars, badges, or actions. |
| `compact` | `--control-h-md` (34) | Numeric matrices with no in-row controls. |

No third density. A 28px row cannot hold a 28px control with padding, so it would force
a second set of control sizes.

### At `sm`, rows change species — they do not scroll

A record table becomes a **stacked list row**: identifying column as the primary line,
two or three supporting values as a secondary line, the most important figure trailing
in mono, hover actions becoming an inline trailing button. This is exactly what Bench's
`RosterList` already does, and it is now the rule.

Horizontal scroll is rejected for record tables because it destroys the two things a
table is for: the header (and therefore the sort affordance) scrolls out of view, and
comparison across rows requires holding an off-screen column in your head.

**One exception.** A **stat matrix** — where every column is the same kind of number and
comparison across columns *is* the task — may scroll horizontally at `sm`, with the
identifying column frozen. Cards would be worse there, because they break the grid that
makes the numbers comparable. A table qualifies as a matrix only if all scrolling
columns share a unit.

### Columns are dropped, not shrunk

Already the responsive rule, restated for tables: from `lg` to `md`, drop the widest
low-value columns. Never reduce type below 13px, never truncate a figure, never wrap a
numeric cell.

## Consequences

- Every table needs an author decision about column priority (which drop first) and
  which single figure survives to `sm`. That is real design work per table and cannot be
  automated. It is also the work that makes the table usable on a phone.
- The `sm` list row and the `md`+ table row are two layouts, so a binding implements
  both. Bench proves this is roughly 60 lines, not a rewrite.
- The matrix exception will be argued about. Requiring a shared unit across scrolling
  columns is the test that keeps it narrow.

## Rejected alternatives

- **Horizontal scroll everywhere.** Simpler to build, and the resulting table cannot be
  sorted or compared on the device most people use.
- **Three densities.** Breaks the "a control fits in a row" property.
- **Zebra striping instead of row borders.** Fights the hairline rule, and stripes carry
  a second meaning in dark mode where the step has to be large enough to see.
- **A dedicated sort icon button per header.** Doubles the header height or halves the
  label space, to make a click target out of something that is already one.
