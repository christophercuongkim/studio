# Table

Rows of records with shared columns, where the point is comparison. Per
[0003](../decisions/0003-tables.md).

**Not for:** layout (use a grid), key/value pairs for a single record (use label/value
rows), or a list you only ever read one item at a time (use `Card`s). If no column is
ever compared across rows, it is not a table.

## Anatomy

```
+---------------------------------------------------------+  1px --border-subtle
| SLOT  PLAYER          MATCHUP      LIVE v   PROJ  OWNED |  header, --type-eyebrow
+---------------------------------------------------------+  1px --border-subtle
| PG    (o) Dana Okafor  ATL vs MIA   24.6    18.4   68%  |  row
| SG    (o) Theo Lindqv  BOS @ NYK    11.2    14.8   54%  |
| C     (o) Nikola Vrba  ORL vs CHI    0.0     0.0   44%  |
+---------------------------------------------------------+
```

| Part | Required | Treatment |
| --- | --- | --- |
| Container | yes | 1px `--border-subtle`, `--radius-none`, no shadow |
| Header row | yes | `--type-eyebrow` uppercase, `--tracking-caps`, `--text-tertiary`, hairline bottom border. Sticky on vertical scroll. |
| Identifying column | yes | `--type-body-sm`, `--text-primary`. Leftmost. Never dropped. |
| Supporting column | no | `--type-body-sm`, `--text-secondary` |
| Numeric column | no | `--type-data`, tabular figures, right-aligned |
| Row action cell | no | Trailing, fixed width, revealed on row hover from `md` up |
| Footer / summary row | no | Hairline top border, `--surface-raised` fill, aligned to the columns it totals |

**Forbidden:** zebra striping, vertical rules in a record table, radius, shadow, a
second accent colour anywhere in the table.

## Density

| Density | Row height | When |
| --- | --- | --- |
| `comfortable` | `--control-h-touch` | Default. Any row containing an avatar, badge, or control. |
| `compact` | `--control-h-md` | Numeric matrices with no in-row controls. |

Cell padding is `--space-4` vertical, `--space-5` horizontal at both densities — the row
height does the work, not the padding.

## Sort

The header cell **is** the control. No separate icon button.

| State | Arrow | Label colour |
| --- | --- | --- |
| Not sorted, not hovered | absent | `--text-tertiary` |
| Not sorted, hovered | `--text-tertiary`, pointing where a click would go | `--text-secondary` |
| Active sort | `--text-accent`, `fill` weight | `--text-accent` |

One sorted column at a time. The active sort arrow is the only accent-coloured thing in
the header — treat the table as its own region for the one-accent rule.

Unsortable columns are not clickable and show no hover affordance.

## Selection

| State | Treatment |
| --- | --- |
| Hover | Background steps to `--surface-hover`. Borders unchanged. |
| Selected | `--surface-selected` fill, 2px `--border-accent` on the leading edge |
| Multi-select | Leading `Checkbox` column, `--control-h-md` wide. Header carries the indeterminate parent. |

A row is not clickable unless activating it does something specific. A whole-row target
that only selects is worse than a checkbox.

## Responsive

### Record table becomes a stacked list row at `sm`

Rows change species. They do not scroll horizontally.

```
+-----------------------------------------+
| PG  (o) Dana Okafor       [Q]      24.6 |  primary line + trailing figure
|         PG · ATL · vs MIA      proj 18.4 |  secondary line, --text-tertiary
+-----------------------------------------+  44px minimum, hairline separated
```

Each table declares three things at design time. None can be derived:

1. **Column priority.** Which columns drop first from `lg` to `md`.
2. **The surviving figure.** The one numeric column that appears at `sm`.
3. **The secondary line.** Two or three supporting values, joined by `·`.

Hover-revealed row actions become an inline trailing button, or move into the row detail
view. Nothing is reachable only by hover.

### The matrix exception

A table may scroll horizontally at `sm` **only** if every scrolling column shares a unit
— a genuine stat matrix where comparing columns is the task. The identifying column
freezes. Different units means it is a record table and the rule above applies.

## Empty, loading, error

- **Empty** — `EmptyState` inside the container border, `compact`. Name the thing and
  offer the action that creates it.
- **Filtered to nothing** — different copy from empty: name the filter, offer to clear it.
  `No players match "novak" at PG.` then `Clear filters`
- **Loading** — hold the container and header; rows become hairline-separated blocks at
  `--surface-inset`. No spinner. Never collapse the header, or the layout jumps.
- **Error** — cause, consequence, next step, inside the container.

## Accessibility

- Real table semantics: the platform table or grid role, header cells associated to their
  columns. Not a stack of divs.
- Sortable headers announce as buttons and expose the current sort direction.
- The `sm` list row is a list, not a table. Announce it as one rather than pretending.
- Row actions are focusable and become visible on focus, not only on pointer hover.
- Numeric cells announce their unit. `24.6` alone is not information.
