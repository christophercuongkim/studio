# 0004 — Date and time selection

- **Status** Accepted
- **Date** 2026-08-04
- **Affects** every binding; `spec/DatePicker.md`; Voyage search and checkout

## Context

Voyage cannot ship without date range selection, and the kits currently fake it with a
mono text input (`2026-03-14`). Material and Cupertino both default to round cells;
SeaKim is `0px` everywhere. Time selection is a second, separable problem.

The square grid turns out to be an advantage rather than a compromise, which is the
crux of this decision.

## Decision

### The calendar is a grid, and it looks like one

Square cells, `0px`, laid out with a **1px hairline gap** so the month reads as a ruled
grid rather than floating numerals. Cell size `--control-h-touch` (44) at `sm`,
`--control-h-lg` (42) from `md` up. Dates are `--type-data` — tabular mono, so columns
of numerals align down the week.

Weekday headers use `--type-eyebrow`, uppercase, two letters.

### Range selection is where 0px wins

With square cells and hairline gaps, a selected range renders as **one continuous bar**
— no notches, no fighting to make round ends meet in the middle. Material has to invent
a pill-with-flat-inner-edges to get there; SeaKim gets it for free.

| Cell state | Treatment |
| --- | --- |
| Range endpoint | `--fill-accent` fill, `--on-accent` text |
| Inside the range | `--surface-selected` fill, `--text-accent` text |
| Hovered while picking the second endpoint | `--surface-hover`, and the provisional range previews at `--surface-selected` |
| Today | 2px `--border-accent` **bottom border only**, text unchanged |
| Unavailable | `opacity: 0.4`, not clickable. No strike-through. |
| Outside the shown month | `--text-tertiary`, still selectable |

**Today is an underline, not a fill or a ring.** Fill means selected and a ring means
focused — both are taken. An underline is unambiguous and survives being inside a
selected range, which a ring does not.

### Selection is always typeable

The calendar is an *affordance over an input*, never the only way in. Every date field
accepts typed `YYYY-MM-DD` in mono, and the calendar opens from a trailing icon button.
Two reasons: a returning traveller who knows the date is faster typing it, and a
keyboard-only or screen-reader user gets a real text field instead of a grid to arrow
through.

The popover gets `--shadow-popover` because it floats. At `sm` it becomes a bottom
sheet, per the overlay species rule.

### Time: no picker

**There is no time picker.** A time is a mono text input with a format hint, optionally
paired with a `Select` when the choices are genuinely enumerable (a pickup slot, a
kickoff window).

SeaKim already displays every time in mono (`6:40–9:15am`), so typing one is consistent
rather than a downgrade. Spinner wheels and clock dials are large, round, platform-
idiosyncratic objects that would each need a SeaKim answer, and neither app has a case
that needs one: Voyage times come from flight inventory, Bench times come from a
schedule.

Revisit if a product ever needs free time entry as its primary interaction.

## Consequences

- Two components rather than one calendar widget: the field and the popover grid. The
  field works alone, which is the point.
- Native platform pickers are not used, so each binding builds the grid. It is a
  7-column layout of square cells — the cheapest custom component in this decision set.
- Typed input needs parsing and validation per binding, including the error copy
  ("cause, consequence, next step").
- No time picker means a product with unusual time needs will hit a gap. Deliberate;
  cheaper to add later than to specify a dial nobody uses.

## Rejected alternatives

- **Round cells, matching Material and iOS.** Would be the only round non-conceptual
  shape in the system, and range bars get notchy.
- **Today as a ring or a fill.** Collides with focus and selected respectively, and
  becomes invisible inside a range.
- **Calendar-only, no typing.** Slower for known dates and worse for keyboard users.
- **Native platform date pickers.** Three different looks, none of them SeaKim, and the
  range-bar problem lands right back in the middle of the brand.
