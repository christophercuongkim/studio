# DatePicker

A date field, and the calendar grid that opens from it. Per
[0004](../decisions/0004-date-and-time-selection.md).

**Not for:** time selection — there is no time picker; a time is a mono `Input`, or a
`Select` when the choices are enumerable. Not for month or year alone either; that is a
`Select`.

## Two parts, and the field works alone

The calendar is an affordance over an input, never the only way in. Every date field
accepts typed `YYYY-MM-DD` in `--type-data`, with a trailing `IconButton` that opens the
grid. A returning traveller who knows the date is faster typing it, and a keyboard or
screen-reader user gets a real text field instead of a grid to arrow through.

### Field

Standard `Input` with `mono`, inside a `Field`. Trailing `IconButton` (`calendar-blank`,
label "Choose date"). Invalid input follows the normal `Field` error path: cause,
consequence, next step.

## Grid anatomy

```
+-------------------------------------------+
|  <   March 2026                        >  |  month bar, --type-subheading
+-------------------------------------------+
| MO   TU   WE   TH   FR   SA   SU          |  --type-eyebrow, 2 letters
+----+----+----+----+----+----+----+
|  2 |  3 |  4 |  5 |  6 |  7 |  8 |          square cells, 1px hairline gap
+----+----+----+----+----+----+----+
|  9 | 10 | 11 | 12 | 13 | 14 | 15 |
+----+----+----+----+----+----+----+
```

| Part | Treatment |
| --- | --- |
| Container | `--surface-overlay`, 1px `--border-default`, `--shadow-popover` — it floats |
| Month bar | `--type-subheading`, prev/next as `IconButton`s |
| Weekday header | `--type-eyebrow`, uppercase, two letters, `--text-tertiary` |
| Cell | Square. `--control-h-touch` at `sm`, `--control-h-lg` from `md`. `--type-data` |
| Grid gap | 1px, `--border-subtle` showing through, so the month reads as a ruled grid |

Dates are tabular mono so numerals align down each week column.

## Cell states

| State | Treatment |
| --- | --- |
| Default | `--surface-raised`, `--text-primary` |
| Hover | `--surface-hover` |
| Today | 2px `--border-accent` **bottom border only**. Text unchanged. |
| Selected (single) | `--fill-accent` fill, `--on-accent` text |
| Range endpoint | `--fill-accent` fill, `--on-accent` text |
| Inside range | `--surface-selected` fill, `--text-accent` text |
| Range preview | While picking the second endpoint, the provisional span shows at `--surface-selected` |
| Unavailable | `opacity: 0.4`, not selectable. No strike-through. |
| Outside shown month | `--text-tertiary`, still selectable |
| Focused | Standard `--focus-ring` on the cell |

**Today is an underline, not a fill or a ring.** Fill means selected; a ring means
focused. Both are taken. An underline is unambiguous and survives being inside a selected
range, which a ring does not.

## Range selection

Square cells with hairline gaps make a range render as **one continuous bar** — no
notches, no rounded ends meeting awkwardly in the middle. This is a place where `0px`
is better than the platform default rather than a constraint to work around.

- First click sets the start and clears any previous range.
- Hovering previews the span; the field shows the provisional range.
- Second click before the start reorders rather than rejecting.
- A single-day range is legal: both endpoints on one cell.
- Two fields (depart, return) may drive one grid. The grid highlights whichever field has
  focus and advances to the other on selection.

## Responsive

| Breakpoint | Presentation |
| --- | --- |
| `sm` | Bottom sheet, per the overlay species rule. Cells at `--control-h-touch`. One month, vertically scrollable to reach the next. |
| `md`, `lg` | Popover anchored to the field. Two months side by side for range selection, one for single. |

## Accessibility

- The field is the primary control and is always usable without opening the grid.
- Grid uses the platform grid role. Arrow keys move by day, Page Up/Down by month, Home
  and End to week bounds.
- Each cell announces its full date, not just the number, plus its state (selected, range
  start, unavailable, today).
- Opening the grid moves focus into it; closing returns focus to the trigger.
- Escape closes without selecting.
- Unavailable dates announce *why* when a reason exists ("no flights on this date").
